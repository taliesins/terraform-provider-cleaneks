package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"k8s.io/client-go/kubernetes"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &CleanEksProvider{}
var _ provider.ProviderWithFunctions = &CleanEksProvider{}

type CleanEksProvider struct {
	Insecure          bool
	CaCertificate     string
	ClientCertificate string
	ClientKey         string
	Token             string
	RequestTimeout    int64
	Host              string

	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type CleanEksProviderModel struct {
	Host              types.String `tfsdk:"host"`
	Insecure          types.Bool   `tfsdk:"insecure"`
	CaCertificate     types.String `tfsdk:"cluster_ca_certificate"`
	ClientCertificate types.String `tfsdk:"client_cert_pem"`
	ClientKey         types.String `tfsdk:"client_key_pem"`
	Token             types.String `tfsdk:"token"`
	RequestTimeout    types.Int64  `tfsdk:"request_timeout_ms"`
}

type CleanEksProviderDataSourceData struct {
	Config    *CleanEksProvider
	ClientSet *kubernetes.Clientset
}

type CleanEksProviderResourceData struct {
	Config    *CleanEksProvider
	ClientSet *kubernetes.Clientset
}

func (p *CleanEksProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cleaneks"
	resp.Version = p.version
}

func (p *CleanEksProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A provider to bootstrap an EKS cluster by removing AWS CNI and Kube-Proxy. It will also add the required annotations and labels to CoreDNS so that Helm can manage CoreDNS. It will also drop managed by AWS labels from CoreDNS deployment and service.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "The Kubernetes endpoint. Supported schemes are `http` and `https`.",
				Required:    true,
			},

			"insecure": schema.BoolAttribute{
				Description: "Disables verification of the server's certificate chain and hostname. Defaults to `false`",
				Optional:    true,
			},

			"cluster_ca_certificate": schema.StringAttribute{
				Description: "Certificate data of the Certificate Authority (CA) " +
					"in [PEM (RFC 1421)](https://datatracker.ietf.org/doc/html/rfc1421) format.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("insecure")),
				},
			},

			"client_cert_pem": schema.StringAttribute{
				Description: "Client Certificate (PEM) to present to the target server." +
					"in [PEM (RFC 1421)](https://datatracker.ietf.org/doc/html/rfc1421) format.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("token")),
				},
			},

			"client_key_pem": schema.StringAttribute{
				Description: "Client Certificate (PEM) private Key to use for mTLS.",
				Optional:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("token")),
				},
			},

			"token": schema.StringAttribute{
				Description: "The Kubernetes token.",
				Optional:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("client_cert_pem")),
				},
			},

			"request_timeout_ms": schema.Int64Attribute{
				Description: "The request timeout in milliseconds.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(10),
				},
			},
		},
	}
}

func (p *CleanEksProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Debug(ctx, "Configuring provider")
	p.resetConfig()

	// Load configuration into the model
	var model CleanEksProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	insecure := false
	if !(model.Insecure.IsNull() || model.Insecure.IsUnknown()) {
		insecure = model.Insecure.ValueBool()
	}

	caCertificate := ""
	if !(model.CaCertificate.IsNull() || model.CaCertificate.IsUnknown()) {
		caCertificate = model.CaCertificate.ValueString()
	}

	clientCertificate := ""
	if !(model.ClientCertificate.IsNull() || model.ClientCertificate.IsUnknown()) {
		clientCertificate = model.ClientCertificate.ValueString()
	}

	clientKey := ""
	if !(model.ClientKey.IsNull() || model.ClientKey.IsUnknown()) {
		clientKey = model.ClientKey.ValueString()
	}

	if (clientCertificate != "" && clientKey == "") || (clientCertificate == "" && clientKey != "") {
		resp.Diagnostics.AddError(
			"Both Client Certificate and Client Key must be specified for Kubernetes Cluster",
			fmt.Sprintf("Both Client Certificate and Client Key must be specified for Kubernetes Cluster"),
		)
		return
	}

	token := ""
	if !(model.Token.IsNull() || model.Token.IsUnknown()) {
		token = model.Token.ValueString()
	}
	if token == "" && clientCertificate == "" {
		resp.Diagnostics.AddError(
			"Token or Client Certificate for Kubernetes Cluster must be specified",
			fmt.Sprintf("Token or Client Certificate for Kubernetes Cluster must be specified"),
		)
		return
	}

	requestTimeout := int64(0)
	if !(model.RequestTimeout.IsNull() || model.RequestTimeout.IsUnknown()) {
		requestTimeout = model.RequestTimeout.ValueInt64()
	}

	host := model.Host.ValueString()

	p.Insecure = insecure
	p.CaCertificate = caCertificate
	p.ClientCertificate = clientCertificate
	p.ClientKey = clientKey
	p.Token = token
	p.RequestTimeout = requestTimeout
	p.Host = host

	clientSet, err := p.GetClient()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating clientSet",
			fmt.Sprintf("Error creating clientSet: %s", err),
		)
		return
	}

	// Since the provider instance is being passed, ensure these response
	// values are always set before early returns, etc.
	resp.DataSourceData = &CleanEksProviderDataSourceData{
		ClientSet: clientSet,
		Config:    p,
	}
	resp.ResourceData = &CleanEksProviderResourceData{
		ClientSet: clientSet,
		Config:    p,
	}

	tflog.Debug(ctx, "Provider configuration", map[string]interface{}{
		"provider": fmt.Sprintf("%+v", p),
	})
}

func (p *CleanEksProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewJobResource,
	}
}

func (p *CleanEksProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *CleanEksProvider) Functions(context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CleanEksProvider{
			version: version,
		}
	}
}

func (p *CleanEksProvider) resetConfig() {

}
