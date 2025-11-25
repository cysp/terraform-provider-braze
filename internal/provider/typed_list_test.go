package provider_test

import (
	"testing"

	. "github.com/cysp/terraform-provider-braze/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypedListStates(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value         TypedList[types.String]
		expectNull    bool
		expectUnknown bool
	}{
		"unknown": {
			value:         NewTypedListUnknown[types.String](),
			expectNull:    false,
			expectUnknown: true,
		},
		"null": {
			value:         NewTypedListNull[types.String](),
			expectNull:    true,
			expectUnknown: false,
		},
		"known empty": {
			value:         NewTypedList([]types.String{}),
			expectNull:    false,
			expectUnknown: false,
		},
		"known with elements": {
			value:         NewTypedList([]types.String{types.StringValue("test")}),
			expectNull:    false,
			expectUnknown: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expectNull, tc.value.IsNull())
			assert.Equal(t, tc.expectUnknown, tc.value.IsUnknown())
		})
	}
}

func TestTypedListEqual(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		left     TypedList[types.String]
		right    attr.Value
		expected bool
	}{
		"null equals null": {
			left:     NewTypedList[types.String](nil),
			right:    NewTypedList[types.String](nil),
			expected: true,
		},
		"null TypedListNull equals null TypedListNull": {
			left:     NewTypedListNull[types.String](),
			right:    NewTypedListNull[types.String](),
			expected: true,
		},
		"unknown equals unknown": {
			left:     NewTypedListUnknown[types.String](),
			right:    NewTypedListUnknown[types.String](),
			expected: true,
		},
		"null not equal to unknown": {
			left:     NewTypedListNull[types.String](),
			right:    NewTypedListUnknown[types.String](),
			expected: false,
		},
		"null not equal to known": {
			left:     NewTypedList[types.String](nil),
			right:    NewTypedList([]types.String{types.StringValue("x")}),
			expected: false,
		},
		"known equals known same values": {
			left:     NewTypedList([]types.String{types.StringValue("x")}),
			right:    NewTypedList([]types.String{types.StringValue("x")}),
			expected: true,
		},
		"known not equal known different values": {
			left:     NewTypedList([]types.String{types.StringValue("x")}),
			right:    NewTypedList([]types.String{types.StringValue("y")}),
			expected: false,
		},
		"known not equal known different lengths": {
			left:     NewTypedList([]types.String{types.StringValue("x")}),
			right:    NewTypedList([]types.String{types.StringValue("x"), types.StringValue("y")}),
			expected: false,
		},
		"empty lists equal": {
			left:     NewTypedList([]types.String{}),
			right:    NewTypedList([]types.String{}),
			expected: true,
		},
		"not equal to different type": {
			left:     NewTypedList([]types.String{types.StringValue("x")}),
			right:    types.StringValue("x"),
			expected: false,
		},
		"null TypedListNull not equal to known": {
			left:     NewTypedListNull[types.String](),
			right:    NewTypedList([]types.String{types.StringValue("x")}),
			expected: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actual := tc.left.Equal(tc.right)
			assert.Equal(t, tc.expected, actual, "Expected Equal() to return %v", tc.expected)
		})
	}
}

func TestTypedListToTerraformValue(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value            TypedList[types.String]
		expectNull       bool
		expectUnknown    bool
		expectedElements int
	}{
		"null": {
			value:            NewTypedListNull[types.String](),
			expectNull:       true,
			expectUnknown:    false,
			expectedElements: 0,
		},
		"unknown": {
			value:            NewTypedListUnknown[types.String](),
			expectNull:       false,
			expectUnknown:    true,
			expectedElements: 0,
		},
		"empty list": {
			value:            NewTypedList([]types.String{}),
			expectNull:       false,
			expectUnknown:    false,
			expectedElements: 0,
		},
		"list with elements": {
			value:            NewTypedList([]types.String{types.StringValue("one"), types.StringValue("two")}),
			expectNull:       false,
			expectUnknown:    false,
			expectedElements: 2,
		},
		"list with unknown elements": {
			value: NewTypedList([]types.String{
				types.StringValue("known"),
				types.StringUnknown(),
				types.StringNull(),
			}),
			expectNull:       false,
			expectUnknown:    false,
			expectedElements: 3,
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			tfValue, err := testCase.value.ToTerraformValue(ctx)
			require.NoError(t, err)

			assert.Equal(t, testCase.expectNull, tfValue.IsNull())

			if !testCase.expectNull {
				assert.Equal(t, testCase.expectUnknown, !tfValue.IsKnown())
			}

			if testCase.expectNull || testCase.expectUnknown {
				return
			}

			var extracted []tftypes.Value
			require.NoError(t, tfValue.As(&extracted))
			assert.Len(t, extracted, testCase.expectedElements)
		})
	}
}

func TestTypedListToListValue(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value            TypedList[types.String]
		expectNull       bool
		expectUnknown    bool
		expectedElements []types.String
	}{
		"unknown": {
			value:         NewTypedListUnknown[types.String](),
			expectUnknown: true,
		},
		"null": {
			value:      NewTypedListNull[types.String](),
			expectNull: true,
		},
		"empty list": {
			value:            NewTypedList([]types.String{}),
			expectedElements: []types.String{},
		},
		"list with elements": {
			value: NewTypedList([]types.String{types.StringValue("a"), types.StringValue("b")}),
			expectedElements: []types.String{
				types.StringValue("a"),
				types.StringValue("b"),
			},
		},
		"list with mixed states": {
			value: NewTypedList([]types.String{
				types.StringValue("known"),
				types.StringUnknown(),
				types.StringNull(),
			}),
			expectedElements: []types.String{
				types.StringValue("known"),
				types.StringUnknown(),
				types.StringNull(),
			},
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			listValue, diags := testCase.value.ToListValue(ctx)
			assert.Empty(t, diags)

			assert.Equal(t, testCase.expectNull, listValue.IsNull())
			assert.Equal(t, testCase.expectUnknown, listValue.IsUnknown())

			if testCase.expectNull || testCase.expectUnknown {
				return
			}

			var elems []types.String

			elemDiags := listValue.ElementsAs(ctx, &elems, false)
			assert.Empty(t, elemDiags)
			assert.Equal(t, testCase.expectedElements, elems)
		})
	}
}

func TestTypedListTypeMetadata(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	listType := TypedList[types.String]{}.Type(ctx)
	customType := TypedList[types.String]{}.CustomType(ctx)
	assert.True(t, listType.Equal(customType))
	assert.Equal(t, listType.String(), customType.String())
}

func TestTypedListTypeApplyTerraform5AttributePathStep(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	listType := TypedList[types.String]{}.Type(ctx)

	// valid element step
	elementAny, err := listType.ApplyTerraform5AttributePathStep(tftypes.ElementKeyInt(0))
	require.NoError(t, err)

	_, ok := elementAny.(attr.Type)
	assert.True(t, ok)

	// invalid step
	_, err = listType.ApplyTerraform5AttributePathStep(tftypes.AttributeName("foo"))
	assert.Error(t, err)
}

func TestTypedListTypeWithElementType(t *testing.T) {
	t.Parallel()
	// Override element type and ensure Equal considers element types
	underlying := types.StringType
	overridden := TypedListType[types.String]{}.WithElementType(underlying)
	assert.True(t, overridden.ElementType().Equal(underlying))
	// Equal should be true comparing same override
	other := TypedListType[types.String]{}.WithElementType(underlying)
	assert.True(t, overridden.Equal(other))
}

func TestTypedListString(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value    TypedList[types.String]
		expected string
	}{
		"null": {
			value:    NewTypedListNull[types.String](),
			expected: "TypedList[basetypes.StringValue]",
		},
		"unknown": {
			value:    NewTypedListUnknown[types.String](),
			expected: "TypedList[basetypes.StringValue]",
		},
		"known": {
			value:    NewTypedList([]types.String{types.StringValue("test")}),
			expected: "TypedList[basetypes.StringValue]",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestTypedListElements(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value    TypedList[types.String]
		expected []types.String
	}{
		"null": {
			value:    NewTypedListNull[types.String](),
			expected: []types.String{},
		},
		"unknown": {
			value:    NewTypedListUnknown[types.String](),
			expected: []types.String{},
		},
		"empty": {
			value:    NewTypedList([]types.String{}),
			expected: []types.String{},
		},
		"with elements": {
			value: NewTypedList([]types.String{
				types.StringValue("a"),
				types.StringValue("b"),
			}),
			expected: []types.String{
				types.StringValue("a"),
				types.StringValue("b"),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			elements := tc.value.Elements()
			assert.Equal(t, tc.expected, elements)
		})
	}
}

func TestTypedListElementsDefensiveCopy(t *testing.T) {
	t.Parallel()

	original := NewTypedList([]types.String{
		types.StringValue("original"),
	})

	elements := original.Elements()
	elements[0] = types.StringValue("modified")

	originalElements := original.Elements()
	assert.Equal(t, types.StringValue("original"), originalElements[0], "Original list should not be modified")
}

func TestTypedListLen(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value    TypedList[types.String]
		expected int
	}{
		"null": {
			value:    NewTypedListNull[types.String](),
			expected: 0,
		},
		"unknown": {
			value:    NewTypedListUnknown[types.String](),
			expected: 0,
		},
		"empty": {
			value:    NewTypedList([]types.String{}),
			expected: 0,
		},
		"single element": {
			value:    NewTypedList([]types.String{types.StringValue("a")}),
			expected: 1,
		},
		"multiple elements": {
			value: NewTypedList([]types.String{
				types.StringValue("a"),
				types.StringValue("b"),
				types.StringValue("c"),
			}),
			expected: 3,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, tc.value.Len())
		})
	}
}

func TestTypedListTypeAndCustomType(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	t.Run("Type returns correct type", func(t *testing.T) {
		t.Parallel()

		value := NewTypedList([]types.String{types.StringValue("test")})
		attrType := value.Type(ctx)

		assert.NotNil(t, attrType)
		assert.Contains(t, attrType.String(), "TypedList")
	})

	t.Run("CustomType returns correct type", func(t *testing.T) {
		t.Parallel()

		value := NewTypedList([]types.String{types.StringValue("test")})
		customType := value.CustomType(ctx)

		assert.NotNil(t, customType)
		assert.True(t, customType.Equal(value.Type(ctx)))
	})

	t.Run("Type and CustomType are equal", func(t *testing.T) {
		t.Parallel()

		value := NewTypedList([]types.String{})
		assert.True(t, value.Type(ctx).Equal(value.CustomType(ctx)))
	})
}
