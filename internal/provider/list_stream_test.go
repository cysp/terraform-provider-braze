//nolint:testpackage
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:contextcheck // Setup uses the test lifetime; only List uses the cancellable operation context.
func httpListStream(ctx context.Context, t *testing.T, listResource list.ListResource, managedResource resource.Resource, handler http.Handler, config map[string]string, limit int64, includeResource bool) list.ListResultsStream {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	p := NewBrazeProvider("test", WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	configured := configureTestProvider(t, p, nil, "fixture-key", false)

	var configuredList resource.ConfigureResponse

	configurable, ok := listResource.(list.ListResourceWithConfigure)
	require.True(t, ok)
	configurable.Configure(t.Context(), resource.ConfigureRequest{ProviderData: configured.ListResourceData}, &configuredList)
	require.False(t, configuredList.Diagnostics.HasError())

	var configSchema list.ListResourceSchemaResponse
	listResource.ListResourceConfigSchema(t.Context(), list.ListResourceSchemaRequest{}, &configSchema)

	values := map[string]tftypes.Value{}

	for name := range configSchema.Schema.Attributes {
		var value any
		if configuredValue, ok := config[name]; ok {
			value = configuredValue
		}

		values[name] = tftypes.NewValue(tftypes.String, value)
	}

	var resourceSchema resource.SchemaResponse
	managedResource.Schema(t.Context(), resource.SchemaRequest{}, &resourceSchema)
	identity := lifecycleIdentity(t, managedResource)

	var stream list.ListResultsStream
	listResource.List(ctx, list.ListRequest{
		Config:         tfsdk.Config{Schema: configSchema.Schema, Raw: tftypes.NewValue(configSchema.Schema.Type().TerraformType(t.Context()), values)},
		ResourceSchema: resourceSchema.Schema, ResourceIdentitySchema: identity.Schema,
		Limit: limit, IncludeResource: includeResource,
	}, &stream)

	return stream
}

func TestContentBlockListStopsWhenConsumerStops(t *testing.T) {
	t.Parallel()

	pages, reads := 0, 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/content_blocks/list" {
			pages++
			items := make([]map[string]string, 0)

			if pages == 1 {
				for i := range 100 {
					items = append(items, map[string]string{"content_block_id": strconv.Itoa(i), "name": fmt.Sprintf("block-%d", i)})
				}
			}

			assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{"count": len(items), "content_blocks": items}))

			return
		}

		reads++

		assert.NoError(t, json.NewEncoder(w).Encode(map[string]string{"content_block_id": r.URL.Query().Get("content_block_id"), "name": "block", "content": "Hello"}))
	})
	stream := httpListStream(t.Context(), t, NewBrazeContentBlockListResource(), NewBrazeContentBlockResource(), handler, nil, 200, true)
	count := 0

	stream.Results(func(result list.ListResult) bool {
		assert.False(t, result.Diagnostics.HasError(), "%v", result.Diagnostics)

		count++

		return false
	})
	assert.Equal(t, 1, count)
	assert.Equal(t, 1, pages)
	assert.Equal(t, 1, reads)
}

func TestCatalogItemListRejectsBrokenContinuation(t *testing.T) {
	t.Parallel()

	for name, link := range map[string]string{
		"repeated":  `</catalogs/products/items?cursor=again>; rel="next"`,
		"malformed": `</catalogs/products/items>; rel="next"`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			calls := 0
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++

				w.Header().Set("Content-Type", "application/json")

				if calls > 2 {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"message":"fixture stopped an endless cursor loop"}`))

					return
				}

				w.Header().Set("Link", link)
				_, _ = w.Write([]byte(`{"items":[],"message":"success"}`))
			})
			stream := httpListStream(t.Context(), t, NewBrazeCatalogItemListResource(), NewBrazeCatalogItemResource(), handler, map[string]string{"catalog_name": "products"}, 10, false)
			errors := 0

			stream.Results(func(result list.ListResult) bool {
				if result.Diagnostics.HasError() {
					errors++

					assert.Contains(t, result.Diagnostics.Errors()[0].Detail(), "cursor")
				}

				return true
			})
			assert.Equal(t, 1, errors)

			if name == "repeated" {
				assert.Equal(t, 2, calls)
			} else {
				assert.Equal(t, 1, calls)
			}
		})
	}
}

func TestListStopsOnCancellationAndReportsPartialFailures(t *testing.T) {
	t.Parallel()

	for _, scenario := range []string{"cancel", "second page fails", "enrichment disappears"} {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			pages, reads := 0, 0
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				if r.URL.Path != "/content_blocks/list" {
					reads++

					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"message":"not found"}`))

					return
				}

				pages++
				if pages > 1 {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(`{"message":"permission denied"}`))

					return
				}

				items := make([]map[string]string, 100)
				for i := range items {
					items[i] = map[string]string{"content_block_id": strconv.Itoa(i), "name": "block"}
				}

				assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{"count": 100, "content_blocks": items}))
			})
			stream := httpListStream(ctx, t, NewBrazeContentBlockListResource(), NewBrazeContentBlockResource(), handler, nil, 200, scenario == "enrichment disappears")
			results, errors := 0, 0

			stream.Results(func(result list.ListResult) bool {
				if result.Diagnostics.HasError() {
					errors++

					return false
				}

				results++

				if scenario == "cancel" {
					cancel()
				}

				return true
			})
			assert.Equal(t, 1, errors)

			switch scenario {
			case "cancel":
				assert.Equal(t, 1, results)
				assert.Equal(t, 1, pages)
				assert.Zero(t, reads)
			case "second page fails":
				assert.Equal(t, 100, results)
				assert.Equal(t, 2, pages)
				assert.Zero(t, reads)
			case "enrichment disappears":
				assert.Zero(t, results)
				assert.Equal(t, 1, pages)
				assert.Equal(t, 1, reads)
			}
		})
	}
}
