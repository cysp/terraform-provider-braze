package provider

import (
	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

func (m brazeEmailTemplateModel) ToUpdateEmailTemplateRequest() brazeclient.UpdateEmailTemplateRequest {
	req := brazeclient.UpdateEmailTemplateRequest{
		EmailTemplateID: m.ID.ValueString(),
	}

	if !m.TemplateName.IsNull() {
		req.TemplateName.SetTo(m.TemplateName.ValueString())
	}

	if !m.Description.IsNull() {
		req.Description = brazeclient.NewOptNilString(m.Description.ValueString())
	}

	if !m.Subject.IsNull() {
		req.Subject.SetTo(m.Subject.ValueString())
	}

	if !m.Preheader.IsNull() {
		req.Preheader = brazeclient.NewOptNilString(m.Preheader.ValueString())
	}

	if !m.Body.IsNull() {
		req.Body.SetTo(m.Body.ValueString())
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
