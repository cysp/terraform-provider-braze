package testing

import (
	"net/http"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

type Server struct {
	server *brazeclient.Server

	handler *Handler
}

var _ http.Handler = (*Server)(nil)

func NewBrazeServer() (*Server, error) {
	handler := NewBrazeHandler()

	server, err := brazeclient.NewServer(handler)
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
	http.NotFound(w, r)
}
