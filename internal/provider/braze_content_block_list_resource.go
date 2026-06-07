package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

//nolint:ireturn
func NewBrazeContentBlockListResource() list.ListResource {
	return &brazeContentBlockListResource{}
}

type brazeContentBlockListResource struct {
	providerData brazeProviderData
}

var (
	_ list.ListResource              = (*brazeContentBlockListResource)(nil)
	_ list.ListResourceWithConfigure = (*brazeContentBlockListResource)(nil)
)

type brazeContentBlockListResourceConfig struct {
	ModifiedAfter  timetypes.RFC3339 `tfsdk:"modified_after"`
	ModifiedBefore timetypes.RFC3339 `tfsdk:"modified_before"`
}

func (r *brazeContentBlockListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_content_block"
}

func (r *brazeContentBlockListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"modified_after": schema.StringAttribute{
				Description: "Filter to content blocks modified after this date/time.",
				CustomType:  timetypes.RFC3339Type{},
				Optional:    true,
			},
			"modified_before": schema.StringAttribute{
				Description: "Filter to content blocks modified before this date/time.",
				CustomType:  timetypes.RFC3339Type{},
				Optional:    true,
			},
		},
	}
}

func (r *brazeContentBlockListResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	SetProviderDataFromResourceConfigureRequest(req, &r.providerData)
}

func (r *brazeContentBlockListResource) List(ctx context.Context, req list.ListRequest, resp *list.ListResultsStream) {
	config := brazeContentBlockListResourceConfig{}

	configDiags := req.Config.Get(ctx, &config)
	if configDiags.HasError() {
		resp.Results = list.ListResultsStreamDiagnostics(configDiags)

		return
	}

	if req.Limit <= 0 {
		resp.Results = emptyBrazeObjectListResults

		return
	}

	query := brazeObjectListQuery{
		Limit:           req.Limit,
		IncludeResource: req.IncludeResource,
	}
	paramsDiags := diag.Diagnostics{}

	if !config.ModifiedAfter.IsNull() {
		modifiedAfter, modifiedAfterDiags := config.ModifiedAfter.ValueRFC3339Time()
		paramsDiags.Append(modifiedAfterDiags...)

		query.ModifiedAfter = &modifiedAfter
	}

	if !config.ModifiedBefore.IsNull() {
		modifiedBefore, modifiedBeforeDiags := config.ModifiedBefore.ValueRFC3339Time()
		paramsDiags.Append(modifiedBeforeDiags...)

		query.ModifiedBefore = &modifiedBefore
	}

	if paramsDiags.HasError() {
		resp.Results = list.ListResultsStreamDiagnostics(paramsDiags)

		return
	}

	resp.Results = func(yield func(list.ListResult) bool) {
		entries, listErr := r.providerData.contentBlocks.List(ctx, query)
		if listErr != nil {
			streamBrazeObjectListError(ctx, req, "Failed to list content blocks", listErr, yield)

			return
		}

		streamBrazeObjectListEntries(ctx, req, entries, "id", "Failed to get content block", yield)
	}
}
