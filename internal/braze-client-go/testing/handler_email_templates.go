package testing

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"sort"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/google/uuid"
)

func (h *Handler) ListEmailTemplates(_ context.Context, params brazeclient.ListEmailTemplatesParams) (*brazeclient.ListEmailTemplatesResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ids := make([]string, 0, len(h.emailTemplates))
	for id := range h.emailTemplates {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	templates := make([]brazeclient.ListEmailTemplatesResponseTemplatesItem, 0, len(h.emailTemplates))
	for _, id := range ids {
		template := h.emailTemplates[id]
		templates = append(templates, brazeclient.ListEmailTemplatesResponseTemplatesItem{
			EmailTemplateID: template.EmailTemplateID,
			TemplateName:    template.TemplateName,
		})
	}

	return &brazeclient.ListEmailTemplatesResponse{
		Count:     len(templates),
		Templates: paginatedItems(templates, params.Limit, params.Offset),
	}, nil
}

func (h *Handler) GetEmailTemplateInfo(_ context.Context, params brazeclient.GetEmailTemplateInfoParams) (*brazeclient.GetEmailTemplateInfoResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	template, exists := h.emailTemplates[params.EmailTemplateID]
	if !exists {
		return nil, errNotFound
	}

	return template, nil
}

func (h *Handler) CreateEmailTemplate(_ context.Context, req *brazeclient.CreateEmailTemplateRequest) (*brazeclient.CreateEmailTemplateResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if req.TemplateName == "" {
		return nil, newStatusCodeError(http.StatusUnprocessableEntity)
	}

	templateID := uuid.NewString()

	template := &brazeclient.GetEmailTemplateInfoResponse{
		EmailTemplateID: templateID,
		TemplateName:    req.TemplateName,
		Subject:         brazeclient.NewOptNilString(req.Subject.Value),
		Body:            brazeclient.NewOptNilString(req.Body.Value),
		PlaintextBody:   req.PlaintextBody,
		Preheader:       req.Preheader,
		ShouldInlineCSS: req.ShouldInlineCSS,
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
		Message:         brazeclient.NewOptString("success"),
	}, nil
}

func (h *Handler) UpdateEmailTemplate(_ context.Context, req *brazeclient.UpdateEmailTemplateRequest) (*brazeclient.UpdateEmailTemplateResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	template, exists := h.emailTemplates[req.EmailTemplateID]
	if !exists {
		return nil, fmt.Errorf("email template not found: %w", errNotFound)
	}

	templateName, templateNameOk := req.TemplateName.Get()
	if templateNameOk {
		if templateName == "" {
			return nil, newStatusCodeError(http.StatusUnprocessableEntity)
		}

		template.TemplateName = templateName
	}

	if req.Subject.IsSet() {
		template.Subject = req.Subject
	}

	if req.Body.IsSet() {
		template.Body = req.Body
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
		EmailTemplateID: template.EmailTemplateID,
		Message:         brazeclient.NewOptString("success"),
	}, nil
}

func (h *Handler) setEmailTemplate(templateID, templateName, subject, body, plaintextBody, preheader string, tags []string, shouldInlineCSS *bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	template := &brazeclient.GetEmailTemplateInfoResponse{
		EmailTemplateID: templateID,
		TemplateName:    templateName,
	}

	if subject != "" {
		template.Subject = brazeclient.NewOptNilString(subject)
	}

	if body != "" {
		template.Body = brazeclient.NewOptNilString(body)
	}

	if plaintextBody != "" {
		template.PlaintextBody = brazeclient.NewOptNilString(plaintextBody)
	}

	if preheader != "" {
		template.Preheader = brazeclient.NewOptNilString(preheader)
	}

	if tags != nil {
		template.Tags.SetTo(slices.Clone(tags))
	} else {
		template.Tags.SetToNull()
	}

	if shouldInlineCSS != nil {
		template.ShouldInlineCSS.SetTo(*shouldInlineCSS)
	}

	h.emailTemplates[templateID] = template
}
