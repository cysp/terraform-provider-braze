package provider

import (
	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

//revive:disable:unexported-return
func NewBrazeEmailTemplateModelFromGetEmailTemplateInfoResponse(response brazeclient.GetEmailTemplateInfoResponse) brazeEmailTemplateModel {
	model := brazeEmailTemplateModel{
		IDIdentityModel: IDIdentityModel{
			ID: types.StringValue(response.GetEmailTemplateID()),
		},

		TemplateName:  types.StringValue(response.GetTemplateName()),
		Subject:       types.StringPointerValue(response.GetSubject().GetPointer()),
		Body:          types.StringPointerValue(response.GetBody().GetPointer()),
		PlaintextBody: types.StringPointerValue(response.GetPlaintextBody().GetPointer()),
		Preheader:     types.StringPointerValue(response.GetPreheader().GetPointer()),
	}

	shouldInlineCSS, shouldInlineCSSOk := response.ShouldInlineCSS.Get()
	if shouldInlineCSSOk {
		model.ShouldInlineCSS = types.BoolValue(shouldInlineCSS)
	} else {
		model.ShouldInlineCSS = types.BoolNull()
	}

	tags, tagsOk := response.Tags.Get()
	if tagsOk {
		model.Tags = NewTypedListFromStringSlice(tags)
	} else {
		model.Tags = NewTypedListNull[types.String]()
	}

	return model
}
