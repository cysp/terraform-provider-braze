package provider_test

import (
	"testing"

	. "github.com/cysp/terraform-provider-braze/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTypedListFromStringSlice(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	tests := map[string]struct {
		input    []string
		expected TypedList[types.String]
	}{
		"nil slice": {
			input:    nil,
			expected: NewTypedList([]types.String{}),
		},
		"empty slice": {
			input:    []string{},
			expected: NewTypedList([]types.String{}),
		},
		"single element": {
			input:    []string{"one"},
			expected: NewTypedList([]types.String{types.StringValue("one")}),
		},
		"multiple elements": {
			input:    []string{"one", "two"},
			expected: NewTypedList([]types.String{types.StringValue("one"), types.StringValue("two")}),
		},
		"empty strings": {
			input:    []string{"", ""},
			expected: NewTypedList([]types.String{types.StringValue(""), types.StringValue("")}),
		},
		"with special characters": {
			input:    []string{"hello world", "test@example.com", "path/to/file"},
			expected: NewTypedList([]types.String{types.StringValue("hello world"), types.StringValue("test@example.com"), types.StringValue("path/to/file")}),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actual := NewTypedListFromStringSlice(tc.input)
			assert.True(t, tc.expected.Equal(actual), "Expected %v but got %v", tc.expected, actual)

			// Round trip ToTerraformValue -> ValueFromTerraform
			tfVal, err := actual.ToTerraformValue(ctx)
			require.NoError(t, err)
			attrVal, err := actual.Type(ctx).ValueFromTerraform(ctx, tfVal)
			require.NoError(t, err)

			roundTrip, ok := attrVal.(TypedList[types.String])
			require.True(t, ok)
			assert.True(t, actual.Equal(roundTrip), "Round trip failed")
		})
	}
}

func TestTypedListToStringSlice(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input    TypedList[types.String]
		expected []string
	}{
		"empty list": {
			input:    NewTypedList([]types.String{}),
			expected: []string{},
		},
		"single known value": {
			input:    NewTypedList([]types.String{types.StringValue("test")}),
			expected: []string{"test"},
		},
		"multiple known values": {
			input:    NewTypedList([]types.String{types.StringValue("a"), types.StringValue("b"), types.StringValue("c")}),
			expected: []string{"a", "b", "c"},
		},
		"filters out unknown and null": {
			input: NewTypedList([]types.String{
				types.StringValue("value1"),
				types.StringUnknown(),
				types.StringNull(),
				types.StringValue("value2"),
			}),
			expected: []string{"value1", "value2"},
		},
		"all unknown": {
			input:    NewTypedList([]types.String{types.StringUnknown(), types.StringUnknown()}),
			expected: []string{},
		},
		"all null": {
			input:    NewTypedList([]types.String{types.StringNull(), types.StringNull()}),
			expected: []string{},
		},
		"empty string values": {
			input:    NewTypedList([]types.String{types.StringValue(""), types.StringValue("")}),
			expected: []string{"", ""},
		},
		"mixed with empty strings": {
			input: NewTypedList([]types.String{
				types.StringValue("before"),
				types.StringValue(""),
				types.StringNull(),
				types.StringValue("after"),
			}),
			expected: []string{"before", "", "after"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			slice := TypedListToStringSlice(tc.input)
			assert.Equal(t, tc.expected, slice)
		})
	}
}

func TestTypedListStringConversionRoundTrip(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input []string
	}{
		"empty":              {input: []string{}},
		"single":             {input: []string{"test"}},
		"multiple":           {input: []string{"a", "b", "c"}},
		"with empty strings": {input: []string{"", "value", ""}},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			typedList := NewTypedListFromStringSlice(testCase.input)
			result := TypedListToStringSlice(typedList)

			assert.Equal(t, testCase.input, result, "Round trip conversion failed")
		})
	}
}
