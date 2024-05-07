package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type cleanEksProvider struct {
	Insecure          bool
	CaCertificate     string
	ClientCertificate string
	ClientKey         string
	Token             string
	RequestTimeout    int64
	Endpoint          string
}

var _ provider.Provider = (*cleanEksProvider)(nil)

func New() provider.Provider {
	return &cleanEksProvider{}
}

func (p *cleanEksProvider) resetConfig() {

}

func (p *cleanEksProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cleaneks"
}

func (p *cleanEksProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A provider to bootstrap an EKS cluster by removing AWS CNI and Kube-Proxy. It will also add the required annotations and labels to CoreDNS so that Helm can manage CoreDNS. It will also drop managed by AWS labels from CoreDNS deployment and service.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Description: "The Kubernetes endpoint. Supported schemes are `http` and `https`.",
				Required:    true,
			},

			"insecure": schema.BoolAttribute{
				Description: "Disables verification of the server's certificate chain and hostname. Defaults to `false`",
				Optional:    true,
			},

			"ca_cert_pem": schema.StringAttribute{
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

func (p *cleanEksProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Debug(ctx, "Configuring provider")
	p.resetConfig()

	// Since the provider instance is being passed, ensure these response
	// values are always set before early returns, etc.
	resp.DataSourceData = p
	resp.ResourceData = p

	// Load configuration into the model
	var model providerConfigModel
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

	endpoint := model.Endpoint.ValueString()

	p.Insecure = insecure
	p.CaCertificate = caCertificate
	p.ClientCertificate = clientCertificate
	p.ClientKey = clientKey
	p.Token = token
	p.RequestTimeout = requestTimeout
	p.Endpoint = endpoint

	tflog.Debug(ctx, "Provider configuration", map[string]interface{}{
		"provider": fmt.Sprintf("%+v", p),
	})
}

func (p *cleanEksProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewJobResource,
	}
}

func (p *cleanEksProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
