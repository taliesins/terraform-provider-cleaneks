package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"k8s.io/client-go/kubernetes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &JobResource{}
var _ resource.ResourceWithImportState = &JobResource{}

func NewJobResource() resource.Resource {
	return &JobResource{}
}

type JobResource struct {
	host      string
	clientset *kubernetes.Clientset
}

type JobResourceModel struct {
	ID types.String `tfsdk:"id"`

	RemoveAwsCni             types.Bool `tfsdk:"remove_aws_cni"`
	RemoveKubeProxy          types.Bool `tfsdk:"remove_kube_proxy"`
	ImportCorednsToHelm      types.Bool `tfsdk:"import_coredns_to_helm"`
	AwsCniDaemonsetExists    types.Bool `tfsdk:"aws_cni_daemonset_exists"`
	KubeProxyDaemonsetExists types.Bool `tfsdk:"kube_proxy_daemonset_exists"`

	CorednsDeploymentLabelHelmReleaseNameSet      types.Bool `tfsdk:"coredns_deployment_label_helm_release_name_set"`
	CorednsDeploymentLabelHelmReleaseNamespaceSet types.Bool `tfsdk:"coredns_deployment_label_helm_release_namespace_set"`
	CorednsDeploymentLabelManagedBySet            types.Bool `tfsdk:"coredns_deployment_label_managed_by_set"`
	CorednsDeploymentLabelAmazonManagedRemoved    types.Bool `tfsdk:"coredns_deployment_label_amazon_managed_removed"`

	CorednsServiceLabelHelmReleaseNameSet      types.Bool `tfsdk:"coredns_service_label_helm_release_name_set"`
	CorednsServiceLabelHelmReleaseNamespaceSet types.Bool `tfsdk:"coredns_service_label_helm_release_namespace_set"`
	CorednsServiceLabelManagedBySet            types.Bool `tfsdk:"coredns_service_label_managed_by_set"`
	CorednsServiceLabelAmazonManagedRemoved    types.Bool `tfsdk:"coredns_service_label_amazon_managed_removed"`
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
				Description: `ID of the job. This is the same values as the endpoint.`,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"remove_aws_cni": schema.BoolAttribute{
				Description: "Remove AWS-CNI from EKS cluster",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},

			"remove_kube_proxy": schema.BoolAttribute{
				Description: "Remove Kube-Proxy from EKS cluster",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},

			"import_coredns_to_helm": schema.BoolAttribute{
				Description: "Add helm attributes to CoreDns service and deployment, so that it can be managed by Helm.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},

			"aws_cni_daemonset_exists": schema.BoolAttribute{
				Description: `Does AWS CNI daemonset exist.`,
				Computed:    true,
			},

			"kube_proxy_daemonset_exists": schema.BoolAttribute{
				Description: `Does Kube-Proxy daemonset exist.`,
				Computed:    true,
			},

			"coredns_deployment_label_helm_release_name_set": schema.BoolAttribute{
				Description: `Does CoreDNS deployment have label meta.helm.sh/release-name with value of coredns.`,
				Computed:    true,
			},

			"coredns_deployment_label_helm_release_namespace_set": schema.BoolAttribute{
				Description: `Does CoreDNS deployment have label meta.helm.sh/release-namespace with value of kube-system.`,
				Computed:    true,
			},

			"coredns_deployment_label_managed_by_set": schema.BoolAttribute{
				Description: `Does CoreDNS deployment have label app.kubernetes.io/managed-by with value of Helm.`,
				Computed:    true,
			},

			"coredns_deployment_label_amazon_managed_removed": schema.BoolAttribute{
				Description: `Is label eks.amazonaws.com/component removed.`,
				Computed:    true,
			},

			"coredns_service_label_helm_release_name_set": schema.BoolAttribute{
				Description: `Does CoreDNS service have label meta.helm.sh/release-name with value of coredns.`,
				Computed:    true,
			},

			"coredns_service_label_helm_release_namespace_set": schema.BoolAttribute{
				Description: `Does CoreDNS service have label meta.helm.sh/release-namespace with value of kube-system.`,
				Computed:    true,
			},

			"coredns_service_label_managed_by_set": schema.BoolAttribute{
				Description: `Does CoreDNS service have label app.kubernetes.io/managed-by with value of Helm.`,
				Computed:    true,
			},

			"coredns_service_label_amazon_managed_removed": schema.BoolAttribute{
				Description: `Is label eks.amazonaws.com/component removed.`,
				Computed:    true,
			},
		},
	}
}

func (r *JobResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	cleanEksProviderResourceData, ok := req.ProviderData.(*CleanEksProviderResourceData)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *CleanEksProviderResourceData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.clientset = cleanEksProviderResourceData.ClientSet
	r.host = cleanEksProviderResourceData.Config.Host
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

	clientset := r.clientset

	var err error

	removeAwsCni := true
	if !(model.RemoveAwsCni.IsNull() || model.RemoveAwsCni.IsUnknown()) {
		removeAwsCni = model.RemoveAwsCni.ValueBool()
	}

	removeKubeProxy := true
	if !(model.RemoveKubeProxy.IsNull() || model.RemoveKubeProxy.IsUnknown()) {
		removeKubeProxy = model.RemoveKubeProxy.ValueBool()
	}

	importCorednsToHelm := true
	if !(model.ImportCorednsToHelm.IsNull() || model.ImportCorednsToHelm.IsUnknown()) {
		importCorednsToHelm = model.ImportCorednsToHelm.ValueBool()
	}

	if removeAwsCni || removeKubeProxy || importCorednsToHelm {
		if removeAwsCni {
			_, err = DeleteDaemonset(ctx, clientset, "kube-system", "aws-node")
			if err != nil {
				res.Diagnostics.AddError(
					"Error removing AWS CNI",
					fmt.Sprintf("Error removing AWS CNI: %s", err),
				)
				return
			}
		}

		if removeKubeProxy {
			_, err = DeleteDaemonset(ctx, clientset, "kube-system", "kube-proxy")
			if err != nil {
				res.Diagnostics.AddError(
					"Error removing Kube Proxy",
					fmt.Sprintf("Error removing Kube Proxy: %s", err),
				)
				return
			}
		}

		if importCorednsToHelm {
			err = ImportDeploymentIntoHelm(ctx, clientset, "kube-system", "coredns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error importing CoreDns deployment to Helm",
					fmt.Sprintf("Error importing CoreDns deployment to Helm: %s", err),
				)
				return
			}

			err = ImportServiceIntoHelm(ctx, clientset, "kube-system", "kube-dns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error importing CoreDns service to Helm",
					fmt.Sprintf("Error importing CoreDns service to Helm: %s", err),
				)
				return
			}
		}
	}

	// Read kubernetes to populate model
	awsCniDaemonsetExists, err := DaemonsetExist(ctx, clientset, "kube-system", "aws-node")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for AWS CNI",
			fmt.Sprintf("Error checking daemonset for AWS CNI: %s", err),
		)
		return
	}
	model.AwsCniDaemonsetExists = basetypes.NewBoolValue(awsCniDaemonsetExists)

	kubeProxyDaemonsetExists, err := DaemonsetExist(ctx, clientset, "kube-system", "kube-proxy")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for Kube Proxy",
			fmt.Sprintf("Error checking for Kube Proxy: %s", err),
		)
		return
	}
	model.KubeProxyDaemonsetExists = basetypes.NewBoolValue(kubeProxyDaemonsetExists)

	deploymentHelmReleaseNameAnnotationSet, deploymentHelmReleaseNamespaceAnnotationSet, deploymentManagedByLabelSet, deploymentAmazonManagedLabelRemoved, err := DeploymentImportedIntoHelm(ctx, clientset, "kube-system", "coredns")
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

	serviceHelmReleaseNameAnnotationSet, serviceHelmReleaseNamespaceAnnotationSet, serviceManagedByLabelSet, serviceAmazonManagedLabelRemoved, err := ServiceImportedIntoHelm(ctx, clientset, "kube-system", "kube-dns")
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

	model.ID = basetypes.NewStringValue(r.host)

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

	clientset := r.clientset

	var err error

	if err != nil {
		res.Diagnostics.AddError(
			"Error making request",
			fmt.Sprintf("Error making request: %s", err),
		)
		return
	}

	// Read kubernetes to populate model

	awsCniDaemonsetExists, err := DaemonsetExist(ctx, clientset, "kube-system", "aws-node")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for AWS CNI",
			fmt.Sprintf("Error checking daemonset for AWS CNI: %s", err),
		)
		return
	}
	model.AwsCniDaemonsetExists = basetypes.NewBoolValue(awsCniDaemonsetExists)

	kubeProxyDaemonsetExists, err := DaemonsetExist(ctx, clientset, "kube-system", "kube-proxy")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for Kube Proxy",
			fmt.Sprintf("Error checking for Kube Proxy: %s", err),
		)
		return
	}
	model.KubeProxyDaemonsetExists = basetypes.NewBoolValue(kubeProxyDaemonsetExists)

	deploymentHelmReleaseNameAnnotationSet, deploymentHelmReleaseNamespaceAnnotationSet, deploymentManagedByLabelSet, deploymentAmazonManagedLabelRemoved, err := DeploymentImportedIntoHelm(ctx, clientset, "kube-system", "coredns")
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

	serviceHelmReleaseNameAnnotationSet, serviceHelmReleaseNamespaceAnnotationSet, serviceManagedByLabelSet, serviceAmazonManagedLabelRemoved, err := ServiceImportedIntoHelm(ctx, clientset, "kube-system", "kube-dns")
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

	clientset := r.clientset

	var err error

	removeAwsCni := true
	if !(model.RemoveAwsCni.IsNull() || model.RemoveAwsCni.IsUnknown()) {
		removeAwsCni = model.RemoveAwsCni.ValueBool()
	}

	removeKubeProxy := true
	if !(model.RemoveKubeProxy.IsNull() || model.RemoveKubeProxy.IsUnknown()) {
		removeKubeProxy = model.RemoveKubeProxy.ValueBool()
	}

	importCorednsToHelm := true
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
			_, err = DeleteDaemonset(ctx, clientset, "kube-system", "aws-node")
			if err != nil {
				res.Diagnostics.AddError(
					"Error removing AWS CNI",
					fmt.Sprintf("Error removing AWS CNI: %s", err),
				)
				return
			}
		}

		if removeKubeProxy {
			_, err = DeleteDaemonset(ctx, clientset, "kube-system", "kube-proxy")
			if err != nil {
				res.Diagnostics.AddError(
					"Error removing Kube Proxy",
					fmt.Sprintf("Error removing Kube Proxy: %s", err),
				)
				return
			}
		}

		if importCorednsToHelm {
			err = ImportDeploymentIntoHelm(ctx, clientset, "kube-system", "coredns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error importing CoreDns deployment to Helm",
					fmt.Sprintf("Error importing CoreDns deployment to Helm: %s", err),
				)
				return
			}

			err = ImportServiceIntoHelm(ctx, clientset, "kube-system", "kube-dns")
			if err != nil {
				res.Diagnostics.AddError(
					"Error importing CoreDns service to Helm",
					fmt.Sprintf("Error importing CoreDns service to Helm: %s", err),
				)
				return
			}
		}
	}

	// Read kubernetes to populate model

	awsCniDaemonsetExists, err := DaemonsetExist(ctx, clientset, "kube-system", "aws-node")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for AWS CNI",
			fmt.Sprintf("Error checking daemonset for AWS CNI: %s", err),
		)
		return
	}
	model.AwsCniDaemonsetExists = basetypes.NewBoolValue(awsCniDaemonsetExists)

	kubeProxyDaemonsetExists, err := DaemonsetExist(ctx, clientset, "kube-system", "kube-proxy")
	if err != nil {
		res.Diagnostics.AddError(
			"Error checking for Kube Proxy",
			fmt.Sprintf("Error checking for Kube Proxy: %s", err),
		)
		return
	}
	model.KubeProxyDaemonsetExists = basetypes.NewBoolValue(kubeProxyDaemonsetExists)

	deploymentHelmReleaseNameAnnotationSet, deploymentHelmReleaseNamespaceAnnotationSet, deploymentManagedByLabelSet, deploymentAmazonManagedLabelRemoved, err := DeploymentImportedIntoHelm(ctx, clientset, "kube-system", "coredns")
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

	serviceHelmReleaseNameAnnotationSet, serviceHelmReleaseNamespaceAnnotationSet, serviceManagedByLabelSet, serviceAmazonManagedLabelRemoved, err := ServiceImportedIntoHelm(ctx, clientset, "kube-system", "kube-dns")
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
