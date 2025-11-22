package provider

import (
	"context"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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

	params := brazeclient.ListContentBlocksParams{
		Limit: brazeclient.NewOptInt(int(req.Limit)),
	}
	paramsDiags := diag.Diagnostics{}

	if !config.ModifiedAfter.IsNull() {
		modifiedAfter, modifiedAfterDiags := config.ModifiedAfter.ValueRFC3339Time()
		paramsDiags.Append(modifiedAfterDiags...)

		params.ModifiedAfter = brazeclient.NewOptDateTime(modifiedAfter)
	}

	if !config.ModifiedBefore.IsNull() {
		modifiedBefore, modifiedBeforeDiags := config.ModifiedBefore.ValueRFC3339Time()
		paramsDiags.Append(modifiedBeforeDiags...)

		params.ModifiedBefore = brazeclient.NewOptDateTime(modifiedBefore)
	}

	if paramsDiags.HasError() {
		resp.Results = list.ListResultsStreamDiagnostics(paramsDiags)

		return
	}

	resp.Results = func(yield func(list.ListResult) bool) {
		listResponse, listErr := r.providerData.client.ListContentBlocks(ctx, params)

		tflog.Info(ctx, "braze_content_block.list", map[string]any{
			"params":   params,
			"response": listResponse,
			"err":      listErr,
		})

		if listErr != nil {
			result := req.NewListResult(ctx)
			result.Diagnostics.AddError("Failed to list content blocks", listErr.Error())

			yield(result)

			return
		}

		for _, block := range listResponse.GetContentBlocks() {
			result := req.NewListResult(ctx)

			result.Diagnostics.Append(result.Identity.SetAttribute(ctx, path.Root("id"), block.GetContentBlockID())...)

			result.DisplayName = block.GetName()

			if req.IncludeResource {
				params := brazeclient.GetContentBlockInfoParams{
					ContentBlockID: block.GetContentBlockID(),
				}

				getResponse, getErr := r.providerData.client.GetContentBlockInfo(ctx, params)

				tflog.Info(ctx, "braze_content_block.list.get", map[string]any{
					"params":   params,
					"response": getResponse,
					"err":      getErr,
				})

				if getResponse == nil || getErr != nil {
					result.Diagnostics.AddError("Failed to get content block", detailFromError(getErr))

					if !yield(result) {
						return
					}
				}

				data := NewBrazeContentBlockModelFromGetContentBlockInfoResponse(*getResponse)

				result.Diagnostics.Append(result.Resource.Set(ctx, data)...)
			}

			if !yield(result) {
				return
			}
		}
	}
}
