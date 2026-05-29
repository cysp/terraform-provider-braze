package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type brazeEmailTemplateModel struct {
	IDIdentityModel

	TemplateName    types.String            `tfsdk:"template_name"`
	Subject         types.String            `tfsdk:"subject"`
	Body            types.String            `tfsdk:"body"`
	PlaintextBody   types.String            `tfsdk:"plaintext_body"`
	Preheader       types.String            `tfsdk:"preheader"`
	Tags            TypedList[types.String] `tfsdk:"tags"`
	ShouldInlineCSS types.Bool              `tfsdk:"should_inline_css"`
}
