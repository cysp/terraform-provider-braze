package provider

import (
	"context"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type brazeCatalogFieldModel struct {
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

type brazeCatalogModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Fields      types.List   `tfsdk:"fields"`
	NumItems    types.Int64  `tfsdk:"num_items"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func catalogFieldTypeValues() []string {
	return []string{
		string(brazeclient.CatalogFieldTypeString),
		string(brazeclient.CatalogFieldTypeNumber),
		string(brazeclient.CatalogFieldTypeBoolean),
		string(brazeclient.CatalogFieldTypeTime),
		string(brazeclient.CatalogFieldTypeArray),
		string(brazeclient.CatalogFieldTypeObject),
		string(brazeclient.CatalogFieldTypeGeo),
	}
}

func catalogFieldsFromTerraform(ctx context.Context, fields types.List) ([]brazeclient.CatalogField, error) {
	var data []brazeCatalogFieldModel

	diags := fields.ElementsAs(ctx, &data, false)
	if diags.HasError() {
		return nil, diagError(diags)
	}

	out := make([]brazeclient.CatalogField, len(data))
	for i, field := range data {
		out[i] = brazeclient.CatalogField{
			Name: field.Name.ValueString(),
			Type: brazeclient.CatalogFieldType(field.Type.ValueString()),
		}
	}

	return out, nil
}

func catalogFieldsToTerraform(ctx context.Context, fields []brazeclient.CatalogField) (types.List, error) {
	data := make([]brazeCatalogFieldModel, len(fields))
	for i, field := range fields {
		data[i] = brazeCatalogFieldModel{
			Name: types.StringValue(field.GetName()),
			Type: types.StringValue(string(field.GetType())),
		}
	}

	value, diags := types.ListValueFrom(ctx, BrazeCatalogFieldObjectType(), data)
	if diags.HasError() {
		return types.ListNull(BrazeCatalogFieldObjectType()), diagError(diags)
	}

	return value, nil
}

func (m brazeCatalogModel) ToCreateCatalogRequest(ctx context.Context) (brazeclient.CreateCatalogRequest, error) {
	fields, err := catalogFieldsFromTerraform(ctx, m.Fields)
	if err != nil {
		return brazeclient.CreateCatalogRequest{}, err
	}

	return brazeclient.CreateCatalogRequest{
		Catalogs: []brazeclient.Catalog{{
			Name:        m.Name.ValueString(),
			Description: m.Description.ValueString(),
			Fields:      fields,
		}},
	}, nil
}

func newBrazeCatalogModelFromCatalog(ctx context.Context, catalog brazeclient.Catalog) (brazeCatalogModel, error) {
	fields, err := catalogFieldsToTerraform(ctx, catalog.GetFields())
	if err != nil {
		return brazeCatalogModel{}, err
	}

	model := brazeCatalogModel{
		Name:        types.StringValue(catalog.GetName()),
		Description: types.StringValue(catalog.GetDescription()),
		Fields:      fields,
		NumItems:    types.Int64Null(),
		UpdatedAt:   types.StringNull(),
	}

	if numItems, ok := catalog.GetNumItems().Get(); ok {
		model.NumItems = types.Int64Value(int64(numItems))
	}

	if updatedAt, ok := catalog.GetUpdatedAt().Get(); ok {
		model.UpdatedAt = types.StringValue(updatedAt.Format("2006-01-02T15:04:05.999Z07:00"))
	}

	return model, nil
}
