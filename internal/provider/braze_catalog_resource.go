package provider

import (
	"context"
	"fmt"
	"strings"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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

func (r *brazeCatalogResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(setProviderData(req.ProviderData, &r.providerData)...)
}

func (r *brazeCatalogResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config brazeCatalogModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() || config.Fields.IsUnknown() || config.Fields.IsNull() {
		return
	}

	fields := config.Fields.Elements()
	if len(fields) == 0 {
		resp.Diagnostics.AddAttributeError(path.Root("fields"), "Invalid catalog fields", "Braze requires the first catalog field to be named \"id\" with type \"string\".")

		return
	}

	seen := make(map[string]bool, len(fields))
	for i, value := range fields {
		fieldPath := path.Root("fields").AtListIndex(i)

		if value.IsUnknown() {
			continue
		}

		if value.IsNull() {
			resp.Diagnostics.AddAttributeError(fieldPath, "Invalid catalog field", "Catalog fields must be non-null objects.")

			continue
		}

		var field brazeCatalogFieldModel

		object, ok := value.(types.Object)
		if !ok {
			resp.Diagnostics.AddAttributeError(fieldPath, "Invalid catalog field", "Catalog fields must be objects.")

			return
		}

		resp.Diagnostics.Append(object.As(ctx, &field, basetypes.ObjectAsOptions{})...)

		if resp.Diagnostics.HasError() {
			return
		}

		if i == 0 && ((!field.Name.IsUnknown() && field.Name.ValueString() != "id") || (!field.Type.IsUnknown() && field.Type.ValueString() != "string")) {
			resp.Diagnostics.AddAttributeError(fieldPath, "Invalid catalog fields", "Braze requires the first catalog field to be named \"id\" with type \"string\".")
		}

		if !field.Name.IsUnknown() {
			name := field.Name.ValueString()
			if name == "" || seen[name] {
				resp.Diagnostics.AddAttributeError(fieldPath.AtName("name"), "Invalid catalog field name", "Catalog field names must be non-empty and unique.")
			}

			seen[name] = true
		}

		if !field.Type.IsUnknown() && brazeclient.CatalogFieldType(field.Type.ValueString()).Validate() != nil {
			resp.Diagnostics.AddAttributeError(fieldPath.AtName("type"), "Invalid catalog field type",
				fmt.Sprintf("Braze catalog field type must be one of %s. Got %q.", strings.Join(catalogFieldTypeValues(), ", "), field.Type.ValueString()))
		}
	}
}

func (r *brazeCatalogResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" && req.Identity == nil {
		resp.Diagnostics.AddError("Invalid catalog import", "Supply the catalog name as the import ID or use an identity import block.")

		return
	}

	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("name"), path.Root("name"), req, resp)
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
