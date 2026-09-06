package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type nonBlankStringValidator struct{}

func (nonBlankStringValidator) Description(context.Context) string {
	return "Must contain a non-whitespace character."
}

func (v nonBlankStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (nonBlankStringValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if strings.TrimSpace(req.ConfigValue.ValueString()) == "" {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Attribute Value", "Braze requires a non-blank value for this attribute.")
	}
}
