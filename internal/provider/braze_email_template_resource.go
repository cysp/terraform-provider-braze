package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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

	data, err := r.providerData.emailTemplates.Create(ctx, plan)
	if err != nil {
		if isBrazeObjectNotFound(err) {
			resp.Diagnostics.AddError("Email Template not found after creation", detailFromError(err))
		} else {
			resp.Diagnostics.AddError("Failed to create Email Template", detailFromError(err))
		}

		return
	}

	resp.Diagnostics.Append(setIdentityAndState(ctx, resp.Identity, &resp.State, data.ID.ValueString(), &data)...)
}

func (r *brazeEmailTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state brazeEmailTemplateModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	data, err := r.providerData.emailTemplates.Read(ctx, state.ID.ValueString())
	if err != nil {
		if isBrazeObjectNotFound(err) {
			resp.Diagnostics.AddWarning("Email Template not found", detailFromError(err))
			resp.State.RemoveResource(ctx)

			return
		}

		resp.Diagnostics.AddError("Failed to read Email Template", detailFromError(err))

		return
	}

	resp.Diagnostics.Append(setIdentityAndState(ctx, resp.Identity, &resp.State, data.ID.ValueString(), &data)...)
}

func (r *brazeEmailTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan brazeEmailTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	data, err := r.providerData.emailTemplates.Update(ctx, plan)
	if err != nil {
		if isBrazeObjectNotFound(err) {
			resp.Diagnostics.AddError("Email Template not found after update", detailFromError(err))
		} else {
			resp.Diagnostics.AddError("Failed to update Email Template", detailFromError(err))
		}

		return
	}

	resp.Diagnostics.Append(setIdentityAndState(ctx, resp.Identity, &resp.State, data.ID.ValueString(), &data)...)
}

func (r *brazeEmailTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state brazeEmailTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddWarning("Email Template not deleted", "Braze does not provide a delete API for email templates; resource removed from Terraform state only.")
}
