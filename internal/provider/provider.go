package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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
	"k8s.io/client-go/kubernetes"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &CleanEksProvider{}
var _ provider.ProviderWithFunctions = &CleanEksProvider{}

type CleanEksProvider struct {
	Host string

	BurstLimit int64
	// Version is set to the provider Version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	Version string

	clientSet *kubernetes.Clientset
	model     CleanEksProviderModel
}

type CleanEksProviderModel struct {
	Host                  types.String   `tfsdk:"host"`
	Username              types.String   `tfsdk:"username"`
	Password              types.String   `tfsdk:"password"`
	Insecure              types.Bool     `tfsdk:"insecure"`
	TLSServerName         types.String   `tfsdk:"tls_server_name"`
	ClientCertificate     types.String   `tfsdk:"client_certificate"`
	ClientKey             types.String   `tfsdk:"client_key"`
	ClusterCACertificate  types.String   `tfsdk:"cluster_ca_certificate"`
	ConfigPaths           []types.String `tfsdk:"config_paths"`
	ConfigPath            types.String   `tfsdk:"config_path"`
	ConfigContext         types.String   `tfsdk:"config_context"`
	ConfigContextAuthInfo types.String   `tfsdk:"config_context_auth_info"`
	ConfigContextCluster  types.String   `tfsdk:"config_context_cluster"`
	Token                 types.String   `tfsdk:"token"`
	ProxyURL              types.String   `tfsdk:"proxy_url"`
	Exec                  []struct {
		APIVersion types.String            `tfsdk:"api_version"`
		Command    types.String            `tfsdk:"command"`
		Env        map[string]types.String `tfsdk:"env"`
		Args       []types.String          `tfsdk:"args"`
	} `tfsdk:"exec"`
	BurstLimit types.Int64 `tfsdk:"burst_limit"`
}

func (p *CleanEksProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cleaneks"
	resp.Version = p.Version
}

func (p *CleanEksProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerSchema.Schema{
		MarkdownDescription: "A provider to bootstrap an EKS cluster by removing **AWS CNI** and **Kube-Proxy**. It will also add the required annotations and labels to import `CoreDNS` into Helm. It will also drop managed by AWS labels from CoreDNS deployment and service.",
		Description:         "A provider to bootstrap an EKS cluster by removing AWS CNI and Kube-Proxy. It will also add the required annotations and labels to CoreDNS so that Helm can manage CoreDNS. It will also drop managed by AWS labels from CoreDNS deployment and service.",
		Attributes: map[string]providerSchema.Attribute{
			"host": resourceSchema.StringAttribute{
				MarkdownDescription: "The hostname (in form of URI) of Kubernetes master. Can be set with `KUBE_HOST` environment variable.",
				Description:         "The hostname (in form of URI) of Kubernetes master. Can be set with KUBE_HOST environment variable.",
				Optional:            true,
				Computed:            true,
				Default:             EnvDefaultString("KUBE_HOST", ""),
			},

			"username": resourceSchema.StringAttribute{
				MarkdownDescription: "The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint. Can be set with `KUBE_USER` environment variable.",
				Description:         "The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint. Can be set with KUBE_USER environment variable.",
				Optional:            true,
				Computed:            true,
				Default:             EnvDefaultString("KUBE_USER", ""),
			},

			"password": resourceSchema.StringAttribute{
				MarkdownDescription: "The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint. Can be set with `KUBE_PASSWORD` environment variable.",
				Description:         "The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint. Can be set with KUBE_PASSWORD environment variable.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Default:             EnvDefaultString("KUBE_PASSWORD", ""),
			},

			"insecure": resourceSchema.BoolAttribute{
				MarkdownDescription: "Whether server should be accessed without verifying the TLS certificate. Can be set with `KUBE_INSECURE` environment variable.",
				Description:         "Whether server should be accessed without verifying the TLS certificate. Can be set with KUBE_INSECURE environment variable.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Bool{
					boolvalidator.ConflictsWith(path.MatchRoot("cluster_ca_certificate")),
				},
				Default: EnvDefaultBool("KUBE_INSECURE", false),
			},

			"tls_server_name": resourceSchema.StringAttribute{
				MarkdownDescription: "Server name passed to the server for SNI and is used in the client to check server certificates against. Can be set with `KUBE_TLS_SERVER_NAME` environment variable.",
				Description:         "Server name passed to the server for SNI and is used in the client to check server certificates against. Can be set with KUBE_TLS_SERVER_NAME environment variable.",
				Optional:            true,
				Computed:            true,
				Default:             EnvDefaultString("KUBE_TLS_SERVER_NAME", ""),
			},

			"client_certificate": resourceSchema.StringAttribute{
				MarkdownDescription: "PEM-encoded client certificate for TLS authentication. Can be set with `KUBE_CLIENT_CERT_DATA` environment variable.",
				Description:         "PEM-encoded client certificate for TLS authentication. Can be set with KUBE_CLIENT_CERT_DATA environment variable.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("token")),
				},
				Default: EnvDefaultString("KUBE_CLIENT_CERT_DATA", ""),
			},

			"client_key": resourceSchema.StringAttribute{
				MarkdownDescription: "PEM-encoded client certificate key for TLS authentication. Can be set with `KUBE_CLIENT_KEY_DATA` environment variable.",
				Description:         "PEM-encoded client certificate key for TLS authentication. Can be set with KUBE_CLIENT_KEY_DATA environment variable.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("token")),
				},
				Default: EnvDefaultString("KUBE_CLIENT_KEY_DATA", ""),
			},

			"cluster_ca_certificate": resourceSchema.StringAttribute{
				MarkdownDescription: "PEM-encoded root certificates bundle for TLS authentication. Can be set with `KUBE_CLUSTER_CA_CERT_DATA` environment variable.",
				Description:         "PEM-encoded root certificates bundle for TLS authentication. Can be set with KUBE_CLUSTER_CA_CERT_DATA environment variable.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("insecure")),
				},
				Default: EnvDefaultString("KUBE_CLUSTER_CA_CERT_DATA", ""),
			},

			"config_paths": resourceSchema.ListAttribute{
				MarkdownDescription: "A list of paths to kube config files. Can be set with `KUBE_CONFIG_PATHS` environment variable.",
				Description:         "A list of paths to kube config files. Can be set with KUBE_CONFIG_PATHS environment variable.",
				Optional:            true,
				ElementType:         types.StringType,
			},

			"config_path": resourceSchema.StringAttribute{
				MarkdownDescription: "Path to the kube config file. Can be set with `KUBE_CONFIG_PATH`.",
				Description:         "Path to the kube config file. Can be set with KUBE_CONFIG_PATH.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("config_paths")),
				},
				Default: EnvDefaultString("KUBE_CONFIG_PATH", ""),
			},

			"config_context": resourceSchema.StringAttribute{
				MarkdownDescription: "Select the Kube context to use. Can be set with `KUBE_CTX` environment variable.",
				Description:         "Select the Kube context to use. Can be set with KUBE_CTX environment variable.",
				Optional:            true,
				Computed:            true,
				Default:             EnvDefaultString("KUBE_CTX", ""),
			},

			"config_context_auth_info": resourceSchema.StringAttribute{
				MarkdownDescription: "Select the Kube authentication context to use. Can be set with `KUBE_CTX_AUTH_INFO` environment variable.",
				Description:         "Select the Kube authentication context to use. Can be set with KUBE_CTX_AUTH_INFO environment variable.",
				Optional:            true,
				Computed:            true,
				Default:             EnvDefaultString("KUBE_CTX_AUTH_INFO", ""),
			},

			"config_context_cluster": resourceSchema.StringAttribute{
				MarkdownDescription: "Select the Kube cluster context to use. Can be set with `KUBE_CTX_CLUSTER` environment variable.",
				Description:         "Select the Kube cluster context to use. Can be set with KUBE_CTX_CLUSTER environment variable.",
				Optional:            true,
				Computed:            true,
				Default:             EnvDefaultString("KUBE_CTX_CLUSTER", ""),
			},

			"token": resourceSchema.StringAttribute{
				MarkdownDescription: "Token to authenticate an service account. Can be set with `KUBE_TOKEN` environment variable.",
				Description:         "Token to authenticate an service account. Can be set with KUBE_TOKEN environment variable.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("client_certificate")),
				},
				Default: EnvDefaultString("KUBE_TOKEN", ""),
			},

			"proxy_url": resourceSchema.StringAttribute{
				MarkdownDescription: "URL to the proxy to be used for all API requests. Can be set with `KUBE_PROXY_URL` environment variable.",
				Description:         "URL to the proxy to be used for all API requests. Can be set with KUBE_PROXY_URL environment variable.",
				Optional:            true,
				Computed:            true,
				Default:             EnvDefaultString("KUBE_PROXY_URL", ""),
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
		Blocks: map[string]providerSchema.Block{
			"exec": providerSchema.ListNestedBlock{
				NestedObject: providerSchema.NestedBlockObject{
					Attributes: map[string]providerSchema.Attribute{
						"api_version": resourceSchema.StringAttribute{
							Description: "The client authentication api Version to use",
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
						"args": resourceSchema.ListAttribute{
							Description: "Arguments to pass to the command",
							Optional:    true,
							ElementType: types.ListType{
								ElemType: types.StringType,
							},
						},
					},
				},
			},
		},
	}
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
			Version: version,
		}
	}
}

func (p *CleanEksProvider) GetClientSet(ctx context.Context) (*kubernetes.Clientset, error) {
	if p.clientSet != nil {
		return p.clientSet, nil
	}

	var clientSet *kubernetes.Clientset
	restConfig, err := newKubernetesClientConfig(ctx, p.model)
	if err != nil {
		return nil, err
	} else {
		clientSet, err = kubernetes.NewForConfig(restConfig)
		if err != nil {
			return nil, err
		}
	}
	return clientSet, nil
}
