package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerSchema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceSchema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &CleanEksProvider{}
var _ provider.ProviderWithFunctions = &CleanEksProvider{}

type CleanEksProvider struct {
	Host string

	BurstLimit int64
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type CleanEksProviderModel struct {
	Host                  types.String `tfsdk:"host"`
	Username              types.String `tfsdk:"username"`
	Password              types.String `tfsdk:"password"`
	Insecure              types.Bool   `tfsdk:"insecure"`
	TlsServerName         types.String `tfsdk:"tls_server_name"`
	ClientCertificate     types.String `tfsdk:"client_certificate"`
	ClientKey             types.String `tfsdk:"client_key"`
	ClusterCaCertificate  types.String `tfsdk:"cluster_ca_certificate"`
	ConfigPaths           types.List   `tfsdk:"config_paths"`
	ConfigPath            types.String `tfsdk:"config_path"`
	ConfigContext         types.String `tfsdk:"config_context"`
	ConfigContextAuthInfo types.String `tfsdk:"config_context_auth_info"`
	ConfigContextCluster  types.String `tfsdk:"config_context_cluster"`
	Token                 types.String `tfsdk:"token"`
	ProxyUrl              types.String `tfsdk:"proxy_url"`
	//exec
	BurstLimit types.Int64 `tfsdk:"burst_limit"`
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
	resp.Schema = providerSchema.Schema{
		Description: "A provider to bootstrap an EKS cluster by removing AWS CNI and Kube-Proxy. It will also add the required annotations and labels to CoreDNS so that Helm can manage CoreDNS. It will also drop managed by AWS labels from CoreDNS deployment and service.",
		Attributes: map[string]providerSchema.Attribute{
			"host": resourceSchema.StringAttribute{
				Description: "The hostname (in form of URI) of Kubernetes master.",
				Optional:    true,
				Computed:    true,
				Default:     EnvDefaultString("KUBE_HOST", ""),
			},

			"username": resourceSchema.StringAttribute{
				Description: "The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
				Optional:    true,
				Computed:    true,
				Default:     EnvDefaultString("KUBE_USER", ""),
			},

			"password": resourceSchema.StringAttribute{
				Description: "The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				Default:     EnvDefaultString("KUBE_PASSWORD", ""),
			},

			"insecure": resourceSchema.BoolAttribute{
				Description: "Whether server should be accessed without verifying the TLS certificate.",
				Optional:    true,
				Computed:    true,
				Default:     EnvDefaultBool("KUBE_INSECURE", false),
			},

			"tls_server_name": resourceSchema.StringAttribute{
				Description: "Server name passed to the server for SNI and is used in the client to check server certificates against.",
				Optional:    true,
				Computed:    true,
				Default:     EnvDefaultString("KUBE_TLS_SERVER_NAME", ""),
			},

			"client_certificate": resourceSchema.StringAttribute{
				Description: "PEM-encoded client certificate for TLS authentication.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("token")),
				},
				Default: EnvDefaultString("KUBE_CLIENT_CERT_DATA", ""),
			},

			"client_key": resourceSchema.StringAttribute{
				Description: "PEM-encoded client certificate key for TLS authentication.",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("token")),
				},
				Default: EnvDefaultString("KUBE_CLIENT_KEY_DATA", ""),
			},

			"cluster_ca_certificate": resourceSchema.StringAttribute{
				Description: "PEM-encoded root certificates bundle for TLS authentication.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("insecure")),
				},
				Default: EnvDefaultString("KUBE_CLUSTER_CA_CERT_DATA", ""),
			},

			"config_paths": resourceSchema.ListAttribute{
				Description: "A list of paths to kube config files. Can be set with KUBE_CONFIG_PATHS environment variable.",
				Optional:    true,
				Computed:    true,
				ElementType: types.ListType{
					ElemType: types.StringType,
				},
			},

			"config_path": resourceSchema.StringAttribute{
				Description: "Path to the kube config file. Can be set with KUBE_CONFIG_PATH.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("config_paths")),
				},
				Default: EnvDefaultString("KUBE_CONFIG_PATH", ""),
			},

			"config_context": resourceSchema.StringAttribute{
				Description: "",
				Optional:    true,
				Computed:    true,
				Default:     EnvDefaultString("KUBE_CTX", ""),
			},

			"config_context_auth_info": resourceSchema.StringAttribute{
				Description: "",
				Optional:    true,
				Computed:    true,
				Default:     EnvDefaultString("KUBE_CTX_AUTH_INFO", ""),
			},

			"config_context_cluster": resourceSchema.StringAttribute{
				Description: "",
				Optional:    true,
				Computed:    true,
				Default:     EnvDefaultString("KUBE_CTX_CLUSTER", ""),
			},

			"token": resourceSchema.StringAttribute{
				Description: "Token to authenticate an service account",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("client_cert_pem")),
				},
				Default: EnvDefaultString("KUBE_TOKEN", ""),
			},

			"proxy_url": resourceSchema.StringAttribute{
				Description: "URL to the proxy to be used for all API requests",
				Optional:    true,
				Computed:    true,
				Default:     EnvDefaultString("KUBE_PROXY_URL", ""),
			},

			"exec": resourceSchema.ListNestedAttribute{
				Description: "A list of commands to execute.",
				Optional:    true,
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: resourceSchema.NestedAttributeObject{
					Attributes: map[string]resourceSchema.Attribute{
						"api_version": resourceSchema.StringAttribute{
							Description: "The client authentication api version to use",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("client.authentication.k8s.io/v1"),
						},
						"env": resourceSchema.MapAttribute{
							Description: "Environment variables to set for the command",
							Optional:    true,
							ElementType: types.ListType{
								ElemType: types.StringType,
							},
						},
						"command": resourceSchema.StringAttribute{
							Description: "Command to execute",
							Required:    true,
						},
						"args": resourceSchema.MapAttribute{
							Description: "Arguments to pass to the command",
							Optional:    true,
							ElementType: types.ListType{
								ElemType: types.StringType,
							},
						},
					},
				},
			},

			"burst_limit": resourceSchema.Int64Attribute{
				Description: "Helm burst limit. Increase this if you have a cluster with many CRDs",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(100),
				Validators: []validator.Int64{
					int64validator.AtMost(100000),
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

	host := ""
	if !(model.Host.IsNull() || model.Host.IsUnknown()) {
		host = model.Host.ValueString()
	}

	username := ""
	if !(model.Username.IsNull() || model.Username.IsUnknown()) {
		username = model.Username.ValueString()
	}

	password := ""
	if !(model.Password.IsNull() || model.Password.IsUnknown()) {
		password = model.Password.ValueString()
	}

	insecure := false
	if !(model.Insecure.IsNull() || model.Insecure.IsUnknown()) {
		insecure = model.Insecure.ValueBool()
	}

	tlsServerName := ""
	if !(model.TlsServerName.IsNull() || model.TlsServerName.IsUnknown()) {
		tlsServerName = model.TlsServerName.ValueString()
	}

	clientCertificate := ""
	if !(model.ClientCertificate.IsNull() || model.ClientCertificate.IsUnknown()) {
		clientCertificate = model.ClientCertificate.ValueString()
	}

	clientKey := ""
	if !(model.ClientKey.IsNull() || model.ClientKey.IsUnknown()) {
		clientKey = model.ClientKey.ValueString()
	}

	clusterCaCertificate := ""
	if !(model.ClusterCaCertificate.IsNull() || model.ClusterCaCertificate.IsUnknown()) {
		clusterCaCertificate = model.ClusterCaCertificate.ValueString()
	}

	configPaths := make([]string, 0)
	if !(model.ConfigPath.IsNull() || model.ConfigPath.IsUnknown()) {
		configPaths = []string{model.ConfigPath.ValueString()}
	} else if !(model.ConfigPaths.IsNull() || model.ConfigPaths.IsUnknown()) {
		configPaths = StringListToStrings(model.ConfigPaths)
	} else if v := os.Getenv("KUBE_CONFIG_PATHS"); v != "" {
		// NOTE we have to do this here because the schema
		// does not yet allow you to set a default for a TypeList
		configPaths = filepath.SplitList(v)
	}

	configContext := ""
	if !(model.ConfigContext.IsNull() || model.ConfigContext.IsUnknown()) {
		configContext = model.ConfigContext.ValueString()
	}

	configContextAuthInfo := ""
	if !(model.ConfigContextAuthInfo.IsNull() || model.ConfigContextAuthInfo.IsUnknown()) {
		configContextAuthInfo = model.ConfigContextAuthInfo.ValueString()
	}

	configContextCluster := ""
	if !(model.ConfigContextCluster.IsNull() || model.ConfigContextCluster.IsUnknown()) {
		configContextCluster = model.ConfigContextCluster.ValueString()
	}

	token := ""
	if !(model.Token.IsNull() || model.Token.IsUnknown()) {
		token = model.Token.ValueString()
	}

	proxyUrl := ""
	if !(model.ProxyUrl.IsNull() || model.ProxyUrl.IsUnknown()) {
		proxyUrl = model.ProxyUrl.ValueString()
	}

	//exec

	burstLimit := int64(0)
	if !(model.BurstLimit.IsNull() || model.BurstLimit.IsUnknown()) {
		burstLimit = model.BurstLimit.ValueInt64()
	}

	////////////////////

	if (clientCertificate != "" && clientKey == "") || (clientCertificate == "" && clientKey != "") {
		resp.Diagnostics.AddError(
			"Both Client Certificate and Client Key must be specified for Kubernetes Cluster",
			fmt.Sprintf("Both Client Certificate and Client Key must be specified for Kubernetes Cluster"),
		)
		return
	}

	if token == "" && clientCertificate == "" {
		resp.Diagnostics.AddError(
			"Token or Client Certificate for Kubernetes Cluster must be specified",
			fmt.Sprintf("Token or Client Certificate for Kubernetes Cluster must be specified"),
		)
		return
	}

	////////////////////////////////////////////////

	namespace := ""

	p.Host = host
	p.BurstLimit = burstLimit

	ok, kubeConfig := getKubeConfig(ctx, resp, configPaths, configContext, configContextAuthInfo, configContextCluster, insecure, tlsServerName, clusterCaCertificate, clientCertificate, host, username, password, clientKey, token, proxyUrl, namespace, burstLimit)
	if !ok {
		return
	}
	restConfig, err := kubeConfig.ToRESTConfig()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to convert KubeConfig to rest config",
			fmt.Sprintf("Unable to convert KubeConfig to rest config: %s", err),
		)
		return
	}

	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Kubernetes client",
			fmt.Sprintf("Error creating Kubernetes client: %s", err),
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
