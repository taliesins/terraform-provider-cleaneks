package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

type jobResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Endpoint          types.String `tfsdk:"endpoint"`
	Insecure          types.Bool   `tfsdk:"insecure"`
	CaCertificate     types.String `tfsdk:"ca_cert_pem"`
	ClientCertificate types.String `tfsdk:"client_cert_pem"`
	ClientKey         types.String `tfsdk:"client_key_pem"`
	Token             types.String `tfsdk:"token"`

	RemoveAwsCni             types.Bool `tfsdk:"remove_aws_cni"`
	RemoveKubeProxy          types.Bool `tfsdk:"remove_kube_proxy"`
	ImportCorednsToHelm      types.Bool `tfsdk:"import_coredns_to_helm"`
	AwsCniDaemonsetExists    types.Bool `tfsdk:"aws_cni_daemonset_exists"`
	KubeProxyDaemonsetExists types.Bool `tfsdk:"kube_proxy_daemonset_exists"`

	RequestTimeout types.Int64 `tfsdk:"request_timeout_ms"`

	CorednsDeploymentLabelHelmReleaseNameSet      types.Bool `tfsdk:"coredns_deployment_label_helm_release_name_set"`
	CorednsDeploymentLabelHelmReleaseNamespaceSet types.Bool `tfsdk:"coredns_deployment_label_helm_release_namespace_set"`
	CorednsDeploymentLabelManagedBySet            types.Bool `tfsdk:"coredns_deployment_label_managed_by_set"`
	CorednsDeploymentLabelAmazonManagedRemoved    types.Bool `tfsdk:"coredns_deployment_label_amazon_managed_removed"`

	CorednsServiceLabelHelmReleaseNameSet      types.Bool `tfsdk:"coredns_service_label_helm_release_name_set"`
	CorednsServiceLabelHelmReleaseNamespaceSet types.Bool `tfsdk:"coredns_service_label_helm_release_namespace_set"`
	CorednsServiceLabelManagedBySet            types.Bool `tfsdk:"coredns_service_label_managed_by_set"`
	CorednsServiceLabelAmazonManagedRemoved    types.Bool `tfsdk:"coredns_service_label_amazon_managed_removed"`
}
