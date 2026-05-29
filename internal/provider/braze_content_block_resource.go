package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.Resource                = (*brazeContentBlockResource)(nil)
	_ resource.ResourceWithConfigure   = (*brazeContentBlockResource)(nil)
	_ resource.ResourceWithIdentity    = (*brazeContentBlockResource)(nil)
	_ resource.ResourceWithImportState = (*brazeContentBlockResource)(nil)
)

//nolint:ireturn
func NewBrazeContentBlockResource() resource.Resource {
	return &brazeContentBlockResource{}
}

type brazeContentBlockResource struct {
	providerData brazeProviderData
}

func (r *brazeContentBlockResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_content_block"
}

func (r *brazeContentBlockResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = BrazeContentBlockResourceIdentitySchema()
}

func (r *brazeContentBlockResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = BrazeContentBlockResourceSchema(ctx)
}

func (r *brazeContentBlockResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	SetProviderDataFromResourceConfigureRequest(req, &r.providerData)
}

func (r *brazeContentBlockResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("id"), req, resp)
}

func (r *brazeContentBlockResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan brazeContentBlockModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	data, err := r.providerData.contentBlocks.Create(ctx, plan)
	if err != nil {
		if isBrazeObjectNotFound(err) {
			resp.Diagnostics.AddError("Content Block not found after creation", detailFromError(err))
		} else {
			resp.Diagnostics.AddError("Failed to create Content Block", detailFromError(err))
		}

		return
	}

	resp.Diagnostics.Append(setIdentityAndState(ctx, resp.Identity, &resp.State, data.ID.ValueString(), &data)...)
}

func (r *brazeContentBlockResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state brazeContentBlockModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	data, err := r.providerData.contentBlocks.Read(ctx, state.ID.ValueString())
	if err != nil {
		if isBrazeObjectNotFound(err) {
			resp.Diagnostics.AddWarning("Content Block not found", detailFromError(err))
			resp.State.RemoveResource(ctx)

			return
		}

		resp.Diagnostics.AddError("Failed to read Content Block", detailFromError(err))

		return
	}

	resp.Diagnostics.Append(setIdentityAndState(ctx, resp.Identity, &resp.State, data.ID.ValueString(), &data)...)
}

func (r *brazeContentBlockResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state brazeContentBlockModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	data, err := r.providerData.contentBlocks.Update(ctx, plan)
	if err != nil {
		if isBrazeObjectNotFound(err) {
			resp.Diagnostics.AddError("Content Block not found after update", detailFromError(err))
		} else {
			resp.Diagnostics.AddError("Failed to update Content Block", detailFromError(err))
		}

		return
	}

	resp.Diagnostics.Append(setIdentityAndState(ctx, resp.Identity, &resp.State, data.ID.ValueString(), &data)...)
}

func (r *brazeContentBlockResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state brazeContentBlockModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddWarning("Content Block not deleted", "Braze does not provide a delete API for content blocks; resource removed from Terraform state only.")
}
