package provider

import (
	"context"
	"errors"
	"net/http"
	"time"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type contentBlockClient interface {
	Create(ctx context.Context, plan brazeContentBlockModel) (brazeContentBlockModel, error)
	Read(ctx context.Context, id string) (brazeContentBlockModel, error)
	Update(ctx context.Context, plan brazeContentBlockModel) (brazeContentBlockModel, error)
	List(ctx context.Context, query contentBlockListQuery) ([]contentBlockListEntry, error)
}

type contentBlockListQuery struct {
	Limit           int
	ModifiedAfter   *time.Time
	ModifiedBefore  *time.Time
	IncludeResource bool
}

type contentBlockListEntry struct {
	ID          string
	DisplayName string
	Resource    *brazeContentBlockModel
	ResourceErr error
}

type contentBlockNotFoundError struct {
	err error
}

func (e contentBlockNotFoundError) Error() string {
	return e.err.Error()
}

func (e contentBlockNotFoundError) Unwrap() error {
	return e.err
}

func isContentBlockNotFound(err error) bool {
	var notFound contentBlockNotFoundError

	return errors.As(err, &notFound)
}

type generatedContentBlockClient struct {
	client *brazeclient.Client
}

var errContentBlockEmptyResponse = errors.New("empty Braze content block response")

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
		return brazeContentBlockModel{}, createErr
	}

	if createResponse == nil {
		return brazeContentBlockModel{}, errContentBlockEmptyResponse
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
		return brazeContentBlockModel{}, classifyContentBlockReadError(getErr)
	}

	if getResponse == nil {
		return brazeContentBlockModel{}, errContentBlockEmptyResponse
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
		return brazeContentBlockModel{}, updateErr
	}

	if updateResponse == nil {
		return brazeContentBlockModel{}, errContentBlockEmptyResponse
	}

	return c.Read(ctx, updateResponse.GetContentBlockID())
}

func (c generatedContentBlockClient) List(ctx context.Context, query contentBlockListQuery) ([]contentBlockListEntry, error) {
	params := brazeclient.ListContentBlocksParams{
		Limit: brazeclient.NewOptInt(query.Limit),
	}

	if query.ModifiedAfter != nil {
		params.ModifiedAfter = brazeclient.NewOptDateTime(*query.ModifiedAfter)
	}

	if query.ModifiedBefore != nil {
		params.ModifiedBefore = brazeclient.NewOptDateTime(*query.ModifiedBefore)
	}

	listResponse, listErr := c.client.ListContentBlocks(ctx, params)

	tflog.Info(ctx, "braze_content_block.list", map[string]any{
		"params":   params,
		"response": listResponse,
		"err":      listErr,
	})

	if listErr != nil {
		return nil, listErr
	}

	if listResponse == nil {
		return nil, errContentBlockEmptyResponse
	}

	entries := make([]contentBlockListEntry, 0, len(listResponse.GetContentBlocks()))
	for _, block := range listResponse.GetContentBlocks() {
		entry := contentBlockListEntry{
			ID:          block.GetContentBlockID(),
			DisplayName: block.GetName(),
		}

		if query.IncludeResource {
			resource, err := c.Read(ctx, entry.ID)
			if err != nil {
				entry.ResourceErr = err
			} else {
				entry.Resource = &resource
			}
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func classifyContentBlockReadError(err error) error {
	var ersc *brazeclient.ErrorResponseStatusCode
	if errors.As(err, &ersc) && ersc.StatusCode == http.StatusNotFound {
		return contentBlockNotFoundError{err: err}
	}

	return err
}
