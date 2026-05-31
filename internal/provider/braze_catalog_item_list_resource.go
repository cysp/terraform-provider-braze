package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

//nolint:ireturn
func NewBrazeCatalogItemListResource() list.ListResource {
	return &brazeCatalogItemListResource{}
}

type brazeCatalogItemListResource struct {
	providerData brazeProviderData
}

type brazeCatalogItemListResourceConfig struct {
	CatalogName types.String `tfsdk:"catalog_name"`
}

var (
	_ list.ListResource              = (*brazeCatalogItemListResource)(nil)
	_ list.ListResourceWithConfigure = (*brazeCatalogItemListResource)(nil)
)

func (r *brazeCatalogItemListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_catalog_item"
}

func (r *brazeCatalogItemListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"catalog_name": schema.StringAttribute{
				Description: "The catalog to list items from.",
				Required:    true,
			},
		},
	}
}

func (r *brazeCatalogItemListResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	SetProviderDataFromResourceConfigureRequest(req, &r.providerData)
}

func (r *brazeCatalogItemListResource) List(ctx context.Context, req list.ListRequest, resp *list.ListResultsStream) {
	config := brazeCatalogItemListResourceConfig{}

	configDiags := req.Config.Get(ctx, &config)
	if configDiags.HasError() {
		resp.Results = list.ListResultsStreamDiagnostics(configDiags)

		return
	}

	if req.Limit <= 0 {
		resp.Results = func(_ func(list.ListResult) bool) {}

		return
	}

	resp.Results = func(yield func(list.ListResult) bool) {
		entries, listErr := r.providerData.catalogItems.List(ctx, config.CatalogName.ValueString(), req.Limit)
		if listErr != nil {
			result := req.NewListResult(ctx)
			result.Diagnostics.AddError("Failed to list catalog items", detailFromError(listErr))

			yield(result)

			return
		}

		for _, entry := range entries {
			result := req.NewListResult(ctx)
			result.Diagnostics.Append(result.Identity.SetAttribute(ctx, path.Root("id"), entry.ID)...)
			result.DisplayName = entry.DisplayName

			if req.IncludeResource {
				if entry.ResourceErr != nil {
					result.Diagnostics.AddError("Failed to get catalog item", detailFromError(entry.ResourceErr))
				} else {
					result.Diagnostics.Append(result.Resource.Set(ctx, *entry.Resource)...)
				}
			}

			if !yield(result) {
				return
			}
		}
	}
}
