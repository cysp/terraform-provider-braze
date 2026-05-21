package testing

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/google/uuid"
)

func (h *Handler) ListEmailTemplates(ctx context.Context, params brazeclient.ListEmailTemplatesParams) (*brazeclient.ListEmailTemplatesResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	templates := make([]brazeclient.ListEmailTemplatesResponseTemplate, 0)
	for _, template := range h.emailTemplates {
		templates = append(templates, brazeclient.ListEmailTemplatesResponseTemplate{
			EmailTemplateID: template.EmailTemplateID,
			TemplateName:    template.TemplateName,
			Tags:            template.Tags,
		})
	}

	return &brazeclient.ListEmailTemplatesResponse{
		Count:     len(templates),
		Templates: templates,
	}, nil
}

func (h *Handler) GetEmailTemplateInfo(ctx context.Context, params brazeclient.GetEmailTemplateInfoParams) (*brazeclient.GetEmailTemplateInfoResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	template, exists := h.emailTemplates[params.EmailTemplateID]
	if !exists {
		return nil, fmt.Errorf("email template not found: %s", params.EmailTemplateID)
	}

	return template, nil
}

func (h *Handler) CreateEmailTemplate(ctx context.Context, req *brazeclient.CreateEmailTemplateRequest) (*brazeclient.CreateEmailTemplateResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if req.TemplateName == "" {
		return nil, newStatusCodeError(http.StatusUnprocessableEntity)
	}

	templateID := uuid.NewString()

	template := &brazeclient.GetEmailTemplateInfoResponse{
		EmailTemplateID: templateID,
		TemplateName:    req.TemplateName,
		Subject:         req.Subject,
		Body:            brazeclient.NewOptNilString(req.Body),
	}

	if req.Description.IsSet() {
		template.Description = req.Description
	}

	if req.PlaintextBody.IsSet() {
		template.PlaintextBody = req.PlaintextBody
	}

	if req.Preheader.IsSet() {
		template.Preheader = req.Preheader
	}

	if req.ShouldInlineCSS.IsSet() {
		template.ShouldInlineCSS = req.ShouldInlineCSS
	}

	if req.Tags.IsSet() {
		if req.Tags.IsNull() {
			template.Tags.SetToNull()
		} else {
			template.Tags.SetTo(slices.Clone(req.Tags.Value))
		}
	}

	h.emailTemplates[templateID] = template

	return &brazeclient.CreateEmailTemplateResponse{
		EmailTemplateID: templateID,
		Message:         "success",
	}, nil
}

func (h *Handler) UpdateEmailTemplate(ctx context.Context, req *brazeclient.UpdateEmailTemplateRequest) (*brazeclient.UpdateEmailTemplateResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	template, exists := h.emailTemplates[req.EmailTemplateID]
	if !exists {
		return nil, fmt.Errorf("email template not found: %s", req.EmailTemplateID)
	}

	if req.TemplateName.IsSet() {
		if req.TemplateName.Value == "" {
			return nil, newStatusCodeError(http.StatusUnprocessableEntity)
		}
		template.TemplateName = req.TemplateName.Value
	}

	if req.Description.IsSet() {
		template.Description = req.Description
	}

	if req.Subject.IsSet() {
		template.Subject = req.Subject.Value
	}

	if req.Body.IsSet() {
		template.Body.SetTo(req.Body.Value)
	}

	if req.PlaintextBody.IsSet() {
		template.PlaintextBody = req.PlaintextBody
	}

	if req.Preheader.IsSet() {
		template.Preheader = req.Preheader
	}

	if req.ShouldInlineCSS.IsSet() {
		template.ShouldInlineCSS = req.ShouldInlineCSS
	}

	if req.Tags.IsSet() {
		if req.Tags.IsNull() {
			template.Tags.SetToNull()
		} else {
			template.Tags.SetTo(slices.Clone(req.Tags.Value))
		}
	}

	return &brazeclient.UpdateEmailTemplateResponse{
		Message: "success",
	}, nil
}

func (h *Handler) setEmailTemplate(emailTemplateID, name, body, description string, tags []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	block := &brazeclient.GetEmailTemplateInfoResponse{
		EmailTemplateID: emailTemplateID,
		TemplateName:    name,
		Body:            brazeclient.NewOptNilString(body),
	}

	if description != "" {
		block.Description = brazeclient.NewOptNilString(description)
	}

	if tags != nil {
		block.Tags.SetTo(slices.Clone(tags))
	} else {
		block.Tags.SetToNull()
	}

	h.emailTemplates[emailTemplateID] = block
}
