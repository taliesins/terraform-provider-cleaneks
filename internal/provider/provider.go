package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type cleanEksProvider struct{}

var _ provider.Provider = (*cleanEksProvider)(nil)

func New() provider.Provider {
	return &cleanEksProvider{}
}

func (p *cleanEksProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cleaneks"
}

func (p *cleanEksProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A provider to bootstrap an EKS cluster by removing AWS CNI and Kube-Proxy. It will also add the required annotations and labels to CoreDNS so that Helm can manage CoreDNS. It will also drop managed by AWS labels from CoreDNS deployment and service.",
		Attributes:  map[string]schema.Attribute{},
	}
}

func (p *cleanEksProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
}

func (p *cleanEksProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewJobResource,
	}
}

func (p *cleanEksProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
