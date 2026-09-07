package testing_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	brazeclienttesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogItemPaginationCursor(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		cursor string
		status int
		count  int
		first  string
		next   bool
	}{
		"first page":    {status: http.StatusOK, count: 50, first: "item-00", next: true},
		"last page":     {cursor: "50", status: http.StatusOK, count: 1, first: "item-50"},
		"end":           {cursor: "51", status: http.StatusOK},
		"past end":      {cursor: "99", status: http.StatusOK},
		"negative":      {cursor: "-1", status: http.StatusBadRequest},
		"not a number":  {cursor: "invalid", status: http.StatusBadRequest},
		"integer range": {cursor: "99999999999999999999", status: http.StatusBadRequest},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fixture, err := brazeclienttesting.NewBrazeServer()
			require.NoError(t, err)

			for i := range 51 {
				fixture.SetCatalogItem("products", fmt.Sprintf("item-%02d", i), nil)
			}

			server := httptest.NewServer(fixture)
			t.Cleanup(server.Close)

			endpoint := server.URL + "/catalogs/products/items"
			if test.cursor != "" {
				endpoint += "?cursor=" + url.QueryEscape(test.cursor)
			}

			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, endpoint, nil)
			require.NoError(t, err)
			request.Header.Set("Authorization", "Bearer fixture-key")
			response, err := server.Client().Do(request)
			require.NoError(t, err, "invalid cursors must return a response without panicking")

			defer response.Body.Close()

			require.Equal(t, test.status, response.StatusCode)

			if test.status != http.StatusOK {
				return
			}

			var body struct {
				Items []struct {
					ID string `json:"id"`
				} `json:"items"`
			}
			require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
			require.Len(t, body.Items, test.count)

			if test.count > 0 {
				assert.Equal(t, test.first, body.Items[0].ID)
			}

			if test.next {
				assert.Equal(t, `</catalogs/products/items?cursor=50>; rel="next"`, response.Header.Get("Link"))
			} else {
				assert.Empty(t, response.Header.Get("Link"))
			}
		})
	}
}
