package provider_test

import (
	"testing"

	. "github.com/cysp/terraform-provider-braze/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypedListTypeValueFromTerraform(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	listType := TypedList[types.String]{}.Type(ctx)

	testcases := map[string]struct {
		tfval       tftypes.Value
		expectError bool
		expected    TypedList[types.String]
	}{
		"null type": {
			tfval:    tftypes.NewValue(nil, nil),
			expected: NewTypedListNull[types.String](),
		},
		"unknown": {
			tfval:    tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, tftypes.UnknownValue),
			expected: NewTypedListUnknown[types.String](),
		},
		"null": {
			tfval:    tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
			expected: NewTypedListNull[types.String](),
		},
		"incorrect type - string instead of list": {
			tfval:       tftypes.NewValue(tftypes.String, "string"),
			expectError: true,
		},
		"incorrect element type - number list": {
			tfval:       tftypes.NewValue(tftypes.List{ElementType: tftypes.Number}, []tftypes.Value{}),
			expectError: true,
		},
		"empty list": {
			tfval:    tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{}),
			expected: NewTypedList([]types.String{}),
		},
		"with elements": {
			tfval: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "value1"),
				tftypes.NewValue(tftypes.String, "value2"),
			}),
			expected: NewTypedList([]types.String{
				types.StringValue("value1"),
				types.StringValue("value2"),
			}),
		},
		"with interior unknown element": {
			tfval: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "value1"),
				tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
				tftypes.NewValue(tftypes.String, "value2"),
			}),
			expected: NewTypedList([]types.String{
				types.StringValue("value1"),
				types.StringUnknown(),
				types.StringValue("value2"),
			}),
		},
		"with interior null element": {
			tfval: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "value1"),
				tftypes.NewValue(tftypes.String, nil),
				tftypes.NewValue(tftypes.String, "value2"),
			}),
			expected: NewTypedList([]types.String{
				types.StringValue("value1"),
				types.StringNull(),
				types.StringValue("value2"),
			}),
		},
		"single element": {
			tfval: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "only"),
			}),
			expected: NewTypedList([]types.String{
				types.StringValue("only"),
			}),
		},
	}

	for name, testcase := range testcases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			actual, err := listType.ValueFromTerraform(ctx, testcase.tfval)

			if testcase.expectError {
				require.Error(t, err)
				assert.Nil(t, actual)

				return
			}

			require.NoError(t, err)

			actualTypedList, ok := actual.(TypedList[types.String])
			require.True(t, ok, "Expected TypedList[types.String] but got %T", actual)

			assert.Equal(t, testcase.expected, actualTypedList)
		})
	}
}

func TestTypedListTypeValueFromList(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	listType := TypedListType[types.String]{}

	tests := map[string]struct {
		listValue basetypes.ListValue
		expected  TypedList[types.String]
	}{
		"unknown": {
			listValue: types.ListUnknown(types.StringType),
			expected:  NewTypedListUnknown[types.String](),
		},
		"null": {
			listValue: types.ListNull(types.StringType),
			expected:  NewTypedListNull[types.String](),
		},
		"empty": {
			listValue: types.ListValueMust(types.StringType, []attr.Value{}),
			expected:  NewTypedList([]types.String{}),
		},
		"with elements": {
			listValue: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("a"),
				types.StringValue("b"),
			}),
			expected: NewTypedList([]types.String{
				types.StringValue("a"),
				types.StringValue("b"),
			}),
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, diags := listType.ValueFromList(ctx, testCase.listValue)
			assert.Empty(t, diags)

			typedList, ok := result.(TypedList[types.String])
			require.True(t, ok)

			assert.True(t, testCase.expected.Equal(typedList))
		})
	}
}

func TestTypedListTypeTerraformType(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	listType := TypedListType[types.String]{}

	tfType := listType.TerraformType(ctx)

	assert.NotNil(t, tfType)
	listTfType, ok := tfType.(tftypes.List)
	assert.True(t, ok, "Expected tftypes.List but got %T", tfType)
	assert.Equal(t, tftypes.String, listTfType.ElementType)
}

func TestTypedListTypeValueType(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	listType := TypedListType[types.String]{}

	valueType := listType.ValueType(ctx)

	assert.NotNil(t, valueType)
	_, ok := valueType.(TypedList[types.String])
	assert.True(t, ok, "Expected TypedList[types.String] but got %T", valueType)
}

func TestTypedListTypeString(t *testing.T) {
	t.Parallel()

	listType := TypedListType[types.String]{}

	str := listType.String()
	assert.Contains(t, str, "TypedList")
	assert.Contains(t, str, "types.String")
}

func TestTypedListTypeElementType(t *testing.T) {
	t.Parallel()

	t.Run("default element type", func(t *testing.T) {
		t.Parallel()

		listType := TypedListType[types.String]{}
		elemType := listType.ElementType()

		assert.NotNil(t, elemType)
		assert.True(t, elemType.Equal(types.StringType))
	})

	t.Run("custom element type", func(t *testing.T) {
		t.Parallel()

		customElemType := types.StringType
		listType := TypedListType[types.String]{}.WithElementType(customElemType)
		elemType := listType.ElementType()

		assert.NotNil(t, elemType)
		assert.True(t, elemType.Equal(customElemType))
	})
}

func TestTypedListTypeEqual(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		left     TypedListType[types.String]
		right    attr.Type
		expected bool
	}{
		"equal default types": {
			left:     TypedListType[types.String]{},
			right:    TypedListType[types.String]{},
			expected: true,
		},
		"not equal to different type": {
			left:     TypedListType[types.String]{},
			right:    types.StringType,
			expected: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, tc.left.Equal(tc.right))
		})
	}
}

func TestTypedListTypeEqualOneNilElementType(t *testing.T) {
	t.Parallel()

	typedListNil := NewTypedListNull[attrValueTypeNil]().Type(t.Context())
	typedListString := NewTypedListNull[types.String]().Type(t.Context())

	t.Run("left", func(t *testing.T) {
		t.Parallel()

		assert.False(t, typedListNil.Equal(typedListString))
	})

	t.Run("right", func(t *testing.T) {
		t.Parallel()

		assert.False(t, typedListString.Equal(typedListNil))
	})
}
