package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

var errUnexpectedUpdateContentBlockResponse = errors.New("unexpected update content block response")

type contentBlockListItem struct {
	item brazeclient.ListContentBlocksResponseContentBlock
}

func newGeneratedContentBlockClient(client *brazeclient.Client) generatedContentBlockClient {
	return generatedContentBlockClient{client: client}
}

func (c generatedContentBlockClient) Create(ctx context.Context, plan brazeContentBlockModel) (brazeContentBlockModel, error) {
	createRequest, err := plan.ToCreateContentBlockRequest(ctx)
	if err != nil {
		return brazeContentBlockModel{}, err
	}

	createResponse, createErr := c.client.CreateContentBlock(ctx, &createRequest)

	tflog.Debug(ctx, "braze_content_block.create")

	if createErr != nil {
		return brazeContentBlockModel{}, fmt.Errorf("create content block: %w", createErr)
	}

	if createResponse == nil {
		return brazeContentBlockModel{}, errBrazeObjectEmptyResponse
	}

	objectID := createResponse.GetContentBlockID()
	if objectID == "" {
		return brazeContentBlockModel{}, errBrazeObjectEmptyResponse
	}

	data, err := c.Read(ctx, objectID)
	if err != nil {
		// Preserve the accepted request and generated identity for recovery after a failed create.
		plan.ID = types.StringValue(objectID)

		return plan, fmt.Errorf("created object %s but could not read it: %w", objectID, err)
	}

	return data, nil
}

func (c generatedContentBlockClient) Read(ctx context.Context, objectID string) (brazeContentBlockModel, error) {
	getParams := brazeclient.GetContentBlockInfoParams{
		ContentBlockID: objectID,
	}

	getResponse, getErr := c.client.GetContentBlockInfo(ctx, getParams)

	tflog.Debug(ctx, "braze_content_block.read", map[string]any{
		"params": getParams,
	})

	if getErr != nil {
		return brazeContentBlockModel{}, classifyBrazeObjectReadError(getErr)
	}

	if getResponse == nil {
		return brazeContentBlockModel{}, errBrazeObjectEmptyResponse
	}

	err := validateBrazeObjectID(objectID, getResponse.GetContentBlockID())
	if err != nil {
		return brazeContentBlockModel{}, err
	}

	return NewBrazeContentBlockModelFromGetContentBlockInfoResponse(*getResponse), nil
}

func (c generatedContentBlockClient) Update(ctx context.Context, plan brazeContentBlockModel) (brazeContentBlockModel, error) {
	updateRequest, err := plan.ToUpdateContentBlockRequest(ctx)
	if err != nil {
		return brazeContentBlockModel{}, err
	}

	updateResponse, updateErr := c.client.UpdateContentBlock(ctx, &updateRequest)

	tflog.Debug(ctx, "braze_content_block.update")

	if updateErr != nil {
		return brazeContentBlockModel{}, fmt.Errorf("update content block: %w", updateErr)
	}

	if updateResponse == nil {
		return brazeContentBlockModel{}, errBrazeObjectEmptyResponse
	}

	contentBlockID, err := contentBlockIDFromUpdateContentBlockResponse(updateResponse)
	if err != nil {
		return brazeContentBlockModel{}, err
	}

	err = validateBrazeObjectID(plan.ID.ValueString(), contentBlockID)
	if err != nil {
		return brazeContentBlockModel{}, err
	}

	return c.Read(ctx, contentBlockID)
}

func contentBlockIDFromUpdateContentBlockResponse(response brazeclient.UpdateContentBlockRes) (string, error) {
	switch response := response.(type) {
	case *brazeclient.UpdateContentBlockOK:
		return (*brazeclient.UpdateContentBlockResponse)(response).GetContentBlockID(), nil
	case *brazeclient.UpdateContentBlockCreated:
		return (*brazeclient.UpdateContentBlockResponse)(response).GetContentBlockID(), nil
	default:
		return "", fmt.Errorf("%w: %T", errUnexpectedUpdateContentBlockResponse, response)
	}
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

	tflog.Debug(ctx, "braze_content_block.list", map[string]any{
		"params": params,
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
