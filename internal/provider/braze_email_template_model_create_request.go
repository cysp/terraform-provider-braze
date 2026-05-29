package provider

import (
	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

func (m brazeEmailTemplateModel) ToCreateEmailTemplateRequest() brazeclient.CreateEmailTemplateRequest {
	req := brazeclient.CreateEmailTemplateRequest{
		TemplateName:  m.TemplateName.ValueString(),
		Subject:       brazeclient.NewNilString(m.Subject.ValueString()),
		Body:          brazeclient.NewNilString(m.Body.ValueString()),
		PlaintextBody: brazeclient.NewOptNilPointerString(m.PlaintextBody.ValueStringPointer()),
		Preheader:     brazeclient.NewOptNilPointerString(m.Preheader.ValueStringPointer()),
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
