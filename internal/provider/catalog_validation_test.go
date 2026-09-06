//nolint:testpackage
package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func catalogFieldValue(name, fieldType types.String) types.Object {
	return types.ObjectValueMust(BrazeCatalogFieldObjectType().AttrTypes, map[string]attr.Value{"name": name, "type": fieldType})
}

func TestCatalogValidationUnknownAndDuplicateFields(t *testing.T) {
	t.Parallel()

	idField := catalogFieldValue(types.StringValue("id"), types.StringValue("string"))
	for name, test := range map[string]struct {
		fields  []attr.Value
		invalid bool
	}{
		"unknown first name":   {[]attr.Value{catalogFieldValue(types.StringUnknown(), types.StringValue("string"))}, false},
		"unknown field type":   {[]attr.Value{idField, catalogFieldValue(types.StringValue("active"), types.StringUnknown())}, false},
		"unknown field object": {[]attr.Value{idField, types.ObjectUnknown(BrazeCatalogFieldObjectType().AttrTypes)}, false},
		"duplicate names":      {[]attr.Value{idField, idField}, true},
		"invalid id type":      {[]attr.Value{catalogFieldValue(types.StringValue("id"), types.StringValue("number"))}, true},
		"invalid type":         {[]attr.Value{idField, catalogFieldValue(types.StringValue("active"), types.StringValue("bool"))}, true},
		"null object":          {[]attr.Value{idField, types.ObjectNull(BrazeCatalogFieldObjectType().AttrTypes)}, true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			plan := tfsdk.Plan{Schema: BrazeCatalogResourceSchema(t.Context())}
			diags := plan.Set(t.Context(), brazeCatalogModel{
				Name: types.StringValue("products"), Description: types.StringValue("Products"),
				Fields: types.ListValueMust(BrazeCatalogFieldObjectType(), test.fields),
			})
			require.False(t, diags.HasError(), "%v", diags)

			var response resource.ValidateConfigResponse

			r, ok := NewBrazeCatalogResource().(resource.ResourceWithValidateConfig)
			require.True(t, ok)
			r.ValidateConfig(t.Context(), resource.ValidateConfigRequest{Config: tfsdk.Config(plan)}, &response)
			assert.Equal(t, test.invalid, response.Diagnostics.HasError(), "%v", response.Diagnostics)
		})
	}
}
