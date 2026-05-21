package provider

import (
	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

func (m brazeEmailTemplateModel) ToCreateEmailTemplateRequest() brazeclient.CreateEmailTemplateRequest {
	req := brazeclient.CreateEmailTemplateRequest{
		TemplateName: m.TemplateName.ValueString(),
		Subject:      m.Subject.ValueString(),
		Body:         m.Body.ValueString(),
	}

	if !m.Description.IsNull() {
		req.Description = brazeclient.NewOptNilString(m.Description.ValueString())
	}

	if !m.Preheader.IsNull() {
		req.Preheader = brazeclient.NewOptNilString(m.Preheader.ValueString())
	}

	if !m.PlaintextBody.IsNull() {
		req.PlaintextBody = brazeclient.NewOptNilString(m.PlaintextBody.ValueString())
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
