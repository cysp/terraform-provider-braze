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

		TemplateName: types.StringValue(response.TemplateName),
		Subject:      types.StringValue(response.Subject),
	}

	if description, ok := response.Description.Get(); ok {
		model.Description = types.StringPointerValue(&description)
	}

	if preheader, ok := response.Preheader.Get(); ok {
		model.Preheader = types.StringPointerValue(&preheader)
	}

	if body, ok := response.Body.Get(); ok {
		model.Body = types.StringPointerValue(&body)
	}

	if plaintextBody, ok := response.PlaintextBody.Get(); ok {
		model.PlaintextBody = types.StringPointerValue(&plaintextBody)
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
