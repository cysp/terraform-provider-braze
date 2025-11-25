package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewTypedListFromStringSlice(slice []string) TypedList[types.String] {
	if slice == nil {
		return NewTypedList([]types.String{})
	}

	listElementValues := make([]types.String, len(slice))
	for index, item := range slice {
		listElementValues[index] = types.StringValue(item)
	}

	return NewTypedList(listElementValues)
}

func TypedListToStringSlice(l TypedList[types.String]) []string {
	elements := l.Elements()
	knownCount := 0

	for _, item := range elements {
		if !item.IsNull() && !item.IsUnknown() {
			knownCount++
		}
	}

	slice := make([]string, 0, knownCount)

	for _, item := range elements {
		if item.IsNull() || item.IsUnknown() {
			continue
		}

		slice = append(slice, item.ValueString())
	}

	return slice
}
