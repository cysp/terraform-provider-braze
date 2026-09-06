package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

//nolint:ireturn
func NewBrazeCatalogListResource() list.ListResource {
	return &brazeCatalogListResource{}
}

type brazeCatalogListResource struct {
	providerData brazeProviderData
}

var (
	_ list.ListResource              = (*brazeCatalogListResource)(nil)
	_ list.ListResourceWithConfigure = (*brazeCatalogListResource)(nil)
)

func (r *brazeCatalogListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_catalog"
}

func (r *brazeCatalogListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = schema.Schema{Description: "Discover catalogs in the configured Braze workspace. Results identify each catalog by name."}
}

func (r *brazeCatalogListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(setProviderData(req.ProviderData, &r.providerData)...)
}

func (r *brazeCatalogListResource) List(ctx context.Context, req list.ListRequest, resp *list.ListResultsStream) {
	if req.Limit <= 0 {
		resp.Results = emptyBrazeObjectListResults

		return
	}

	resp.Results = func(yield func(list.ListResult) bool) {
		streamBrazeObjectListEntries(ctx, req, r.providerData.catalogs.List(ctx), "name", "Failed to list catalogs", "Failed to get catalog", yield)
	}
}
