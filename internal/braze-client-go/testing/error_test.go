package testing_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	brazetesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type errorResponseHandler struct {
	*brazetesting.Handler

	failure error
}

var errInjectedHandlerFailure = errors.New("injected handler failure")

func (handler errorResponseHandler) ListCatalogs(ctx context.Context) (*brazeclient.ListCatalogsResponse, error) {
	if handler.failure != nil {
		return nil, handler.failure
	}

	return handler.Handler.ListCatalogs(ctx) //nolint:wrapcheck // Preserve the error being tested by the HTTP response mapper.
}

type errorResponseSecurityHandler struct{}

//nolint:revive // The generated security interface defines the method name.
func (errorResponseSecurityHandler) HandleBrazeApiKey(ctx context.Context, _ brazeclient.OperationName, _ brazeclient.BrazeApiKey) (context.Context, error) {
	return ctx, nil
}

func TestServerErrorResponses(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		method, path, body, contentType string
		authorization                   bool
		failure                         error
		status                          int
	}{
		"missing credentials": {method: http.MethodGet, path: "/catalogs", status: http.StatusUnauthorized},
		"malformed body": {
			method: http.MethodPost, path: "/content_blocks/create", body: `{`, contentType: "application/json",
			authorization: true, status: http.StatusBadRequest,
		},
		"unsupported content type": {
			method: http.MethodPost, path: "/content_blocks/create", body: `{}`, contentType: "text/plain",
			authorization: true, status: http.StatusUnsupportedMediaType,
		},
		"typed not found": {
			method: http.MethodGet, path: "/content_blocks/info?content_block_id=missing",
			authorization: true, status: http.StatusNotFound,
		},
		"wrapped typed not found": {
			method: http.MethodPost, path: "/content_blocks/update", body: `{"content_block_id":"missing","name":"Welcome"}`, contentType: "application/json",
			authorization: true, status: http.StatusNotFound,
		},
		"typed validation error": {
			method: http.MethodPost, path: "/content_blocks/create", body: `{"name":"","content":"Hello"}`, contentType: "application/json",
			authorization: true, status: http.StatusUnprocessableEntity,
		},
		"unclassified handler failure": {
			method: http.MethodGet, path: "/catalogs", failure: errInjectedHandlerFailure,
			authorization: true, status: http.StatusInternalServerError,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			handler := errorResponseHandler{Handler: brazetesting.NewBrazeHandler(), failure: test.failure}
			server, err := brazeclient.NewServer(handler, errorResponseSecurityHandler{})
			require.NoError(t, err)
			request := httptest.NewRequestWithContext(t.Context(), test.method, test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", test.contentType)

			if test.authorization {
				request.Header.Set("Authorization", "Bearer fixture-key")
			}

			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			assert.Equal(t, test.status, response.Code, "%s", response.Body.String())

			var body struct {
				Message      string `json:"message"`
				ErrorMessage string `json:"error_message"` //nolint:tagliatelle // ogen's default error response.
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
			assert.NotEmpty(t, body.Message+body.ErrorMessage)
		})
	}
}
