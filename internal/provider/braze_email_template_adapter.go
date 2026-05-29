package provider

import (
	"context"
	"fmt"
	"time"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type emailTemplateClient interface {
	Create(ctx context.Context, plan brazeEmailTemplateModel) (brazeEmailTemplateModel, error)
	Read(ctx context.Context, id string) (brazeEmailTemplateModel, error)
	Update(ctx context.Context, plan brazeEmailTemplateModel) (brazeEmailTemplateModel, error)
	List(ctx context.Context, query brazeObjectListQuery) ([]brazeObjectListEntry[brazeEmailTemplateModel], error)
}

type generatedEmailTemplateClient struct {
	client *brazeclient.Client
}

type emailTemplateListItem struct {
	item brazeclient.ListEmailTemplatesResponseTemplatesItem
}

func newGeneratedEmailTemplateClient(client *brazeclient.Client) generatedEmailTemplateClient {
	return generatedEmailTemplateClient{client: client}
}

func (c generatedEmailTemplateClient) Create(ctx context.Context, plan brazeEmailTemplateModel) (brazeEmailTemplateModel, error) {
	createRequest := plan.ToCreateEmailTemplateRequest()

	createResponse, createErr := c.client.CreateEmailTemplate(ctx, &createRequest)

	tflog.Info(ctx, "braze_email_template.create", map[string]any{
		"request":  createRequest,
		"response": createResponse,
		"err":      createErr,
	})

	if createErr != nil {
		return brazeEmailTemplateModel{}, fmt.Errorf("create email template: %w", createErr)
	}

	if createResponse == nil {
		return brazeEmailTemplateModel{}, errBrazeObjectEmptyResponse
	}

	return c.Read(ctx, createResponse.GetEmailTemplateID())
}

func (c generatedEmailTemplateClient) Read(ctx context.Context, id string) (brazeEmailTemplateModel, error) {
	getParams := brazeclient.GetEmailTemplateInfoParams{
		EmailTemplateID: id,
	}

	getResponse, getErr := c.client.GetEmailTemplateInfo(ctx, getParams)

	tflog.Info(ctx, "braze_email_template.read", map[string]any{
		"params":   getParams,
		"response": getResponse,
		"err":      getErr,
	})

	if getErr != nil {
		return brazeEmailTemplateModel{}, classifyBrazeObjectReadError(getErr)
	}

	if getResponse == nil {
		return brazeEmailTemplateModel{}, errBrazeObjectEmptyResponse
	}

	return NewBrazeEmailTemplateModelFromGetEmailTemplateInfoResponse(*getResponse), nil
}

func (c generatedEmailTemplateClient) Update(ctx context.Context, plan brazeEmailTemplateModel) (brazeEmailTemplateModel, error) {
	updateRequest := plan.ToUpdateEmailTemplateRequest()

	updateResponse, updateErr := c.client.UpdateEmailTemplate(ctx, &updateRequest)

	tflog.Info(ctx, "braze_email_template.update", map[string]any{
		"request":  updateRequest,
		"response": updateResponse,
		"err":      updateErr,
	})

	if updateErr != nil {
		return brazeEmailTemplateModel{}, fmt.Errorf("update email template: %w", updateErr)
	}

	if updateResponse == nil {
		return brazeEmailTemplateModel{}, errBrazeObjectEmptyResponse
	}

	return c.Read(ctx, updateResponse.GetEmailTemplateID())
}

func (c generatedEmailTemplateClient) List(ctx context.Context, query brazeObjectListQuery) ([]brazeObjectListEntry[brazeEmailTemplateModel], error) {
	return listBrazeObjectEntries(query, func(offset, limit int) ([]emailTemplateListItem, error) {
		return c.listPage(ctx, query, offset, limit)
	}, func(id string) (brazeEmailTemplateModel, error) {
		return c.Read(ctx, id)
	})
}

//nolint:dupl // The generated list endpoint types differ; abstracting this would add callback-heavy plumbing.
func (c generatedEmailTemplateClient) listPage(ctx context.Context, query brazeObjectListQuery, offset, limit int) ([]emailTemplateListItem, error) {
	params := brazeclient.ListEmailTemplatesParams{}

	applyBrazeObjectListQuery(
		query,
		offset,
		limit,
		func(value int) { params.Limit = brazeclient.NewOptInt(value) },
		func(value int) { params.Offset = brazeclient.NewOptInt(value) },
		func(value time.Time) { params.ModifiedAfter = brazeclient.NewOptDateTime(value) },
		func(value time.Time) { params.ModifiedBefore = brazeclient.NewOptDateTime(value) },
	)

	listResponse, listErr := c.client.ListEmailTemplates(ctx, params)

	tflog.Info(ctx, "braze_email_template.list", map[string]any{
		"params":   params,
		"response": listResponse,
		"err":      listErr,
	})

	if listErr != nil {
		return nil, fmt.Errorf("list email templates: %w", listErr)
	}

	if listResponse == nil {
		return nil, errBrazeObjectEmptyResponse
	}

	emailTemplates := listResponse.GetTemplates()

	items := make([]emailTemplateListItem, len(emailTemplates))
	for i, emailTemplate := range emailTemplates {
		items[i] = emailTemplateListItem{item: emailTemplate}
	}

	return items, nil
}

func (i emailTemplateListItem) ListEntry() brazeObjectListEntry[brazeEmailTemplateModel] {
	return brazeObjectListEntry[brazeEmailTemplateModel]{
		ID:          i.item.GetEmailTemplateID(),
		DisplayName: i.item.GetTemplateName(),
	}
}
