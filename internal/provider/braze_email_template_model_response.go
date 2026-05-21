package provider

import (
	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

//revive:disable:unexported-return
func NewBrazeEmailTemplateModelFromGetEmailTemplateInfoResponse(response brazeclient.GetEmailTemplateInfoResponse) brazeEmailTemplateModel {
	model := brazeEmailTemplateModel{
		IDIdentityModel: IDIdentityModel{
			ID: types.StringValue(response.EmailTemplateID),
		},

		TemplateName:  types.StringValue(response.TemplateName),
		Description:   types.StringPointerValue(response.Description.GetPointer()),
		Subject:       types.StringValue(response.Subject),
		Preheader:     types.StringPointerValue(response.Preheader.GetPointer()),
		PlaintextBody: types.StringPointerValue(response.PlaintextBody.GetPointer()),
	}

	if body := response.Body.GetPointer(); body != nil {
		model.Body = types.StringValue(*body)
	} else {
		model.Body = types.StringValue("")
	}

	if shouldInlineCSS, ok := response.ShouldInlineCSS.Get(); ok {
		model.ShouldInlineCSS = types.BoolValue(shouldInlineCSS)
	}

	if tags, ok := response.Tags.Get(); ok {
		model.Tags = NewTypedListFromStringSlice(tags)
	} else {
		model.Tags = NewTypedListNull[types.String]()
	}

	return model
}
