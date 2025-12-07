package provider

import (
	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

func (m brazeEmailTemplateModel) ToUpdateEmailTemplateRequest() brazeclient.UpdateEmailTemplateRequest {
	req := brazeclient.UpdateEmailTemplateRequest{
		EmailTemplateID: m.ID.ValueString(),
		TemplateName:    brazeclient.NewOptString(m.TemplateName.ValueString()),
		Description:     brazeclient.NewOptNilPointerString(m.Description.ValueStringPointer()),
		Subject:         brazeclient.NewOptString(m.Subject.ValueString()),
		Preheader:       brazeclient.NewOptNilPointerString(m.Preheader.ValueStringPointer()),
		Body:            brazeclient.NewOptString(m.Body.ValueString()),
		PlaintextBody:   brazeclient.NewOptNilPointerString(m.PlaintextBody.ValueStringPointer()),
	}

	if !m.ShouldInlineCSS.IsNull() {
		req.ShouldInlineCSS.SetTo(m.ShouldInlineCSS.ValueBool())
	}

	tags := TypedListToStringSlice(m.Tags)
	if tags != nil {
		req.Tags.SetTo(tags)
	} else {
		req.Tags.SetToNull()
	}

	return req
}
