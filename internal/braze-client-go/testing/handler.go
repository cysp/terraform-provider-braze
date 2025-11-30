package testing

import (
	"context"
	"errors"
	"sync"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

type Handler struct {
	mu sync.Mutex

	contentBlocks map[string]*brazeclient.GetContentBlockInfoResponse
	// orphanedBlocks are blocks that appear in the list but will fail when getting details
	orphanedBlocks map[string]brazeclient.ListContentBlocksResponseContentBlock
}

var _ brazeclient.Handler = (*Handler)(nil)

func NewBrazeHandler() *Handler {
	return &Handler{
		mu: sync.Mutex{},

		contentBlocks:  make(map[string]*brazeclient.GetContentBlockInfoResponse),
		orphanedBlocks: make(map[string]brazeclient.ListContentBlocksResponseContentBlock),
	}
}

func (h *Handler) NewError(_ context.Context, err error) *brazeclient.ErrorResponseStatusCode {
	var statusCode int

	var sce statusCodeError
	if errors.As(err, &sce) {
		statusCode = sce.StatusCode
	}

	return &brazeclient.ErrorResponseStatusCode{
		StatusCode: statusCode,
		Response: brazeclient.ErrorResponse{
			Message: err.Error(),
		},
	}
}
