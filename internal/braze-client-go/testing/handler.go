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

	emailTemplates map[string]*brazeclient.GetEmailTemplateInfoResponse
}

var _ brazeclient.Handler = (*Handler)(nil)

func NewBrazeHandler() *Handler {
	return &Handler{
		mu: sync.Mutex{},

		contentBlocks: make(map[string]*brazeclient.GetContentBlockInfoResponse),

		emailTemplates: make(map[string]*brazeclient.GetEmailTemplateInfoResponse),
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
