package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type contentBlockClient interface {
	Create(ctx context.Context, plan brazeContentBlockModel) (brazeContentBlockModel, error)
	Read(ctx context.Context, id string) (brazeContentBlockModel, error)
	Update(ctx context.Context, plan brazeContentBlockModel) (brazeContentBlockModel, error)
	List(ctx context.Context, query brazeObjectListQuery) ([]brazeObjectListEntry[brazeContentBlockModel], error)
}

type generatedContentBlockClient struct {
	client *brazeclient.Client
}

type contentBlockListItem struct {
	item brazeclient.ListContentBlocksResponseContentBlock
}

func newGeneratedContentBlockClient(client *brazeclient.Client) generatedContentBlockClient {
	return generatedContentBlockClient{client: client}
}

func (c generatedContentBlockClient) Create(ctx context.Context, plan brazeContentBlockModel) (brazeContentBlockModel, error) {
	createRequest := plan.ToCreateContentBlockRequest()

	createResponse, createErr := c.client.CreateContentBlock(ctx, &createRequest)

	tflog.Info(ctx, "braze_content_block.create", map[string]any{
		"request":  createRequest,
		"response": createResponse,
		"err":      createErr,
	})

	if createErr != nil {
		return brazeContentBlockModel{}, fmt.Errorf("create content block: %w", createErr)
	}

	if createResponse == nil {
		return brazeContentBlockModel{}, errBrazeObjectEmptyResponse
	}

	return c.Read(ctx, createResponse.GetContentBlockID())
}

func (c generatedContentBlockClient) Read(ctx context.Context, id string) (brazeContentBlockModel, error) {
	getParams := brazeclient.GetContentBlockInfoParams{
		ContentBlockID: id,
	}

	getResponse, getErr := c.client.GetContentBlockInfo(ctx, getParams)

	tflog.Info(ctx, "braze_content_block.read", map[string]any{
		"params":   getParams,
		"response": getResponse,
		"err":      getErr,
	})

	if getErr != nil {
		return brazeContentBlockModel{}, classifyBrazeObjectReadError(getErr)
	}

	if getResponse == nil {
		return brazeContentBlockModel{}, errBrazeObjectEmptyResponse
	}

	return NewBrazeContentBlockModelFromGetContentBlockInfoResponse(*getResponse), nil
}

func (c generatedContentBlockClient) Update(ctx context.Context, plan brazeContentBlockModel) (brazeContentBlockModel, error) {
	updateRequest := plan.ToUpdateContentBlockRequest()

	updateResponse, updateErr := c.client.UpdateContentBlock(ctx, &updateRequest)

	tflog.Info(ctx, "braze_content_block.update", map[string]any{
		"request":  updateRequest,
		"response": updateResponse,
		"err":      updateErr,
	})

	if updateErr != nil {
		return brazeContentBlockModel{}, fmt.Errorf("update content block: %w", updateErr)
	}

	if updateResponse == nil {
		return brazeContentBlockModel{}, errBrazeObjectEmptyResponse
	}

	return c.Read(ctx, updateResponse.GetContentBlockID())
}

func (c generatedContentBlockClient) List(ctx context.Context, query brazeObjectListQuery) ([]brazeObjectListEntry[brazeContentBlockModel], error) {
	return listBrazeObjectEntries(query, func(offset, limit int) ([]contentBlockListItem, error) {
		return c.listPage(ctx, query, offset, limit)
	}, func(id string) (brazeContentBlockModel, error) {
		return c.Read(ctx, id)
	})
}

//nolint:dupl // The generated list endpoint types differ; abstracting this would add callback-heavy plumbing.
func (c generatedContentBlockClient) listPage(ctx context.Context, query brazeObjectListQuery, offset, limit int) ([]contentBlockListItem, error) {
	params := brazeclient.ListContentBlocksParams{}

	applyBrazeObjectListQuery(
		query,
		offset,
		limit,
		func(value int) { params.Limit = brazeclient.NewOptInt(value) },
		func(value int) { params.Offset = brazeclient.NewOptInt(value) },
		func(value time.Time) { params.ModifiedAfter = brazeclient.NewOptDateTime(value) },
		func(value time.Time) { params.ModifiedBefore = brazeclient.NewOptDateTime(value) },
	)

	listResponse, listErr := c.client.ListContentBlocks(ctx, params)

	tflog.Info(ctx, "braze_content_block.list", map[string]any{
		"params":   params,
		"response": listResponse,
		"err":      listErr,
	})

	if listErr != nil {
		return nil, fmt.Errorf("list content blocks: %w", listErr)
	}

	if listResponse == nil {
		return nil, errBrazeObjectEmptyResponse
	}

	contentBlocks := listResponse.GetContentBlocks()

	items := make([]contentBlockListItem, len(contentBlocks))
	for i, contentBlock := range contentBlocks {
		items[i] = contentBlockListItem{item: contentBlock}
	}

	return items, nil
}

func (i contentBlockListItem) ListEntry() brazeObjectListEntry[brazeContentBlockModel] {
	return brazeObjectListEntry[brazeContentBlockModel]{
		ID:          i.item.GetContentBlockID(),
		DisplayName: i.item.GetName(),
	}
}

func classifyBrazeObjectReadError(err error) error {
	var ersc *brazeclient.ErrorResponseStatusCode
	if errors.As(err, &ersc) && ersc.StatusCode == http.StatusNotFound {
		return brazeObjectNotFoundError{err: err}
	}

	return err
}
