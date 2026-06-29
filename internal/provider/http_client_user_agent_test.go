package provider_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/cysp/terraform-provider-braze/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTestTransport = errors.New("transport failed")

func TestHTTPClientWithUserAgentDo(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		userAgent         string
		requestUserAgent  string
		expectedUserAgent string
	}{
		"sets configured user agent": {
			userAgent:         "terraform-provider-braze/test",
			expectedUserAgent: "terraform-provider-braze/test",
		},
		"preserves existing user agent": {
			userAgent:         "terraform-provider-braze/test",
			requestUserAgent:  "custom-client",
			expectedUserAgent: "custom-client",
		},
		"leaves user agent empty when unconfigured": {},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := provider.NewHTTPClientWithUserAgent(&http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, test.expectedUserAgent, req.Header.Get("User-Agent"))

					return &http.Response{
						StatusCode: http.StatusNoContent,
						Header:     make(http.Header),
						Body:       http.NoBody,
						Request:    req,
					}, nil
				}),
			}, test.userAgent)

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.test", nil)
			require.NoError(t, err)

			if test.requestUserAgent != "" {
				req.Header.Set("User-Agent", test.requestUserAgent)
			}

			resp, err := client.Do(req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.NoError(t, resp.Body.Close())
		})
	}
}

func TestHTTPClientWithUserAgentDoWrapsTransportError(t *testing.T) {
	t.Parallel()

	client := provider.NewHTTPClientWithUserAgent(&http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errTestTransport
		}),
	}, "terraform-provider-braze/test")

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.test", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	if resp != nil {
		require.NoError(t, resp.Body.Close())
	}

	require.ErrorIs(t, err, errTestTransport)
	assert.Contains(t, err.Error(), "http client do")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
