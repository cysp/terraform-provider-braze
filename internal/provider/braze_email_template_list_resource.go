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
func NewBrazeEmailTemplateListResource() list.ListResource {
	return &brazeEmailTemplateListResource{}
}

type brazeEmailTemplateListResource struct {
	providerData brazeProviderData
}

var (
	_ list.ListResource              = (*brazeEmailTemplateListResource)(nil)
	_ list.ListResourceWithConfigure = (*brazeEmailTemplateListResource)(nil)
)

type brazeEmailTemplateListResourceConfig struct {
	ModifiedAfter  timetypes.RFC3339 `tfsdk:"modified_after"`
	ModifiedBefore timetypes.RFC3339 `tfsdk:"modified_before"`
}

const brazeEmailTemplateListPageLimit = 100

func (r *brazeEmailTemplateListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_email_template"
}

func (r *brazeEmailTemplateListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"modified_after": schema.StringAttribute{
				Description: "Filter to email templates modified after this date/time.",
				CustomType:  timetypes.RFC3339Type{},
				Optional:    true,
			},
			"modified_before": schema.StringAttribute{
				Description: "Filter to email templates modified before this date/time.",
				CustomType:  timetypes.RFC3339Type{},
				Optional:    true,
			},
		},
	}
}

func (r *brazeEmailTemplateListResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	SetProviderDataFromResourceConfigureRequest(req, &r.providerData)
}

func (r *brazeEmailTemplateListResource) List(ctx context.Context, req list.ListRequest, resp *list.ListResultsStream) {
	config := brazeEmailTemplateListResourceConfig{}

	configDiags := req.Config.Get(ctx, &config)
	if configDiags.HasError() {
		resp.Results = list.ListResultsStreamDiagnostics(configDiags)

		return
	}

	if req.Limit <= 0 {
		resp.Results = func(_ func(list.ListResult) bool) {}

		return
	}

	params := brazeclient.ListEmailTemplatesParams{
		Limit: brazeclient.NewOptInt(brazeEmailTemplateListPageLimit),
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
		r.listEmailTemplates(ctx, req, params, yield)
	}
}

func (r *brazeEmailTemplateListResource) listEmailTemplates(
	ctx context.Context,
	req list.ListRequest,
	baseParams brazeclient.ListEmailTemplatesParams,
	yield func(list.ListResult) bool,
) {
	offset := 0
	remaining := req.Limit

	for {
		params := baseParams
		if offset > 0 {
			params.Offset = brazeclient.NewOptInt(offset)
		}

		listResponse, listErr := r.providerData.client.ListEmailTemplates(ctx, params)

		tflog.Info(ctx, "braze_email_template.list", map[string]any{
			"params":   params,
			"response": listResponse,
			"err":      listErr,
		})

		if listErr != nil {
			result := req.NewListResult(ctx)
			result.Diagnostics.AddError("Failed to list email templates", listErr.Error())

			yield(result)

			return
		}

		emailTemplates := listResponse.GetTemplates()
		if !r.yieldEmailTemplateResults(ctx, req, emailTemplates, &remaining, yield) {
			return
		}

		if remaining <= 0 || len(emailTemplates) < brazeEmailTemplateListPageLimit {
			return
		}

		offset += brazeEmailTemplateListPageLimit
	}
}

func (r *brazeEmailTemplateListResource) yieldEmailTemplateResults(
	ctx context.Context,
	req list.ListRequest,
	emailTemplates []brazeclient.ListEmailTemplatesResponseTemplatesItem,
	remaining *int64,
	yield func(list.ListResult) bool,
) bool {
	for _, template := range emailTemplates {
		if *remaining <= 0 {
			return false
		}

		result := req.NewListResult(ctx)

		result.Diagnostics.Append(result.Identity.SetAttribute(ctx, path.Root("id"), template.GetEmailTemplateID())...)

		result.DisplayName = template.GetTemplateName()

		if req.IncludeResource {
			r.setEmailTemplateResource(ctx, template, &result)
		}

		if !yield(result) {
			return false
		}

		*remaining--
	}

	return true
}

func (r *brazeEmailTemplateListResource) setEmailTemplateResource(
	ctx context.Context,
	template brazeclient.ListEmailTemplatesResponseTemplatesItem,
	result *list.ListResult,
) {
	params := brazeclient.GetEmailTemplateInfoParams{
		EmailTemplateID: template.GetEmailTemplateID(),
	}

	getResponse, getErr := r.providerData.client.GetEmailTemplateInfo(ctx, params)

	tflog.Info(ctx, "braze_email_template.list.get", map[string]any{
		"params":   params,
		"response": getResponse,
		"err":      getErr,
	})

	if getResponse == nil || getErr != nil {
		result.Diagnostics.AddError("Failed to get email template", detailFromError(getErr))

		return
	}

	data := NewBrazeEmailTemplateModelFromGetEmailTemplateInfoResponse(*getResponse)

	result.Diagnostics.Append(result.Resource.Set(ctx, data)...)
}
