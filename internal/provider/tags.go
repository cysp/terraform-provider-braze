package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stringListValue(values []string) types.List {
	if values == nil {
		return types.ListNull(types.StringType)
	}

	elements := make([]attr.Value, len(values))
	for i, value := range values {
		elements[i] = types.StringValue(value)
	}

	return types.ListValueMust(types.StringType, elements)
}

func stringListToSlice(ctx context.Context, value types.List) ([]string, error) {
	var values []string
	if diags := value.ElementsAs(ctx, &values, false); diags.HasError() {
		return nil, diagError(diags)
	}

	return values, nil
}

type tagsValidator struct{}

func (tagsValidator) Description(context.Context) string {
	return "Tags must be non-null strings."
}

func (v tagsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (tagsValidator) ValidateList(_ context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	for i, value := range req.ConfigValue.Elements() {
		if value.IsNull() {
			resp.Diagnostics.AddAttributeError(req.Path.AtListIndex(i), "Invalid tags", "Each tag must be a non-null string. Use an empty list to remove all tags.")
		}
	}
}
