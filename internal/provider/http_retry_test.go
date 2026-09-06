//nolint:testpackage
package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPRetriesRespectMutationSafety(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		read          bool
		status, calls int
		success       bool
	}{
		"ambiguous create": {false, http.StatusServiceUnavailable, 1, false},
		"throttled create": {false, http.StatusTooManyRequests, 2, true},
		"transient read":   {true, http.StatusServiceUnavailable, 2, true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var calls atomic.Int32

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				if calls.Add(1) == 1 {
					w.Header().Set("Retry-After", "0")
					w.WriteHeader(test.status)
					_, _ = w.Write([]byte(`{"message":"try later"}`))

					return
				}

				if !test.read {
					w.WriteHeader(http.StatusCreated)
				}

				_, _ = w.Write([]byte(`{"catalogs":[{"name":"items","description":"Items","fields":[{"name":"id","type":"string"}]}],"message":"success"}`))
			}))
			t.Cleanup(server.Close)
			p := NewBrazeProvider("test", WithBaseURL(server.URL), WithHTTPClient(server.Client()))
			configured := configureTestProvider(t, p, nil, "fixture-key", false)
			require.False(t, configured.Diagnostics.HasError())
			data, ok := configured.ResourceData.(brazeProviderData)
			require.True(t, ok)

			client := data.catalogs

			var err error
			if test.read {
				_, err = client.Read(t.Context(), "items")
			} else {
				_, err = client.Create(t.Context(), brazeCatalogModel{
					Name: types.StringValue("items"), Description: types.StringValue("Items"),
					Fields: types.ListValueMust(BrazeCatalogFieldObjectType(), []attr.Value{catalogFieldValue(types.StringValue("id"), types.StringValue("string"))}),
				})
			}

			assert.Equal(t, test.success, err == nil, "%v", err)
			assert.Equal(t, test.calls, int(calls.Load()))
		})
	}
}

func TestHTTPRateLimitWaitIsBoundedAndCancellable(t *testing.T) {
	t.Parallel()

	for _, scenario := range []string{"oversized wait", "cancel wait"} {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()

			var calls atomic.Int32

			received := make(chan struct{}, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)

				if scenario == "oversized wait" {
					w.Header().Set("Retry-After", "600")
				} else {
					w.Header().Set("Retry-After", "120")
				}

				w.WriteHeader(http.StatusTooManyRequests)

				select {
				case received <- struct{}{}:
				default:
				}
			}))
			t.Cleanup(server.Close)

			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()

			if scenario == "cancel wait" {
				go func() {
					select {
					case <-received:
						cancel()
					case <-ctx.Done():
					}
				}()
			}

			request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			require.NoError(t, err)

			response, err := newBrazeHTTPClient(server.Client()).Do(request)
			if response != nil {
				require.NoError(t, response.Body.Close())
			}

			if scenario == "cancel wait" {
				require.ErrorIs(t, err, context.Canceled)
			} else {
				require.NoError(t, err)
				assert.Equal(t, http.StatusTooManyRequests, response.StatusCode)
			}

			assert.EqualValues(t, 1, calls.Load())
		})
	}
}
