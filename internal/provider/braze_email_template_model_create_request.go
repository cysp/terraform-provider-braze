package provider

import (
	"context"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

func (m brazeEmailTemplateModel) ToCreateEmailTemplateRequest(ctx context.Context) (brazeclient.CreateEmailTemplateRequest, error) {
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
