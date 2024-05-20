package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &JobResource{}
var _ resource.ResourceWithImportState = &JobResource{}

func NewJobResource() resource.Resource {
	return &JobResource{}
}

type JobResource struct {
	provider *CleanEksProvider
}

type JobResourceModel struct {
	ID types.String `tfsdk:"id"`

	RemoveAwsCni        types.Bool `tfsdk:"remove_aws_cni"`
	RemoveKubeProxy     types.Bool `tfsdk:"remove_kube_proxy"`
	RemoveCoreDns       types.Bool `tfsdk:"remove_core_dns"`
	ImportCorednsToHelm types.Bool `tfsdk:"import_coredns_to_helm"`

	AwsCniDaemonsetExists    types.Bool `tfsdk:"aws_cni_daemonset_exists"`
	KubeProxyDaemonsetExists types.Bool `tfsdk:"kube_proxy_daemonset_exists"`
	KubeProxyConfigMapExists types.Bool `tfsdk:"kube_proxy_config_map_exists"`

	AwsCoreDnsDeploymentExists          types.Bool `tfsdk:"aws_coredns_deployment_exists"`
	AwsCoreDnsServiceExists             types.Bool `tfsdk:"aws_coredns_service_exists"`
	AwsCoreDnsServiceAccountExists      types.Bool `tfsdk:"aws_coredns_service_account_exists"`
	AwsCoreDnsConfigMapExists           types.Bool `tfsdk:"aws_coredns_config_map_exists"`
	AwsCoreDnsPodDisruptionBudgetExists types.Bool `tfsdk:"aws_coredns_pod_disruption_budget_exists"`

	CorednsDeploymentLabelHelmReleaseNameSet      types.Bool `tfsdk:"coredns_deployment_label_helm_release_name_set"`
	CorednsDeploymentLabelHelmReleaseNamespaceSet types.Bool `tfsdk:"coredns_deployment_label_helm_release_namespace_set"`
	CorednsDeploymentLabelManagedBySet            types.Bool `tfsdk:"coredns_deployment_label_managed_by_set"`
	CorednsDeploymentLabelAmazonManagedRemoved    types.Bool `tfsdk:"coredns_deployment_label_amazon_managed_removed"`

	CorednsServiceLabelHelmReleaseNameSet      types.Bool `tfsdk:"coredns_service_label_helm_release_name_set"`
	CorednsServiceLabelHelmReleaseNamespaceSet types.Bool `tfsdk:"coredns_service_label_helm_release_namespace_set"`
	CorednsServiceLabelManagedBySet            types.Bool `tfsdk:"coredns_service_label_managed_by_set"`
	CorednsServiceLabelAmazonManagedRemoved    types.Bool `tfsdk:"coredns_service_label_amazon_managed_removed"`

	CorednsServiceAccountLabelHelmReleaseNameSet      types.Bool `tfsdk:"coredns_service_account_label_helm_release_name_set"`
	CorednsServiceAccountLabelHelmReleaseNamespaceSet types.Bool `tfsdk:"coredns_service_account_label_helm_release_namespace_set"`
	CorednsServiceAccountLabelManagedBySet            types.Bool `tfsdk:"coredns_service_account_label_managed_by_set"`
	CorednsServiceAccountLabelAmazonManagedRemoved    types.Bool `tfsdk:"coredns_service_account_label_amazon_managed_removed"`

	CorednsConfigMapLabelHelmReleaseNameSet      types.Bool `tfsdk:"coredns_config_map_label_helm_release_name_set"`
	CorednsConfigMapLabelHelmReleaseNamespaceSet types.Bool `tfsdk:"coredns_config_map_label_helm_release_namespace_set"`
	CorednsConfigMapLabelManagedBySet            types.Bool `tfsdk:"coredns_config_map_label_managed_by_set"`
	CorednsConfigMapLabelAmazonManagedRemoved    types.Bool `tfsdk:"coredns_config_map_label_amazon_managed_removed"`

	CorednsPodDistruptionBudgetLabelHelmReleaseNameSet      types.Bool `tfsdk:"coredns_pod_disruption_budget_label_helm_release_name_set"`
	CorednsPodDistruptionBudgetLabelHelmReleaseNamespaceSet types.Bool `tfsdk:"coredns_pod_disruption_budget_label_helm_release_namespace_set"`
	CorednsPodDistruptionBudgetLabelManagedBySet            types.Bool `tfsdk:"coredns_pod_disruption_budget_label_managed_by_set"`
	CorednsPodDistruptionBudgetLabelAmazonManagedRemoved    types.Bool `tfsdk:"coredns_pod_disruption_budget_label_amazon_managed_removed"`
}

func (r *JobResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job"
}

func (r *JobResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Cleans an EKS cluster of default AWS-CNI, Kube-Proxy and imports CoreDNS deployment " +
			"and service into Helm. By importing CoreDNS into Helm, we don't loose DNS at any point and we " +
			"can manage CoreDNS using a Helm chart.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: `ID of the job.`,
				Computed:    true,
			},

			"remove_aws_cni": schema.BoolAttribute{
				MarkdownDescription: "Remove **AWS-CNI** from EKS cluster",
				Description:         "Remove AWS-CNI from EKS cluster",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},

			"remove_kube_proxy": schema.BoolAttribute{
				MarkdownDescription: "Remove **Kube-Proxy** from EKS cluster",
				Description:         "Remove Kube-Proxy from EKS cluster",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},

			"remove_core_dns": schema.BoolAttribute{
				MarkdownDescription: "Remove **CoreDNS** from EKS cluster",
				Description:         "Remove CoreDNS from EKS cluster",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},

			"import_coredns_to_helm": schema.BoolAttribute{
				Description: "Add helm attributes to CoreDns service and deployment, so that it can be managed by Helm.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},

			"aws_cni_daemonset_exists": schema.BoolAttribute{
				MarkdownDescription: "Does **AWS CNI** daemonset exist.",
				Description:         "Does AWS CNI daemonset exist.",
				Computed:            true,
			},

			"kube_proxy_daemonset_exists": schema.BoolAttribute{
				MarkdownDescription: "Does **Kube-Proxy** daemonset exist.",
				Description:         "Does Kube-Proxy daemonset exist.",
				Computed:            true,
			},

			"kube_proxy_config_map_exists": schema.BoolAttribute{
				MarkdownDescription: "Does **Kube-Proxy** config map exist.",
				Description:         "Does Kube-Proxy config map exist.",
				Computed:            true,
			},

			"aws_coredns_deployment_exists": schema.BoolAttribute{
				MarkdownDescription: "Does **AWS CoreDNS** deployment exist.",
				Description:         "Does AWS CoreDNS deployment exist.",
				Computed:            true,
			},

			"aws_coredns_service_exists": schema.BoolAttribute{
				MarkdownDescription: "Does **AWS CoreDNS** service exist.",
				Description:         "Does AWS CoreDNS service exist.",
				Computed:            true,
			},

			"aws_coredns_service_account_exists": schema.BoolAttribute{
				MarkdownDescription: "Does **AWS CoreDNS** service account exist.",
				Description:         "Does AWS CoreDNS service account exist.",
				Computed:            true,
			},

			"aws_coredns_config_map_exists": schema.BoolAttribute{
				MarkdownDescription: "Does **AWS CoreDNS** config map exist.",
				Description:         "Does AWS CoreDNS config map exist.",
				Computed:            true,
			},

			"aws_coredns_pod_disruption_budget_exists": schema.BoolAttribute{
				MarkdownDescription: "Does **AWS CoreDNS** pod disruption budget exist.",
				Description:         "Does AWS CoreDNS pod disruption budget exist.",
				Computed:            true,
			},

			"coredns_deployment_label_helm_release_name_set": schema.BoolAttribute{
				MarkdownDescription: "Does CoreDNS deployment have label **meta.helm.sh/release-name** with value of **coredns**. Returns **true** if deployment does not exist as Helm chart can be deployed.",
				Description:         "Does CoreDNS deployment have label meta.helm.sh/release-name with value of coredns. Returns true if deployment does not exist as Helm chart can be deployed.",
				Computed:            true,
			},

			"coredns_deployment_label_helm_release_namespace_set": schema.BoolAttribute{
				MarkdownDescription: "Does CoreDNS deployment have label **meta.helm.sh/release-namespace** with value of **kube-system**. Returns **true** if deployment does not exist as Helm chart can be deployed.",
				Description:         "Does CoreDNS deployment have label meta.helm.sh/release-namespace with value of kube-system. Returns true if deployment does not exist as Helm chart can be deployed.",
				Computed:            true,
			},

			"coredns_deployment_label_managed_by_set": schema.BoolAttribute{
				MarkdownDescription: "Does CoreDNS deployment have label **app.kubernetes.io/managed-by** with value of **Helm**. Returns **true** if deployment does not exist as Helm chart can be deployed.",
				Description:         "Does CoreDNS deployment have label app.kubernetes.io/managed-by with value of Helm. Returns true if deployment does not exist as Helm chart can be deployed.",
				Computed:            true,
			},

			"coredns_deployment_label_amazon_managed_removed": schema.BoolAttribute{
				MarkdownDescription: "Is label **eks.amazonaws.com/component** removed. Returns **true** if deployment does not exist as Helm chart can be deployed.",
				Description:         "Is label eks.amazonaws.com/component removed. Returns true if deployment does not exist as Helm chart can be deployed.",
				Computed:            true,
			},

			"coredns_service_label_helm_release_name_set": schema.BoolAttribute{
				MarkdownDescription: "Does CoreDNS service have label **meta.helm.sh/release-name** with value of **coredns**. Returns **true** if service does not exist as Helm chart can be deployed.",
				Description:         "Does CoreDNS service have label meta.helm.sh/release-name with value of coredns. Returns true if service does not exist as Helm chart can be deployed.",
				Computed:            true,
			},

			"coredns_service_label_helm_release_namespace_set": schema.BoolAttribute{
				MarkdownDescription: "Does CoreDNS service have label **meta.helm.sh/release-namespace** with value of **kube-system**. Returns **true** if service does not exist as Helm chart can be deployed.",
				Description:         "Does CoreDNS service have label meta.helm.sh/release-namespace with value of kube-system. Returns true if service does not exist as Helm chart can be deployed.",
				Computed:            true,
			},

			"coredns_service_label_managed_by_set": schema.BoolAttribute{
				MarkdownDescription: "Does CoreDNS service have label **app.kubernetes.io/managed-by** with value of **Helm**. Returns **true** if service does not exist as Helm chart can be deployed.",
				Description:         "Does CoreDNS service have label app.kubernetes.io/managed-by with value of Helm. Returns true if service does not exist as Helm chart can be deployed.",
				Computed:            true,
			},

			"coredns_service_label_amazon_managed_removed": schema.BoolAttribute{
				MarkdownDescription: "Is label **eks.amazonaws.com/component** removed. Returns **true** if service does not exist as Helm chart can be deployed.",
				Description:         "Is label eks.amazonaws.com/component removed. Returns true if service does not exist as Helm chart can be deployed.",
				Computed:            true,
			},

			"coredns_service_account_label_helm_release_name_set": schema.BoolAttribute{
				MarkdownDescription: "Does CoreDNS service have label **meta.helm.sh/release-name** with value of **coredns**. Returns **true** if service account does not exist as Helm chart can be deployed.",
				Description:         "Does CoreDNS service have label meta.helm.sh/release-name with value of coredns. Returns true if service account does not exist as Helm chart can be deployed.",
				Computed:            true,
			},

			"coredns_service_account_label_helm_release_namespace_set": schema.BoolAttribute{
				MarkdownDescription: "Does CoreDNS service have label **meta.helm.sh/release-namespace** with value of **kube-system**. Returns **true** if service account does not exist as Helm chart can be deployed.",
				Description:         "Does CoreDNS service have label meta.helm.sh/release-namespace with value of kube-system. Returns true if service account does not exist as Helm chart can be deployed.",
				Computed:            true,
			},

			"coredns_service_account_label_managed_by_set": schema.BoolAttribute{
				MarkdownDescription: "Does CoreDNS service have label **app.kubernetes.io/managed-by** with value of **Helm**. Returns **true** if service account does not exist as Helm chart can be deployed.",
				Description:         "Does CoreDNS service have label app.kubernetes.io/managed-by with value of Helm. Returns true if service account does not exist as Helm chart can be deployed.",
				Computed:            true,
			},

			"coredns_service_account_label_amazon_managed_removed": schema.BoolAttribute{
				MarkdownDescription: "Is label **eks.amazonaws.com/component** removed. Returns **true** if service account does not exist as Helm chart can be deployed.",
				Description:         "Is label eks.amazonaws.com/component removed. Returns true if service account does not exist as Helm chart can be deployed.",
				Computed:            true,
			},

			"coredns_config_map_label_helm_release_name_set": schema.BoolAttribute{
				MarkdownDescription: "Does CoreDNS service have label **meta.helm.sh/release-name** with value of **coredns**. Returns **true** if config map does not exist as Helm chart can be deployed.",
				Description:         "Does CoreDNS service have label meta.helm.sh/release-name with value of coredns. Returns true if config map does not exist as Helm chart can be deployed.",
				Computed:            true,
			},

			"coredns_config_map_label_helm_release_namespace_set": schema.BoolAttribute{
				MarkdownDescription: "Does CoreDNS service have label **meta.helm.sh/release-namespace** with value of **kube-system**. Returns **true** if config map does not exist as Helm chart can be deployed.",
				Description:         "Does CoreDNS service have label meta.helm.sh/release-namespace with value of kube-system. Returns true if config map does not exist as Helm chart can be deployed.",
				Computed:            true,
			},

			"coredns_config_map_label_managed_by_set": schema.BoolAttribute{
				MarkdownDescription: "Does CoreDNS service have label **app.kubernetes.io/managed-by** with value of **Helm**. Returns **true** if config map does not exist as Helm chart can be deployed.",
				Description:         "Does CoreDNS service have label app.kubernetes.io/managed-by with value of Helm. Returns true if config map does not exist as Helm chart can be deployed.",
				Computed:            true,
			},

			"coredns_config_map_label_amazon_managed_removed": schema.BoolAttribute{
				MarkdownDescription: "Is label **eks.amazonaws.com/component** removed. Returns **true** if config map does not exist as Helm chart can be deployed.",
				Description:         "Is label eks.amazonaws.com/component removed. Returns true if config map does not exist as Helm chart can be deployed.",
				Computed:            true,
			},

			"coredns_pod_disruption_budget_label_helm_release_name_set": schema.BoolAttribute{
				MarkdownDescription: "Does CoreDNS service have label **meta.helm.sh/release-name** with value of **coredns**. Returns **true** if pod disruption budget does not exist as Helm chart can be deployed.",
				Description:         "Does CoreDNS service have label meta.helm.sh/release-name with value of coredns. Returns true if pod disruption budget does not exist as Helm chart can be deployed.",
				Computed:            true,
			},

			"coredns_pod_disruption_budget_label_helm_release_namespace_set": schema.BoolAttribute{
				MarkdownDescription: "Does CoreDNS service have label **meta.helm.sh/release-namespace** with value of **kube-system**. Returns **true** if pod disruption budget does not exist as Helm chart can be deployed.",
				Description:         "Does CoreDNS service have label meta.helm.sh/release-namespace with value of kube-system. Returns true if pod disruption budget does not exist as Helm chart can be deployed.",
				Computed:            true,
			},

			"coredns_pod_disruption_budget_label_managed_by_set": schema.BoolAttribute{
				MarkdownDescription: "Does CoreDNS service have label **app.kubernetes.io/managed-by** with value of **Helm**. Returns **true** if pod disruption budget does not exist as Helm chart can be deployed.",
				Description:         "Does CoreDNS service have label app.kubernetes.io/managed-by with value of Helm. Returns true if pod disruption budget does not exist as Helm chart can be deployed.",
				Computed:            true,
			},

			"coredns_pod_disruption_budget_label_amazon_managed_removed": schema.BoolAttribute{
				MarkdownDescription: "Is label **eks.amazonaws.com/component** removed. Returns **true** if pod disruption budget does not exist as Helm chart can be deployed.",
				Description:         "Is label eks.amazonaws.com/component removed. Returns true if pod disruption budget does not exist as Helm chart can be deployed.",
				Computed:            true,
			},
		},
	}
}

func (r *JobResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	cleanEksProviderResourceData, ok := req.ProviderData.(*CleanEksProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *CleanEksProviderResourceData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	} else {
		r.provider = cleanEksProviderResourceData
	}
}

func (r *JobResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Debug(ctx, "Creating clean EKS resource")

	// Load entire configuration into the model
	var model JobResourceModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Loaded job configuration", map[string]interface{}{
		"jobConfig": fmt.Sprintf("%+v", model),
	})

	cleanEksProviderResourceData := r.provider
	if cleanEksProviderResourceData == nil {
		res.Diagnostics.AddError(
			"Provider not configured",
			fmt.Sprintf("Provider not configured"),
		)
		return
	}

	var err error

	clientSet := cleanEksProviderResourceData.clientSet
	if clientSet == nil {
		clientSet, err = cleanEksProviderResourceData.GetClientSet(ctx)
		if err != nil {
			res.Diagnostics.AddError(
				"Error getting Kubernetes client during JobResource.Create",
				fmt.Sprintf("Error getting Kubernetes client during JobResource.Create: %s", err),
			)
			return
		}
		cleanEksProviderResourceData.clientSet = clientSet
	}

	removeAwsCni := true
	if !(model.RemoveAwsCni.IsNull() || model.RemoveAwsCni.IsUnknown()) {
		removeAwsCni = model.RemoveAwsCni.ValueBool()
	}

	removeKubeProxy := true
	if !(model.RemoveKubeProxy.IsNull() || model.RemoveKubeProxy.IsUnknown()) {
		removeKubeProxy = model.RemoveKubeProxy.ValueBool()
	}

	removeCoreDns := true
	if !(model.RemoveCoreDns.IsNull() || model.RemoveCoreDns.IsUnknown()) {
		removeCoreDns = model.RemoveCoreDns.ValueBool()
	}

	importCorednsToHelm := false
	if !(model.ImportCorednsToHelm.IsNull() || model.ImportCorednsToHelm.IsUnknown()) {
		importCorednsToHelm = model.ImportCorednsToHelm.ValueBool()
	}

	if removeAwsCni || removeKubeProxy || removeCoreDns || importCorednsToHelm {
		if removeAwsCni {
			_, err = DeleteDaemonset(ctx, clientSet, "kube-system", "aws-node")
			if err != nil {
				res.Diagnostics.AddError(
					"Error removing AWS CNI daemonset",
					fmt.Sprintf("Error removing AWS CNI daemonset: %s", err),
				)
				return
			}
		}

		if removeKubeProxy {
			_, err = DeleteDaemonset(ctx, clientSet, "kube-system", "kube-proxy")
			if err != nil {
				res.Diagnostics.AddError(
					"Error removing Kube Proxy daemonset",
					fmt.Sprintf("Error removing Kube Proxy daemonset: %s", err),
				)
				return
			}

			_, err = DeleteConfigMap(ctx, clientSet, "kube-system", "kube-proxy")
			if err != nil {
				res.Diagnostics.AddError(
					"Error removing Kube Proxy config map",
					fmt.Sprintf("Error removing Kube Proxy config map: %s", err),
				)
				return
			}
		}

		if removeCoreDns {
			// We only want to delete the Amazon CoreDNS and not any further deployed versions
			deploymentExistsAndIsAwsOne, err := DeploymentExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error checking CoreDNS deployment is AWS one",
					fmt.Sprintf("Error checking CoreDNS deployment is AWS one: %s", err),
				)
				return
			}

			if deploymentExistsAndIsAwsOne {
				_, err = DeleteDeployment(ctx, clientSet, "kube-system", "coredns")
				if err != nil {
					res.Diagnostics.AddError(
						"Error removing CoreDNS deployment",
						fmt.Sprintf("Error removing CoreDNS deployment: %s", err),
					)
					return
				}
			}

			serviceExistsAndIsAwsOne, err := ServiceExistsAndIsAwsOne(ctx, clientSet, "kube-system", "kube-dns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error checking CoreDNS service is AWS one",
					fmt.Sprintf("Error checking CoreDNS service is AWS one: %s", err),
				)
				return
			}

			if serviceExistsAndIsAwsOne {
				_, err = DeleteService(ctx, clientSet, "kube-system", "kube-dns")
				if err != nil {
					res.Diagnostics.AddError(
						"Error removing CoreDNS service",
						fmt.Sprintf("Error removing CoreDNS service: %s", err),
					)
					return
				}
			}

			serviceAccountExistsAndIsAwsOne, err := ServiceAccountExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error checking CoreDNS service account is AWS one",
					fmt.Sprintf("Error checking CoreDNS service account is AWS one: %s", err),
				)
				return
			}

			if serviceAccountExistsAndIsAwsOne {
				_, err = DeleteServiceAccount(ctx, clientSet, "kube-system", "coredns")
				if err != nil {
					res.Diagnostics.AddError(
						"Error removing CoreDNS service account",
						fmt.Sprintf("Error removing CoreDNS service account: %s", err),
					)
					return
				}
			}

			configMapExistsAndIsAwsOne, err := ConfigMapExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error checking CoreDNS config map is AWS one",
					fmt.Sprintf("Error checking CoreDNS config map is AWS one: %s", err),
				)
				return
			}

			if configMapExistsAndIsAwsOne {
				_, err = DeleteConfigMap(ctx, clientSet, "kube-system", "coredns")
				if err != nil {
					res.Diagnostics.AddError(
						"Error removing CoreDNS configmap",
						fmt.Sprintf("Error removing CoreDNS configmap: %s", err),
					)
					return
				}
			}

			podDisruptionBudgetExistsAndIsAwsOne, err := PodDisruptionBudgetExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error checking CoreDNS pod disruption budget is AWS one",
					fmt.Sprintf("Error checking CoreDNS pod disruption budget is AWS one: %s", err),
				)
				return
			}

			if podDisruptionBudgetExistsAndIsAwsOne {
				_, err = DeletePodDisruptionBudget(ctx, clientSet, "kube-system", "coredns")
				if err != nil {
					res.Diagnostics.AddError(
						"Error removing CoreDNS pod disruption budget",
						fmt.Sprintf("Error removing CoreDNS pod disruption budget: %s", err),
					)
					return
				}
			}
		}

		if !removeCoreDns && importCorednsToHelm {
			err = ImportDeploymentIntoHelm(ctx, clientSet, "kube-system", "coredns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error importing CoreDns deployment to Helm",
					fmt.Sprintf("Error importing CoreDns deployment to Helm: %s", err),
				)
				return
			}

			err = ImportServiceIntoHelm(ctx, clientSet, "kube-system", "kube-dns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error importing CoreDns service to Helm",
					fmt.Sprintf("Error importing CoreDns service to Helm: %s", err),
				)
				return
			}

			err = ImportServiceAccountIntoHelm(ctx, clientSet, "kube-system", "coredns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error importing CoreDns service account to Helm",
					fmt.Sprintf("Error importing CoreDns service account to Helm: %s", err),
				)
				return
			}

			err = ImportConfigMapAccountIntoHelm(ctx, clientSet, "kube-system", "coredns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error importing CoreDns config map to Helm",
					fmt.Sprintf("Error importing CoreDns config map to Helm: %s", err),
				)
				return
			}

			err = ImportPodDisruptionBudgetIntoHelm(ctx, clientSet, "kube-system", "coredns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error importing CoreDns pod disruption budget to Helm",
					fmt.Sprintf("Error importing CoreDns pod disruption budget to Helm: %s", err),
				)
				return
			}
		}
	}

	// Read kubernetes to populate model
	awsCniDaemonsetExists, err := DaemonsetExist(ctx, clientSet, "kube-system", "aws-node")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for AWS CNI",
			fmt.Sprintf("Error checking daemonset for AWS CNI: %s", err),
		)
		return
	}
	model.AwsCniDaemonsetExists = basetypes.NewBoolValue(removeAwsCni && awsCniDaemonsetExists)

	model.RemoveAwsCni = basetypes.NewBoolValue(!(awsCniDaemonsetExists))

	kubeProxyDaemonsetExists, err := DaemonsetExist(ctx, clientSet, "kube-system", "kube-proxy")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for Kube Proxy",
			fmt.Sprintf("Error checking for Kube Proxy: %s", err),
		)
		return
	}
	model.KubeProxyDaemonsetExists = basetypes.NewBoolValue(kubeProxyDaemonsetExists)

	kubeProxyConfigMapExists, err := ConfigMapExist(ctx, clientSet, "kube-system", "kube-proxy")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for Kube Proxy config map",
			fmt.Sprintf("Error checking for Kube Proxy config map: %s", err),
		)
		return
	}
	model.KubeProxyConfigMapExists = basetypes.NewBoolValue(kubeProxyConfigMapExists)

	model.RemoveKubeProxy = basetypes.NewBoolValue(removeKubeProxy && !(kubeProxyDaemonsetExists && kubeProxyConfigMapExists))

	awsCoreDnsAwsDeploymentExists, err := DeploymentExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for CoreDNS deployment",
			fmt.Sprintf("Error checking for Kube Proxy config map: %s", err),
		)
		return
	}
	model.AwsCoreDnsDeploymentExists = basetypes.NewBoolValue(awsCoreDnsAwsDeploymentExists)

	awsCoreDnsServiceExists, err := ServiceExistsAndIsAwsOne(ctx, clientSet, "kube-system", "kube-dns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for CoreDNS service",
			fmt.Sprintf("Error checking for CoreDNS service: %s", err),
		)
		return
	}
	model.AwsCoreDnsServiceExists = basetypes.NewBoolValue(awsCoreDnsServiceExists)

	awsCoreDnsServiceAccountExists, err := ServiceAccountExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for CoreDNS service account",
			fmt.Sprintf("Error checking for CoreDNS service account: %s", err),
		)
		return
	}
	model.AwsCoreDnsServiceAccountExists = basetypes.NewBoolValue(awsCoreDnsServiceAccountExists)

	awsCoreDnsConfigMapExists, err := ConfigMapExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for CoreDNS config map",
			fmt.Sprintf("Error checking for CoreDNS config map: %s", err),
		)
		return
	}
	model.AwsCoreDnsConfigMapExists = basetypes.NewBoolValue(awsCoreDnsConfigMapExists)

	awsCoreDnsPodDisruptionBudgetExists, err := PodDisruptionBudgetExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for CoreDNS pod disruption budget",
			fmt.Sprintf("Error checking for CoreDNS pod disruption budget: %s", err),
		)
		return
	}
	model.AwsCoreDnsPodDisruptionBudgetExists = basetypes.NewBoolValue(awsCoreDnsPodDisruptionBudgetExists)

	model.RemoveCoreDns = basetypes.NewBoolValue(removeCoreDns && !(awsCoreDnsAwsDeploymentExists && awsCoreDnsServiceExists && awsCoreDnsServiceAccountExists && awsCoreDnsConfigMapExists && awsCoreDnsPodDisruptionBudgetExists))

	deploymentHelmReleaseNameAnnotationSet, deploymentHelmReleaseNamespaceAnnotationSet, deploymentManagedByLabelSet, deploymentAmazonManagedLabelRemoved, err := DeploymentImportedIntoHelm(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking CoreDns deployment to Helm",
			fmt.Sprintf("Error checking CoreDns deployment to Helm: %s", err),
		)
		return
	}
	model.CorednsDeploymentLabelHelmReleaseNameSet = basetypes.NewBoolValue(deploymentHelmReleaseNameAnnotationSet)
	model.CorednsDeploymentLabelHelmReleaseNamespaceSet = basetypes.NewBoolValue(deploymentHelmReleaseNamespaceAnnotationSet)
	model.CorednsDeploymentLabelManagedBySet = basetypes.NewBoolValue(deploymentManagedByLabelSet)
	model.CorednsDeploymentLabelAmazonManagedRemoved = basetypes.NewBoolValue(deploymentAmazonManagedLabelRemoved)

	serviceHelmReleaseNameAnnotationSet, serviceHelmReleaseNamespaceAnnotationSet, serviceManagedByLabelSet, serviceAmazonManagedLabelRemoved, err := ServiceImportedIntoHelm(ctx, clientSet, "kube-system", "kube-dns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking CoreDns service to Helm",
			fmt.Sprintf("Error checking CoreDns service to Helm: %s", err),
		)
		return
	}

	model.CorednsServiceLabelHelmReleaseNameSet = basetypes.NewBoolValue(serviceHelmReleaseNameAnnotationSet)
	model.CorednsServiceLabelHelmReleaseNamespaceSet = basetypes.NewBoolValue(serviceHelmReleaseNamespaceAnnotationSet)
	model.CorednsServiceLabelManagedBySet = basetypes.NewBoolValue(serviceManagedByLabelSet)
	model.CorednsServiceLabelAmazonManagedRemoved = basetypes.NewBoolValue(serviceAmazonManagedLabelRemoved)

	serviceAccountHelmReleaseNameAnnotationSet, serviceAccountHelmReleaseNamespaceAnnotationSet, serviceAccountManagedByLabelSet, serviceAccountAmazonManagedLabelRemoved, err := ServiceAccountImportedIntoHelm(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking CoreDns service account to Helm",
			fmt.Sprintf("Error checking CoreDns service account to Helm: %s", err),
		)
		return
	}

	model.CorednsServiceAccountLabelHelmReleaseNameSet = basetypes.NewBoolValue(serviceAccountHelmReleaseNameAnnotationSet)
	model.CorednsServiceAccountLabelHelmReleaseNamespaceSet = basetypes.NewBoolValue(serviceAccountHelmReleaseNamespaceAnnotationSet)
	model.CorednsServiceAccountLabelManagedBySet = basetypes.NewBoolValue(serviceAccountManagedByLabelSet)
	model.CorednsServiceAccountLabelAmazonManagedRemoved = basetypes.NewBoolValue(serviceAccountAmazonManagedLabelRemoved)

	configMapHelmReleaseNameAnnotationSet, configMapHelmReleaseNamespaceAnnotationSet, configMapManagedByLabelSet, configMapAmazonManagedLabelRemoved, err := ConfigMapImportedIntoHelm(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking CoreDns config map to Helm",
			fmt.Sprintf("Error checking CoreDns config map to Helm: %s", err),
		)
		return
	}

	model.CorednsConfigMapLabelHelmReleaseNameSet = basetypes.NewBoolValue(configMapHelmReleaseNameAnnotationSet)
	model.CorednsConfigMapLabelHelmReleaseNamespaceSet = basetypes.NewBoolValue(configMapHelmReleaseNamespaceAnnotationSet)
	model.CorednsConfigMapLabelManagedBySet = basetypes.NewBoolValue(configMapManagedByLabelSet)
	model.CorednsConfigMapLabelAmazonManagedRemoved = basetypes.NewBoolValue(configMapAmazonManagedLabelRemoved)

	podDistruptionBudgetHelmReleaseNameAnnotationSet, podDistruptionBudgetHelmReleaseNamespaceAnnotationSet, podDistruptionBudgetManagedByLabelSet, podDistruptionBudgetAmazonManagedLabelRemoved, err := ConfigMapImportedIntoHelm(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking CoreDns pod disruption budget to Helm",
			fmt.Sprintf("Error checking CoreDns pod disruption budget to Helm: %s", err),
		)
		return
	}

	model.CorednsPodDistruptionBudgetLabelHelmReleaseNameSet = basetypes.NewBoolValue(podDistruptionBudgetHelmReleaseNameAnnotationSet)
	model.CorednsPodDistruptionBudgetLabelHelmReleaseNamespaceSet = basetypes.NewBoolValue(podDistruptionBudgetHelmReleaseNamespaceAnnotationSet)
	model.CorednsPodDistruptionBudgetLabelManagedBySet = basetypes.NewBoolValue(podDistruptionBudgetManagedByLabelSet)
	model.CorednsPodDistruptionBudgetLabelAmazonManagedRemoved = basetypes.NewBoolValue(podDistruptionBudgetAmazonManagedLabelRemoved)

	model.ImportCorednsToHelm = basetypes.NewBoolValue(importCorednsToHelm && (deploymentHelmReleaseNameAnnotationSet && deploymentHelmReleaseNamespaceAnnotationSet && deploymentManagedByLabelSet && deploymentAmazonManagedLabelRemoved && serviceHelmReleaseNameAnnotationSet && serviceHelmReleaseNamespaceAnnotationSet && serviceManagedByLabelSet && serviceAmazonManagedLabelRemoved && serviceAccountHelmReleaseNameAnnotationSet && serviceAccountHelmReleaseNamespaceAnnotationSet && serviceAccountManagedByLabelSet && serviceAccountAmazonManagedLabelRemoved && configMapHelmReleaseNameAnnotationSet && configMapHelmReleaseNamespaceAnnotationSet && configMapManagedByLabelSet && configMapAmazonManagedLabelRemoved && podDistruptionBudgetHelmReleaseNameAnnotationSet && podDistruptionBudgetHelmReleaseNamespaceAnnotationSet && podDistruptionBudgetManagedByLabelSet && podDistruptionBudgetAmazonManagedLabelRemoved))

	model.ID = basetypes.NewStringValue(r.provider.Host)

	// Finally, set the state
	tflog.Debug(ctx, "Storing job info into the state")
	res.Diagnostics.Append(res.State.Set(ctx, model)...)
}

func (r *JobResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	// NO-OP: all there is to read is in the State, and response is already populated with that.
	tflog.Debug(ctx, "Reading job from state")

	// Load entire configuration into the model
	var model JobResourceModel
	res.Diagnostics.Append(req.State.Get(ctx, &model)...)
	tflog.Debug(ctx, "Loaded job configuration", map[string]interface{}{
		"jobConfig": fmt.Sprintf("%+v", model),
	})

	cleanEksProviderResourceData := r.provider
	if cleanEksProviderResourceData == nil {
		res.Diagnostics.AddError(
			"Provider not configured",
			fmt.Sprintf("Provider not configured"),
		)
		return
	}

	var err error

	clientSet := cleanEksProviderResourceData.clientSet
	if clientSet == nil {
		clientSet, err = cleanEksProviderResourceData.GetClientSet(ctx)
		if err != nil {
			res.Diagnostics.AddError(
				"Error getting Kubernetes client during JobResource.Read",
				fmt.Sprintf("Error getting Kubernetes client during JobResource.Read: %s", err),
			)
			return
		}
		cleanEksProviderResourceData.clientSet = clientSet
	}

	removeAwsCni := true
	if !(model.RemoveAwsCni.IsNull() || model.RemoveAwsCni.IsUnknown()) {
		removeAwsCni = model.RemoveAwsCni.ValueBool()
	}

	removeKubeProxy := true
	if !(model.RemoveKubeProxy.IsNull() || model.RemoveKubeProxy.IsUnknown()) {
		removeKubeProxy = model.RemoveKubeProxy.ValueBool()
	}

	removeCoreDns := true
	if !(model.RemoveCoreDns.IsNull() || model.RemoveCoreDns.IsUnknown()) {
		removeCoreDns = model.RemoveCoreDns.ValueBool()
	}

	importCorednsToHelm := false
	if !(model.ImportCorednsToHelm.IsNull() || model.ImportCorednsToHelm.IsUnknown()) {
		importCorednsToHelm = model.ImportCorednsToHelm.ValueBool()
	}

	// Read kubernetes to populate model

	awsCniDaemonsetExists, err := DaemonsetExist(ctx, clientSet, "kube-system", "aws-node")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for AWS CNI daemonset",
			fmt.Sprintf("Error checking daemonset for AWS CNI daemonset: %s", err),
		)
		return
	}
	model.AwsCniDaemonsetExists = basetypes.NewBoolValue(awsCniDaemonsetExists)

	model.RemoveAwsCni = basetypes.NewBoolValue(removeAwsCni && !(awsCniDaemonsetExists))

	kubeProxyDaemonsetExists, err := DaemonsetExist(ctx, clientSet, "kube-system", "kube-proxy")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for Kube Proxy daemonset",
			fmt.Sprintf("Error checking for Kube Proxy daemonset: %s", err),
		)
		return
	}
	model.KubeProxyDaemonsetExists = basetypes.NewBoolValue(kubeProxyDaemonsetExists)

	kubeProxyConfigMapExists, err := ConfigMapExist(ctx, clientSet, "kube-system", "kube-proxy")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for Kube Proxy config map",
			fmt.Sprintf("Error checking for Kube Proxy config map: %s", err),
		)
		return
	}
	model.KubeProxyConfigMapExists = basetypes.NewBoolValue(kubeProxyConfigMapExists)

	model.RemoveKubeProxy = basetypes.NewBoolValue(removeKubeProxy && !(kubeProxyDaemonsetExists && kubeProxyConfigMapExists))

	awsCoreDnsAwsDeploymentExists, err := DeploymentExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for CoreDNS deployment",
			fmt.Sprintf("Error checking for Kube Proxy config map: %s", err),
		)
		return
	}
	model.AwsCoreDnsDeploymentExists = basetypes.NewBoolValue(awsCoreDnsAwsDeploymentExists)

	awsCoreDnsServiceExists, err := ServiceExistsAndIsAwsOne(ctx, clientSet, "kube-system", "kube-dns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for CoreDNS service",
			fmt.Sprintf("Error checking for CoreDNS service: %s", err),
		)
		return
	}
	model.AwsCoreDnsServiceExists = basetypes.NewBoolValue(awsCoreDnsServiceExists)

	awsCoreDnsServiceAccountExists, err := ServiceAccountExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for CoreDNS service account",
			fmt.Sprintf("Error checking for CoreDNS service account: %s", err),
		)
		return
	}
	model.AwsCoreDnsServiceAccountExists = basetypes.NewBoolValue(awsCoreDnsServiceAccountExists)

	awsCoreDnsConfigMapExists, err := ConfigMapExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for CoreDNS config map",
			fmt.Sprintf("Error checking for CoreDNS config map: %s", err),
		)
		return
	}
	model.AwsCoreDnsConfigMapExists = basetypes.NewBoolValue(awsCoreDnsConfigMapExists)

	awsCoreDnsPodDisruptionBudgetExists, err := PodDisruptionBudgetExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for CoreDNS pod disruption budget",
			fmt.Sprintf("Error checking for CoreDNS pod disruption budget: %s", err),
		)
		return
	}
	model.AwsCoreDnsPodDisruptionBudgetExists = basetypes.NewBoolValue(awsCoreDnsPodDisruptionBudgetExists)

	model.RemoveCoreDns = basetypes.NewBoolValue(removeCoreDns && !(awsCoreDnsAwsDeploymentExists && awsCoreDnsServiceExists && awsCoreDnsServiceAccountExists && awsCoreDnsConfigMapExists && awsCoreDnsPodDisruptionBudgetExists))

	deploymentHelmReleaseNameAnnotationSet, deploymentHelmReleaseNamespaceAnnotationSet, deploymentManagedByLabelSet, deploymentAmazonManagedLabelRemoved, err := DeploymentImportedIntoHelm(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking CoreDns deployment to Helm",
			fmt.Sprintf("Error checking CoreDns deployment to Helm: %s", err),
		)
		return
	}
	model.CorednsDeploymentLabelHelmReleaseNameSet = basetypes.NewBoolValue(deploymentHelmReleaseNameAnnotationSet)
	model.CorednsDeploymentLabelHelmReleaseNamespaceSet = basetypes.NewBoolValue(deploymentHelmReleaseNamespaceAnnotationSet)
	model.CorednsDeploymentLabelManagedBySet = basetypes.NewBoolValue(deploymentManagedByLabelSet)
	model.CorednsDeploymentLabelAmazonManagedRemoved = basetypes.NewBoolValue(deploymentAmazonManagedLabelRemoved)

	serviceHelmReleaseNameAnnotationSet, serviceHelmReleaseNamespaceAnnotationSet, serviceManagedByLabelSet, serviceAmazonManagedLabelRemoved, err := ServiceImportedIntoHelm(ctx, clientSet, "kube-system", "kube-dns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking CoreDns service to Helm",
			fmt.Sprintf("Error checking CoreDns service to Helm: %s", err),
		)
		return
	}

	model.CorednsServiceLabelHelmReleaseNameSet = basetypes.NewBoolValue(serviceHelmReleaseNameAnnotationSet)
	model.CorednsServiceLabelHelmReleaseNamespaceSet = basetypes.NewBoolValue(serviceHelmReleaseNamespaceAnnotationSet)
	model.CorednsServiceLabelManagedBySet = basetypes.NewBoolValue(serviceManagedByLabelSet)
	model.CorednsServiceLabelAmazonManagedRemoved = basetypes.NewBoolValue(serviceAmazonManagedLabelRemoved)

	serviceAccountHelmReleaseNameAnnotationSet, serviceAccountHelmReleaseNamespaceAnnotationSet, serviceAccountManagedByLabelSet, serviceAccountAmazonManagedLabelRemoved, err := ServiceAccountImportedIntoHelm(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking CoreDns service account to Helm",
			fmt.Sprintf("Error checking CoreDns service account to Helm: %s", err),
		)
		return
	}

	model.CorednsServiceAccountLabelHelmReleaseNameSet = basetypes.NewBoolValue(serviceAccountHelmReleaseNameAnnotationSet)
	model.CorednsServiceAccountLabelHelmReleaseNamespaceSet = basetypes.NewBoolValue(serviceAccountHelmReleaseNamespaceAnnotationSet)
	model.CorednsServiceAccountLabelManagedBySet = basetypes.NewBoolValue(serviceAccountManagedByLabelSet)
	model.CorednsServiceAccountLabelAmazonManagedRemoved = basetypes.NewBoolValue(serviceAccountAmazonManagedLabelRemoved)

	configMapHelmReleaseNameAnnotationSet, configMapHelmReleaseNamespaceAnnotationSet, configMapManagedByLabelSet, configMapAmazonManagedLabelRemoved, err := ConfigMapImportedIntoHelm(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking CoreDns config map to Helm",
			fmt.Sprintf("Error checking CoreDns config map to Helm: %s", err),
		)
		return
	}

	model.CorednsConfigMapLabelHelmReleaseNameSet = basetypes.NewBoolValue(configMapHelmReleaseNameAnnotationSet)
	model.CorednsConfigMapLabelHelmReleaseNamespaceSet = basetypes.NewBoolValue(configMapHelmReleaseNamespaceAnnotationSet)
	model.CorednsConfigMapLabelManagedBySet = basetypes.NewBoolValue(configMapManagedByLabelSet)
	model.CorednsConfigMapLabelAmazonManagedRemoved = basetypes.NewBoolValue(configMapAmazonManagedLabelRemoved)

	podDistruptionBudgetHelmReleaseNameAnnotationSet, podDistruptionBudgetHelmReleaseNamespaceAnnotationSet, podDistruptionBudgetManagedByLabelSet, podDistruptionBudgetAmazonManagedLabelRemoved, err := ConfigMapImportedIntoHelm(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking CoreDns pod disruption budget to Helm",
			fmt.Sprintf("Error checking CoreDns pod disruption budget to Helm: %s", err),
		)
		return
	}

	model.CorednsPodDistruptionBudgetLabelHelmReleaseNameSet = basetypes.NewBoolValue(podDistruptionBudgetHelmReleaseNameAnnotationSet)
	model.CorednsPodDistruptionBudgetLabelHelmReleaseNamespaceSet = basetypes.NewBoolValue(podDistruptionBudgetHelmReleaseNamespaceAnnotationSet)
	model.CorednsPodDistruptionBudgetLabelManagedBySet = basetypes.NewBoolValue(podDistruptionBudgetManagedByLabelSet)
	model.CorednsPodDistruptionBudgetLabelAmazonManagedRemoved = basetypes.NewBoolValue(podDistruptionBudgetAmazonManagedLabelRemoved)

	model.ImportCorednsToHelm = basetypes.NewBoolValue(importCorednsToHelm && (deploymentHelmReleaseNameAnnotationSet && deploymentHelmReleaseNamespaceAnnotationSet && deploymentManagedByLabelSet && deploymentAmazonManagedLabelRemoved && serviceHelmReleaseNameAnnotationSet && serviceHelmReleaseNamespaceAnnotationSet && serviceManagedByLabelSet && serviceAmazonManagedLabelRemoved && serviceAccountHelmReleaseNameAnnotationSet && serviceAccountHelmReleaseNamespaceAnnotationSet && serviceAccountManagedByLabelSet && serviceAccountAmazonManagedLabelRemoved && configMapHelmReleaseNameAnnotationSet && configMapHelmReleaseNamespaceAnnotationSet && configMapManagedByLabelSet && configMapAmazonManagedLabelRemoved && podDistruptionBudgetHelmReleaseNameAnnotationSet && podDistruptionBudgetHelmReleaseNamespaceAnnotationSet && podDistruptionBudgetManagedByLabelSet && podDistruptionBudgetAmazonManagedLabelRemoved))

	model.ID = basetypes.NewStringValue(r.provider.Host)

	// Finally, set the state
	tflog.Debug(ctx, "Storing job info into the state")
	res.Diagnostics.Append(res.State.Set(ctx, model)...)
}

func (r *JobResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Debug(ctx, "Updating job")

	// Load entire configuration into the model
	var model JobResourceModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Loaded job configuration", map[string]interface{}{
		"jobConfig": fmt.Sprintf("%+v", model),
	})

	cleanEksProviderResourceData := r.provider
	if cleanEksProviderResourceData == nil {
		res.Diagnostics.AddError(
			"Provider not configured",
			fmt.Sprintf("Provider not configured"),
		)
		return
	}

	var err error

	clientSet := cleanEksProviderResourceData.clientSet
	if clientSet == nil {
		clientSet, err = cleanEksProviderResourceData.GetClientSet(ctx)
		if err != nil {
			res.Diagnostics.AddError(
				"Error getting Kubernetes client during JobResource.Update",
				fmt.Sprintf("Error getting Kubernetes client during JobResource.Update: %s", err),
			)
			return
		}
		cleanEksProviderResourceData.clientSet = clientSet
	}

	removeAwsCni := true
	if !(model.RemoveAwsCni.IsNull() || model.RemoveAwsCni.IsUnknown()) {
		removeAwsCni = model.RemoveAwsCni.ValueBool()
	}

	removeKubeProxy := true
	if !(model.RemoveKubeProxy.IsNull() || model.RemoveKubeProxy.IsUnknown()) {
		removeKubeProxy = model.RemoveKubeProxy.ValueBool()
	}

	removeCoreDns := true
	if !(model.RemoveCoreDns.IsNull() || model.RemoveCoreDns.IsUnknown()) {
		removeCoreDns = model.RemoveCoreDns.ValueBool()
	}

	importCorednsToHelm := false
	if !(model.ImportCorednsToHelm.IsNull() || model.ImportCorednsToHelm.IsUnknown()) {
		importCorednsToHelm = model.ImportCorednsToHelm.ValueBool()
	}

	if err != nil {
		res.Diagnostics.AddError(
			"Error making request",
			fmt.Sprintf("Error making request: %s", err),
		)
		return
	}

	if removeAwsCni || removeKubeProxy || importCorednsToHelm {
		if removeAwsCni {
			_, err = DeleteDaemonset(ctx, clientSet, "kube-system", "aws-node")
			if err != nil {
				res.Diagnostics.AddError(
					"Error removing AWS CNI daemonset",
					fmt.Sprintf("Error removing AWS CNI daemonset: %s", err),
				)
				return
			}
		}

		if removeKubeProxy {
			_, err = DeleteDaemonset(ctx, clientSet, "kube-system", "kube-proxy")
			if err != nil {
				res.Diagnostics.AddError(
					"Error removing Kube Proxy daemonset",
					fmt.Sprintf("Error removing Kube Proxy daemonset: %s", err),
				)
				return
			}

			_, err = DeleteConfigMap(ctx, clientSet, "kube-system", "kube-proxy")
			if err != nil {
				res.Diagnostics.AddError(
					"Error removing Kube Proxy config map",
					fmt.Sprintf("Error removing Kube Proxy config map: %s", err),
				)
				return
			}
		}

		if removeCoreDns {
			// We only want to delete the Amazon CoreDNS and not any further deployed versions
			deploymentExistsAndIsAwsOne, err := DeploymentExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error checking CoreDNS deployment is AWS one",
					fmt.Sprintf("Error checking CoreDNS deployment is AWS one: %s", err),
				)
				return
			}

			if deploymentExistsAndIsAwsOne {
				_, err = DeleteDeployment(ctx, clientSet, "kube-system", "coredns")
				if err != nil {
					res.Diagnostics.AddError(
						"Error removing CoreDNS deployment",
						fmt.Sprintf("Error removing CoreDNS deployment: %s", err),
					)
					return
				}
			}

			serviceExistsAndIsAwsOne, err := ServiceExistsAndIsAwsOne(ctx, clientSet, "kube-system", "kube-dns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error checking CoreDNS service is AWS one",
					fmt.Sprintf("Error checking CoreDNS service is AWS one: %s", err),
				)
				return
			}

			if serviceExistsAndIsAwsOne {
				_, err = DeleteService(ctx, clientSet, "kube-system", "kube-dns")
				if err != nil {
					res.Diagnostics.AddError(
						"Error removing CoreDNS service",
						fmt.Sprintf("Error removing CoreDNS service: %s", err),
					)
					return
				}
			}

			serviceAccountExistsAndIsAwsOne, err := ServiceAccountExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error checking CoreDNS service account is AWS one",
					fmt.Sprintf("Error checking CoreDNS service account is AWS one: %s", err),
				)
				return
			}

			if serviceAccountExistsAndIsAwsOne {
				_, err = DeleteServiceAccount(ctx, clientSet, "kube-system", "coredns")
				if err != nil {
					res.Diagnostics.AddError(
						"Error removing CoreDNS service account",
						fmt.Sprintf("Error removing CoreDNS service account: %s", err),
					)
					return
				}
			}

			configMapExistsAndIsAwsOne, err := ConfigMapExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error checking CoreDNS config map is AWS one",
					fmt.Sprintf("Error checking CoreDNS config map is AWS one: %s", err),
				)
				return
			}

			if configMapExistsAndIsAwsOne {
				_, err = DeleteConfigMap(ctx, clientSet, "kube-system", "coredns")
				if err != nil {
					res.Diagnostics.AddError(
						"Error removing CoreDNS configmap",
						fmt.Sprintf("Error removing CoreDNS configmap: %s", err),
					)
					return
				}
			}

			podDisruptionBudgetExistsAndIsAwsOne, err := PodDisruptionBudgetExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error checking CoreDNS pod disruption budget is AWS one",
					fmt.Sprintf("Error checking CoreDNS pod disruption budget is AWS one: %s", err),
				)
				return
			}

			if podDisruptionBudgetExistsAndIsAwsOne {
				_, err = DeletePodDisruptionBudget(ctx, clientSet, "kube-system", "coredns")
				if err != nil {
					res.Diagnostics.AddError(
						"Error removing CoreDNS pod disruption budget",
						fmt.Sprintf("Error removing CoreDNS pod disruption budget: %s", err),
					)
					return
				}
			}
		}

		if !removeCoreDns && importCorednsToHelm {
			err = ImportDeploymentIntoHelm(ctx, clientSet, "kube-system", "coredns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error importing CoreDns deployment to Helm",
					fmt.Sprintf("Error importing CoreDns deployment to Helm: %s", err),
				)
				return
			}

			err = ImportServiceIntoHelm(ctx, clientSet, "kube-system", "kube-dns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error importing CoreDns service to Helm",
					fmt.Sprintf("Error importing CoreDns service to Helm: %s", err),
				)
				return
			}

			err = ImportServiceAccountIntoHelm(ctx, clientSet, "kube-system", "coredns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error importing CoreDns service account to Helm",
					fmt.Sprintf("Error importing CoreDns service account to Helm: %s", err),
				)
				return
			}

			err = ImportConfigMapAccountIntoHelm(ctx, clientSet, "kube-system", "coredns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error importing CoreDns config map to Helm",
					fmt.Sprintf("Error importing CoreDns config map to Helm: %s", err),
				)
				return
			}

			err = ImportPodDisruptionBudgetIntoHelm(ctx, clientSet, "kube-system", "coredns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error importing CoreDns pod disruption budget to Helm",
					fmt.Sprintf("Error importing CoreDns pod disruption budget to Helm: %s", err),
				)
				return
			}
		}
	}

	// Read kubernetes to populate model

	awsCniDaemonsetExists, err := DaemonsetExist(ctx, clientSet, "kube-system", "aws-node")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for AWS CNI",
			fmt.Sprintf("Error checking daemonset for AWS CNI: %s", err),
		)
		return
	}
	model.AwsCniDaemonsetExists = basetypes.NewBoolValue(awsCniDaemonsetExists)

	model.RemoveAwsCni = basetypes.NewBoolValue(removeAwsCni && !(awsCniDaemonsetExists))

	kubeProxyDaemonsetExists, err := DaemonsetExist(ctx, clientSet, "kube-system", "kube-proxy")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for Kube Proxy",
			fmt.Sprintf("Error checking for Kube Proxy: %s", err),
		)
		return
	}
	model.KubeProxyDaemonsetExists = basetypes.NewBoolValue(kubeProxyDaemonsetExists)

	kubeProxyConfigMapExists, err := ConfigMapExist(ctx, clientSet, "kube-system", "kube-proxy")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for Kube Proxy config map",
			fmt.Sprintf("Error checking for Kube Proxy config map: %s", err),
		)
		return
	}
	model.KubeProxyConfigMapExists = basetypes.NewBoolValue(kubeProxyConfigMapExists)

	model.RemoveKubeProxy = basetypes.NewBoolValue(removeKubeProxy && !(kubeProxyDaemonsetExists && kubeProxyConfigMapExists))

	awsCoreDnsAwsDeploymentExists, err := DeploymentExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for CoreDNS deployment",
			fmt.Sprintf("Error checking for Kube Proxy config map: %s", err),
		)
		return
	}
	model.AwsCoreDnsDeploymentExists = basetypes.NewBoolValue(awsCoreDnsAwsDeploymentExists)

	awsCoreDnsServiceExists, err := ServiceExistsAndIsAwsOne(ctx, clientSet, "kube-system", "kube-dns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for CoreDNS service",
			fmt.Sprintf("Error checking for CoreDNS service: %s", err),
		)
		return
	}
	model.AwsCoreDnsServiceExists = basetypes.NewBoolValue(awsCoreDnsServiceExists)

	awsCoreDnsServiceAccountExists, err := ServiceAccountExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for CoreDNS service account",
			fmt.Sprintf("Error checking for CoreDNS service account: %s", err),
		)
		return
	}
	model.AwsCoreDnsServiceAccountExists = basetypes.NewBoolValue(awsCoreDnsServiceAccountExists)

	awsCoreDnsConfigMapExists, err := ConfigMapExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for CoreDNS config map",
			fmt.Sprintf("Error checking for CoreDNS config map: %s", err),
		)
		return
	}
	model.AwsCoreDnsConfigMapExists = basetypes.NewBoolValue(awsCoreDnsConfigMapExists)

	awsCoreDnsPodDisruptionBudgetExists, err := PodDisruptionBudgetExistsAndIsAwsOne(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for CoreDNS pod disruption budget",
			fmt.Sprintf("Error checking for CoreDNS pod disruption budget: %s", err),
		)
		return
	}
	model.AwsCoreDnsPodDisruptionBudgetExists = basetypes.NewBoolValue(awsCoreDnsPodDisruptionBudgetExists)

	model.RemoveCoreDns = basetypes.NewBoolValue(removeCoreDns && !(awsCoreDnsAwsDeploymentExists && awsCoreDnsServiceExists && awsCoreDnsServiceAccountExists && awsCoreDnsConfigMapExists && awsCoreDnsPodDisruptionBudgetExists))

	deploymentHelmReleaseNameAnnotationSet, deploymentHelmReleaseNamespaceAnnotationSet, deploymentManagedByLabelSet, deploymentAmazonManagedLabelRemoved, err := DeploymentImportedIntoHelm(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking CoreDns deployment to Helm",
			fmt.Sprintf("Error checking CoreDns deployment to Helm: %s", err),
		)
		return
	}
	model.CorednsDeploymentLabelHelmReleaseNameSet = basetypes.NewBoolValue(deploymentHelmReleaseNameAnnotationSet)
	model.CorednsDeploymentLabelHelmReleaseNamespaceSet = basetypes.NewBoolValue(deploymentHelmReleaseNamespaceAnnotationSet)
	model.CorednsDeploymentLabelManagedBySet = basetypes.NewBoolValue(deploymentManagedByLabelSet)
	model.CorednsDeploymentLabelAmazonManagedRemoved = basetypes.NewBoolValue(deploymentAmazonManagedLabelRemoved)

	serviceHelmReleaseNameAnnotationSet, serviceHelmReleaseNamespaceAnnotationSet, serviceManagedByLabelSet, serviceAmazonManagedLabelRemoved, err := ServiceImportedIntoHelm(ctx, clientSet, "kube-system", "kube-dns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking CoreDns service to Helm",
			fmt.Sprintf("Error checking CoreDns service to Helm: %s", err),
		)
		return
	}

	model.CorednsServiceLabelHelmReleaseNameSet = basetypes.NewBoolValue(serviceHelmReleaseNameAnnotationSet)
	model.CorednsServiceLabelHelmReleaseNamespaceSet = basetypes.NewBoolValue(serviceHelmReleaseNamespaceAnnotationSet)
	model.CorednsServiceLabelManagedBySet = basetypes.NewBoolValue(serviceManagedByLabelSet)
	model.CorednsServiceLabelAmazonManagedRemoved = basetypes.NewBoolValue(serviceAmazonManagedLabelRemoved)

	serviceAccountHelmReleaseNameAnnotationSet, serviceAccountHelmReleaseNamespaceAnnotationSet, serviceAccountManagedByLabelSet, serviceAccountAmazonManagedLabelRemoved, err := ServiceAccountImportedIntoHelm(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking CoreDns service account to Helm",
			fmt.Sprintf("Error checking CoreDns service account to Helm: %s", err),
		)
		return
	}

	model.CorednsServiceAccountLabelHelmReleaseNameSet = basetypes.NewBoolValue(serviceAccountHelmReleaseNameAnnotationSet)
	model.CorednsServiceAccountLabelHelmReleaseNamespaceSet = basetypes.NewBoolValue(serviceAccountHelmReleaseNamespaceAnnotationSet)
	model.CorednsServiceAccountLabelManagedBySet = basetypes.NewBoolValue(serviceAccountManagedByLabelSet)
	model.CorednsServiceAccountLabelAmazonManagedRemoved = basetypes.NewBoolValue(serviceAccountAmazonManagedLabelRemoved)

	configMapHelmReleaseNameAnnotationSet, configMapHelmReleaseNamespaceAnnotationSet, configMapManagedByLabelSet, configMapAmazonManagedLabelRemoved, err := ConfigMapImportedIntoHelm(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking CoreDns config map to Helm",
			fmt.Sprintf("Error checking CoreDns config map to Helm: %s", err),
		)
		return
	}

	model.CorednsConfigMapLabelHelmReleaseNameSet = basetypes.NewBoolValue(configMapHelmReleaseNameAnnotationSet)
	model.CorednsConfigMapLabelHelmReleaseNamespaceSet = basetypes.NewBoolValue(configMapHelmReleaseNamespaceAnnotationSet)
	model.CorednsConfigMapLabelManagedBySet = basetypes.NewBoolValue(configMapManagedByLabelSet)
	model.CorednsConfigMapLabelAmazonManagedRemoved = basetypes.NewBoolValue(configMapAmazonManagedLabelRemoved)

	podDistruptionBudgetHelmReleaseNameAnnotationSet, podDistruptionBudgetHelmReleaseNamespaceAnnotationSet, podDistruptionBudgetManagedByLabelSet, podDistruptionBudgetAmazonManagedLabelRemoved, err := ConfigMapImportedIntoHelm(ctx, clientSet, "kube-system", "coredns")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking CoreDns pod disruption budget to Helm",
			fmt.Sprintf("Error checking CoreDns pod disruption budget to Helm: %s", err),
		)
		return
	}

	model.CorednsPodDistruptionBudgetLabelHelmReleaseNameSet = basetypes.NewBoolValue(podDistruptionBudgetHelmReleaseNameAnnotationSet)
	model.CorednsPodDistruptionBudgetLabelHelmReleaseNamespaceSet = basetypes.NewBoolValue(podDistruptionBudgetHelmReleaseNamespaceAnnotationSet)
	model.CorednsPodDistruptionBudgetLabelManagedBySet = basetypes.NewBoolValue(podDistruptionBudgetManagedByLabelSet)
	model.CorednsPodDistruptionBudgetLabelAmazonManagedRemoved = basetypes.NewBoolValue(podDistruptionBudgetAmazonManagedLabelRemoved)

	model.ImportCorednsToHelm = basetypes.NewBoolValue(importCorednsToHelm && (deploymentHelmReleaseNameAnnotationSet && deploymentHelmReleaseNamespaceAnnotationSet && deploymentManagedByLabelSet && deploymentAmazonManagedLabelRemoved && serviceHelmReleaseNameAnnotationSet && serviceHelmReleaseNamespaceAnnotationSet && serviceManagedByLabelSet && serviceAmazonManagedLabelRemoved && serviceAccountHelmReleaseNameAnnotationSet && serviceAccountHelmReleaseNamespaceAnnotationSet && serviceAccountManagedByLabelSet && serviceAccountAmazonManagedLabelRemoved && configMapHelmReleaseNameAnnotationSet && configMapHelmReleaseNamespaceAnnotationSet && configMapManagedByLabelSet && configMapAmazonManagedLabelRemoved && podDistruptionBudgetHelmReleaseNameAnnotationSet && podDistruptionBudgetHelmReleaseNamespaceAnnotationSet && podDistruptionBudgetManagedByLabelSet && podDistruptionBudgetAmazonManagedLabelRemoved))

	model.ID = basetypes.NewStringValue(r.provider.Host)

	// Finally, set the state
	tflog.Debug(ctx, "Storing job info into the state")
	res.Diagnostics.Append(res.State.Set(ctx, model)...)
}

func (r *JobResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// NO-OP: Returning no error is enough for the framework to remove the resource from state.
	tflog.Debug(ctx, "Removing job from state")
}

func (r *JobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
