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
	schema, schemaDiags := BrazeContentBlockResourceSchema(ctx)
	resp.Diagnostics.Append(schemaDiags...)

	resp.Schema = schema
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

	createRequest := plan.ToCreateContentBlockRequest()

	createResponse, createErr := r.providerData.client.CreateContentBlock(ctx, &createRequest)

	tflog.Info(ctx, "braze_content_block.create", map[string]any{
		"request":  createRequest,
		"response": createResponse,
		"err":      createErr,
	})

	if createResponse == nil || createErr != nil {
		resp.Diagnostics.AddError("Failed to create Content Block", detailFromError(createErr))

		return
	}

	contentBlockID := createResponse.GetContentBlockID()

	resp.Diagnostics.Append(setIdentity(ctx, resp.Identity, &resp.State, contentBlockID)...)

	getParams := brazeclient.GetContentBlockInfoParams{
		ContentBlockID: contentBlockID,
	}

	getResponse, getErr := r.providerData.client.GetContentBlockInfo(ctx, getParams)

	tflog.Info(ctx, "braze_content_block.create.get", map[string]any{
		"params":   getParams,
		"response": getResponse,
		"err":      getErr,
	})

	if getResponse == nil || getErr != nil {
		var ersc *brazeclient.ErrorResponseStatusCode
		if errors.As(getErr, &ersc) && ersc.StatusCode == http.StatusNotFound {
			resp.Diagnostics.AddError("Content Block not found after creation", ersc.Error())

			return
		}

		resp.Diagnostics.AddError("Failed to retrieve Content Block after creation", detailFromError(getErr))

		return
	}

	contentBlockID = getResponse.GetContentBlockID()
	data := NewBrazeContentBlockModelFromGetContentBlockInfoResponse(*getResponse)

	resp.Diagnostics.Append(setIdentityAndState(ctx, resp.Identity, &resp.State, contentBlockID, &data)...)
}

func (r *brazeContentBlockResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state brazeContentBlockModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	getParams := brazeclient.GetContentBlockInfoParams{
		ContentBlockID: state.ID.ValueString(),
	}

	getResponse, getErr := r.providerData.client.GetContentBlockInfo(ctx, getParams)

	tflog.Info(ctx, "braze_content_block.read", map[string]any{
		"params":   getParams,
		"response": getResponse,
		"err":      getErr,
	})

	if getResponse == nil || getErr != nil {
		var ersc *brazeclient.ErrorResponseStatusCode
		if errors.As(getErr, &ersc) && ersc.StatusCode == http.StatusNotFound {
			resp.Diagnostics.AddWarning("Content Block not found", ersc.Error())
			resp.State.RemoveResource(ctx)

			return
		}

		resp.Diagnostics.AddError("Failed to read Content Block", detailFromError(getErr))

		return
	}

	contentBlockID := getResponse.GetContentBlockID()
	data := NewBrazeContentBlockModelFromGetContentBlockInfoResponse(*getResponse)

	resp.Diagnostics.Append(setIdentityAndState(ctx, resp.Identity, &resp.State, contentBlockID, &data)...)
}

func (r *brazeContentBlockResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state brazeContentBlockModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest := plan.ToUpdateContentBlockRequest()

	updateResponse, updateErr := r.providerData.client.UpdateContentBlock(ctx, &updateRequest)

	tflog.Info(ctx, "braze_content_block.update", map[string]any{
		"request":  updateRequest,
		"response": updateResponse,
		"err":      updateErr,
	})

	if updateResponse == nil || updateErr != nil {
		resp.Diagnostics.AddError("Failed to update Content Block", detailFromError(updateErr))

		return
	}

	contentBlockID := updateResponse.GetContentBlockID()

	resp.Diagnostics.Append(setIdentity(ctx, resp.Identity, &resp.State, contentBlockID)...)

	getParams := brazeclient.GetContentBlockInfoParams{
		ContentBlockID: contentBlockID,
	}

	getResponse, getErr := r.providerData.client.GetContentBlockInfo(ctx, getParams)

	tflog.Info(ctx, "braze_content_block.update.get", map[string]any{
		"params":   getParams,
		"response": getResponse,
		"err":      getErr,
	})

	if getResponse == nil || getErr != nil {
		var ersc *brazeclient.ErrorResponseStatusCode
		if errors.As(getErr, &ersc) && ersc.StatusCode == http.StatusNotFound {
			resp.Diagnostics.AddError("Content Block not found after update", ersc.Error())

			return
		}

		resp.Diagnostics.AddError("Failed to retrieve Content Block after update", detailFromError(getErr))

		return
	}

	contentBlockID = getResponse.GetContentBlockID()
	data := NewBrazeContentBlockModelFromGetContentBlockInfoResponse(*getResponse)

	resp.Diagnostics.Append(setIdentityAndState(ctx, resp.Identity, &resp.State, contentBlockID, &data)...)
}

func (r *brazeContentBlockResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state brazeContentBlockModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddWarning("Content Block not deleted", "Braze does not provide a delete API for content blocks; resource removed from Terraform state only.")
}
