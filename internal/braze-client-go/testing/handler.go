package testing

import (
	"sync"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

type Handler struct {
	mu sync.Mutex
}

var _ brazeclient.Handler = (*Handler)(nil)

func NewBrazeHandler() *Handler {
	return &Handler{
		mu: sync.Mutex{},
	}
}
