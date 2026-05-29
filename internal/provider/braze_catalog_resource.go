package provider

import (
	"context"
	"fmt"
	"strings"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.Resource                   = (*brazeCatalogResource)(nil)
	_ resource.ResourceWithConfigure      = (*brazeCatalogResource)(nil)
	_ resource.ResourceWithIdentity       = (*brazeCatalogResource)(nil)
	_ resource.ResourceWithImportState    = (*brazeCatalogResource)(nil)
	_ resource.ResourceWithValidateConfig = (*brazeCatalogResource)(nil)
)

//nolint:ireturn
func NewBrazeCatalogResource() resource.Resource {
	return &brazeCatalogResource{}
}

type brazeCatalogResource struct {
	providerData brazeProviderData
}

func (r *brazeCatalogResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_catalog"
}

func (r *brazeCatalogResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = BrazeCatalogResourceIdentitySchema()
}

func (r *brazeCatalogResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = BrazeCatalogResourceSchema(ctx)
}

func (r *brazeCatalogResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	SetProviderDataFromResourceConfigureRequest(req, &r.providerData)
}

func (r *brazeCatalogResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config brazeCatalogModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() || config.Fields.IsUnknown() || config.Fields.IsNull() {
		return
	}

	fields, err := catalogFieldsFromTerraform(ctx, config.Fields)
	if err != nil {
		resp.Diagnostics.AddError("Invalid catalog fields", detailFromError(err))

		return
	}

	if len(fields) == 0 || fields[0].GetName() != "id" || fields[0].GetType() != brazeclient.CatalogFieldTypeString {
		resp.Diagnostics.AddAttributeError(
			path.Root("fields"),
			"Invalid catalog fields",
			"Braze requires the first catalog field to be named \"id\" with type \"string\".",
		)
	}

	for i, field := range fields {
		if field.GetType().Validate() != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("fields").AtListIndex(i).AtName("type"),
				"Invalid catalog field type",
				fmt.Sprintf("Braze catalog field type must be one of %s. Got %q.", strings.Join(catalogFieldTypeValues(), ", "), field.GetType()),
			)
		}
	}
}

func (r *brazeCatalogResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("name"), req.ID)...)
}

func (r *brazeCatalogResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan brazeCatalogModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	data, err := r.providerData.catalogs.Create(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Catalog", detailFromError(err))

		return
	}

	resp.Diagnostics.Append(setNamedIdentityAndState(ctx, resp.Identity, &resp.State, data.Name.ValueString(), &data)...)
}

func (r *brazeCatalogResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state brazeCatalogModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	data, err := r.providerData.catalogs.Read(ctx, state.Name.ValueString())
	if err != nil {
		if isBrazeObjectNotFound(err) {
			resp.Diagnostics.AddWarning("Catalog not found", detailFromError(err))
			resp.State.RemoveResource(ctx)

			return
		}

		resp.Diagnostics.AddError("Failed to read Catalog", detailFromError(err))

		return
	}

	resp.Diagnostics.Append(setNamedIdentityAndState(ctx, resp.Identity, &resp.State, data.Name.ValueString(), &data)...)
}

func (r *brazeCatalogResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Catalog update is not supported", "Braze does not provide a synchronous catalog update endpoint; schema changes require replacement.")
}

func (r *brazeCatalogResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state brazeCatalogModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.providerData.catalogs.Delete(ctx, state.Name.ValueString())
	if err != nil && !isBrazeObjectNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete Catalog", detailFromError(err))
	}
}
