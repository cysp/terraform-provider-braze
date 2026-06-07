package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = (*brazeCatalogItemResource)(nil)
	_ resource.ResourceWithConfigure      = (*brazeCatalogItemResource)(nil)
	_ resource.ResourceWithIdentity       = (*brazeCatalogItemResource)(nil)
	_ resource.ResourceWithImportState    = (*brazeCatalogItemResource)(nil)
	_ resource.ResourceWithValidateConfig = (*brazeCatalogItemResource)(nil)
)

//nolint:ireturn
func NewBrazeCatalogItemResource() resource.Resource {
	return &brazeCatalogItemResource{}
}

type brazeCatalogItemResource struct {
	providerData brazeProviderData
}

func (r *brazeCatalogItemResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_catalog_item"
}

func (r *brazeCatalogItemResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = BrazeCatalogItemResourceIdentitySchema()
}

func (r *brazeCatalogItemResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = BrazeCatalogItemResourceSchema(ctx)
}

func (r *brazeCatalogItemResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	SetProviderDataFromResourceConfigureRequest(req, &r.providerData)
}

func (r *brazeCatalogItemResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config brazeCatalogItemModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() || config.ValuesJSON.IsUnknown() || config.ValuesJSON.IsNull() {
		return
	}

	_, err := config.ToCatalogItemWrite()
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("values_json"), "Invalid catalog item values", detailFromError(err))
	}
}

func (r *brazeCatalogItemResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != "" || req.Identity == nil {
		resp.Diagnostics.AddError(
			"Invalid import identity",
			"Import catalog items with an identity import block containing `catalog_name` and `item_id`.",
		)

		return
	}

	var (
		catalogName types.String
		itemID      types.String
	)

	resp.Diagnostics.Append(req.Identity.GetAttribute(ctx, path.Root("catalog_name"), &catalogName)...)
	resp.Diagnostics.Append(req.Identity.GetAttribute(ctx, path.Root("item_id"), &itemID)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if catalogName.ValueString() == "" || itemID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Invalid import identity",
			"Import catalog items with an identity import block containing non-empty `catalog_name` and `item_id`.",
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), catalogName.ValueString()+"/"+itemID.ValueString())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("catalog_name"), catalogName.ValueString())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("item_id"), itemID.ValueString())...)
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("catalog_name"), catalogName.ValueString())...)
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("item_id"), itemID.ValueString())...)
}

func (r *brazeCatalogItemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan brazeCatalogItemModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(plan.CatalogName.ValueString() + "/" + plan.ItemID.ValueString())

	data, err := r.providerData.catalogItems.Create(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Catalog Item", detailFromError(err))

		return
	}

	resp.Diagnostics.Append(setCatalogItemIdentityAndState(ctx, resp.Identity, &resp.State, data.CatalogName.ValueString(), data.ItemID.ValueString(), &data)...)
}

func (r *brazeCatalogItemResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state brazeCatalogItemModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	data, err := r.providerData.catalogItems.Read(ctx, state.CatalogName.ValueString(), state.ItemID.ValueString())
	if err != nil {
		if isBrazeObjectNotFound(err) {
			resp.Diagnostics.AddWarning("Catalog Item not found", detailFromError(err))
			resp.State.RemoveResource(ctx)

			return
		}

		resp.Diagnostics.AddError("Failed to read Catalog Item", detailFromError(err))

		return
	}

	resp.Diagnostics.Append(setCatalogItemIdentityAndState(ctx, resp.Identity, &resp.State, data.CatalogName.ValueString(), data.ItemID.ValueString(), &data)...)
}

func (r *brazeCatalogItemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan brazeCatalogItemModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(plan.CatalogName.ValueString() + "/" + plan.ItemID.ValueString())

	data, err := r.providerData.catalogItems.Update(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Catalog Item", detailFromError(err))

		return
	}

	resp.Diagnostics.Append(setCatalogItemIdentityAndState(ctx, resp.Identity, &resp.State, data.CatalogName.ValueString(), data.ItemID.ValueString(), &data)...)
}

func (r *brazeCatalogItemResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state brazeCatalogItemModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.providerData.catalogItems.Delete(ctx, state.CatalogName.ValueString(), state.ItemID.ValueString())
	if err != nil && !isBrazeObjectNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete Catalog Item", detailFromError(err))
	}
}
