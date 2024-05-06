package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type jobResource struct{}

var _ resource.Resource = (*jobResource)(nil)

func NewJobResource() resource.Resource {
	return &jobResource{}
}

func (r *jobResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job"
}

func (r *jobResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Cleans an EKS cluster of default AWS-CNI, Kube-Proxy and imports CoreDNS deployment " +
			"and service into Helm. By importing CoreDNS into Helm, we don't loose DNS at any point and we " +
			"can manage CoreDNS using a Helm chart.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: `ID of the job. This is the same values as the endpoint.`,
				Computed:    true,
			},

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

			"request_timeout_ms": schema.Int64Attribute{
				Description: "The request timeout in milliseconds.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(10),
				},
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

func (r *jobResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Debug(ctx, "Creating clean EKS resource")

	// Load entire configuration into the model
	var model jobResourceModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Loaded job configuration", map[string]interface{}{
		"jobConfig": fmt.Sprintf("%+v", model),
	})

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
		res.Diagnostics.AddError(
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
		res.Diagnostics.AddError(
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

	model.ID = model.Endpoint

	clientset, err := GetClient(endpoint, requestTimeout, insecure, caCertificate, token, clientCertificate, clientKey)
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

func (r *jobResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	// NO-OP: all there is to read is in the State, and response is already populated with that.
	tflog.Debug(ctx, "Reading job from state")

	// Load entire configuration into the model
	var model jobResourceModel
	res.Diagnostics.Append(req.State.Get(ctx, &model)...)

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
		res.Diagnostics.AddError(
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
		res.Diagnostics.AddError(
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

	clientset, err := GetClient(endpoint, requestTimeout, insecure, caCertificate, token, clientCertificate, clientKey)
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

func (r *jobResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Debug(ctx, "Updating job")

	// Load entire configuration into the model
	var model jobResourceModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Loaded job configuration", map[string]interface{}{
		"jobConfig": fmt.Sprintf("%+v", model),
	})

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
		res.Diagnostics.AddError(
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
		res.Diagnostics.AddError(
			"Token or Client Certificate and Client Key for Kubernetes Cluster must be specified",
			fmt.Sprintf("Token or Client Certificate and Client Key for Kubernetes Cluster must be specified"),
		)
		return
	}

	requestTimeout := int64(0)
	if !(model.RequestTimeout.IsNull() || model.RequestTimeout.IsUnknown()) {
		requestTimeout = model.RequestTimeout.ValueInt64()
	}

	endpoint := model.Endpoint.ValueString()

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

	clientset, err := GetClient(endpoint, requestTimeout, insecure, caCertificate, token, clientCertificate, clientKey)
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

func (r *jobResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// NO-OP: Returning no error is enough for the framework to remove the resource from state.
	tflog.Debug(ctx, "Removing job from state")
}
