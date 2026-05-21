package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type brazeEmailTemplateModel struct {
	IDIdentityModel

	TemplateName    types.String            `tfsdk:"template_name"`
	Description     types.String            `tfsdk:"description"`
	Subject         types.String            `tfsdk:"subject"`
	Preheader       types.String            `tfsdk:"preheader"`
	Body            types.String            `tfsdk:"body"`
	PlaintextBody   types.String            `tfsdk:"plaintext_body"`
	ShouldInlineCSS types.Bool              `tfsdk:"should_inline_css"`
	Tags            TypedList[types.String] `tfsdk:"tags"`
}
