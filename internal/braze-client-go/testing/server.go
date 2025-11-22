package testing

import (
	"context"
	"net/http"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

type Server struct {
	server *brazeclient.Server

	handler *Handler
}

var _ http.Handler = (*Server)(nil)

type noOpSecurityHandler struct{}

//revive:disable:var-naming
func (noOpSecurityHandler) HandleBrazeApiKey(ctx context.Context, _ brazeclient.OperationName, _ brazeclient.BrazeApiKey) (context.Context, error) {
	return ctx, nil
}

func NewBrazeServer() (*Server, error) {
	handler := NewBrazeHandler()

	server, err := brazeclient.NewServer(handler, noOpSecurityHandler{})
	if err != nil {
		//nolint:wrapcheck
		return nil, err
	}

	return &Server{
		server:  server,
		handler: handler,
	}, nil
}

func (s *Server) Handler() *Handler {
	return s.handler
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.server.ServeHTTP(w, r)
}
