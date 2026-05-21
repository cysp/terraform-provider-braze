package provider

import (
	"context"
	"errors"
	"net/http"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*brazeEmailTemplateResource)(nil)
	_ resource.ResourceWithConfigure   = (*brazeEmailTemplateResource)(nil)
	_ resource.ResourceWithIdentity    = (*brazeEmailTemplateResource)(nil)
	_ resource.ResourceWithImportState = (*brazeEmailTemplateResource)(nil)
)

//nolint:ireturn
func NewBrazeEmailTemplateResource() resource.Resource {
	return &brazeEmailTemplateResource{}
}

type brazeEmailTemplateResource struct {
	providerData brazeProviderData
}

func (r *brazeEmailTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_email_template"
}

func (r *brazeEmailTemplateResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = BrazeEmailTemplateResourceIdentitySchema()
}

func (r *brazeEmailTemplateResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = BrazeEmailTemplateResourceSchema(ctx)
}

func (r *brazeEmailTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	SetProviderDataFromResourceConfigureRequest(req, &r.providerData)
}

func (r *brazeEmailTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("id"), req, resp)
}

func (r *brazeEmailTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan brazeEmailTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createRequest := plan.ToCreateEmailTemplateRequest()

	createResponse, createErr := r.providerData.client.CreateEmailTemplate(ctx, &createRequest)

	tflog.Info(ctx, "braze_email_template.create", map[string]any{
		"request":  createRequest,
		"response": createResponse,
		"err":      createErr,
	})

	if createResponse == nil || createErr != nil {
		resp.Diagnostics.AddError("Failed to create Email Template", detailFromError(createErr))

		return
	}

	emailTemplateID := createResponse.EmailTemplateID

	resp.Diagnostics.Append(setIdentity(ctx, resp.Identity, &resp.State, emailTemplateID)...)

	getParams := brazeclient.GetEmailTemplateInfoParams{
		EmailTemplateID: emailTemplateID,
	}

	getResponse, getErr := r.providerData.client.GetEmailTemplateInfo(ctx, getParams)

	tflog.Info(ctx, "braze_email_template.create.get", map[string]any{
		"params":   getParams,
		"response": getResponse,
		"err":      getErr,
	})

	if getResponse == nil || getErr != nil {
		var ersc *brazeclient.ErrorResponseStatusCode
		if errors.As(getErr, &ersc) && ersc.StatusCode == http.StatusNotFound {
			resp.Diagnostics.AddError("Email Template not found after creation", ersc.Error())

			return
		}

		resp.Diagnostics.AddError("Failed to retrieve Email Template after creation", detailFromError(getErr))

		return
	}

	data := NewBrazeEmailTemplateModelFromGetEmailTemplateInfoResponse(*getResponse)

	resp.Diagnostics.Append(setIdentityAndState(ctx, resp.Identity, &resp.State, emailTemplateID, &data)...)
}

func (r *brazeEmailTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state brazeEmailTemplateModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	getParams := brazeclient.GetEmailTemplateInfoParams{
		EmailTemplateID: state.ID.ValueString(),
	}

	getResponse, getErr := r.providerData.client.GetEmailTemplateInfo(ctx, getParams)

	tflog.Info(ctx, "braze_email_template.read", map[string]any{
		"params":   getParams,
		"response": getResponse,
		"err":      getErr,
	})

	if getResponse == nil || getErr != nil {
		var ersc *brazeclient.ErrorResponseStatusCode
		if errors.As(getErr, &ersc) && ersc.StatusCode == http.StatusNotFound {
			resp.Diagnostics.AddWarning("Email Template not found", ersc.Error())
			resp.State.RemoveResource(ctx)

			return
		}

		resp.Diagnostics.AddError("Failed to read Email Template", detailFromError(getErr))

		return
	}

	emailTemplateID := getResponse.EmailTemplateID
	data := NewBrazeEmailTemplateModelFromGetEmailTemplateInfoResponse(*getResponse)

	resp.Diagnostics.Append(setIdentityAndState(ctx, resp.Identity, &resp.State, emailTemplateID, &data)...)
}

func (r *brazeEmailTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state brazeEmailTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest := plan.ToUpdateEmailTemplateRequest()

	updateResponse, updateErr := r.providerData.client.UpdateEmailTemplate(ctx, &updateRequest)

	tflog.Info(ctx, "braze_email_template.update", map[string]any{
		"request":  updateRequest,
		"response": updateResponse,
		"err":      updateErr,
	})

	if updateResponse == nil || updateErr != nil {
		resp.Diagnostics.AddError("Failed to update Email Template", detailFromError(updateErr))

		return
	}

	emailTemplateID := state.ID.ValueString()

	resp.Diagnostics.Append(setIdentity(ctx, resp.Identity, &resp.State, emailTemplateID)...)

	getParams := brazeclient.GetEmailTemplateInfoParams{
		EmailTemplateID: emailTemplateID,
	}

	getResponse, getErr := r.providerData.client.GetEmailTemplateInfo(ctx, getParams)

	tflog.Info(ctx, "braze_email_template.update.get", map[string]any{
		"params":   getParams,
		"response": getResponse,
		"err":      getErr,
	})

	if getResponse == nil || getErr != nil {
		var ersc *brazeclient.ErrorResponseStatusCode
		if errors.As(getErr, &ersc) && ersc.StatusCode == http.StatusNotFound {
			resp.Diagnostics.AddError("Email Template not found after update", ersc.Error())

			return
		}

		resp.Diagnostics.AddError("Failed to retrieve Email Template after update", detailFromError(getErr))

		return
	}

	data := NewBrazeEmailTemplateModelFromGetEmailTemplateInfoResponse(*getResponse)

	resp.Diagnostics.Append(setIdentityAndState(ctx, resp.Identity, &resp.State, emailTemplateID, &data)...)
}

func (r *brazeEmailTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state brazeEmailTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddWarning("Email Template not deleted", "Braze does not provide a delete API for email templates; resource removed from Terraform state only.")
}
