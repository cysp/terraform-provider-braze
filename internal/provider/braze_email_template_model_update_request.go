package provider

import (
	"context"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

func (m brazeEmailTemplateModel) ToUpdateEmailTemplateRequest(ctx context.Context) (brazeclient.UpdateEmailTemplateRequest, error) {
	req := brazeclient.UpdateEmailTemplateRequest{
		EmailTemplateID: m.ID.ValueString(),
		TemplateName:    brazeclient.NewOptString(m.TemplateName.ValueString()),
		Subject:         brazeclient.NewOptNilPointerString(m.Subject.ValueStringPointer()),
		Body:            brazeclient.NewOptNilPointerString(m.Body.ValueStringPointer()),
		PlaintextBody:   brazeclient.NewOptNilPointerString(m.PlaintextBody.ValueStringPointer()),
		Preheader:       brazeclient.NewOptNilPointerString(m.Preheader.ValueStringPointer()),
	}

	if !m.ShouldInlineCSS.IsNull() {
		req.ShouldInlineCSS.SetTo(m.ShouldInlineCSS.ValueBool())
	} else {
		req.ShouldInlineCSS.SetToNull()
	}

	tags, err := stringListToSlice(ctx, m.Tags)
	if err != nil {
		return req, err
	}

	if tags != nil {
		req.Tags.SetTo(tags)
	} else {
		req.Tags.SetToNull()
	}

	return req, nil
}
