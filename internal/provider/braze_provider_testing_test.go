package provider_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	. "github.com/cysp/terraform-provider-braze/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func BrazeProviderMockedResourceTest(t *testing.T, handler http.Handler, testcase resource.TestCase) {
	t.Helper()

	if testcase.ProtoV6ProviderFactories != nil {
		t.Fatal("testcase.ProtoV6ProviderFactories must be nil")
	}

	var testserver *httptest.Server
	if handler != nil {
		testserver = httptest.NewServer(handler)
		t.Cleanup(testserver.Close)

		if testcase.CheckDestroy == nil {
			testcase.CheckDestroy = checkBrazeDestroy(t, testserver)
		}
	}

	testcase.ProtoV6ProviderFactories = makeTestAccProtoV6ProviderFactories(BrazeProviderOptionsWithHTTPTestServer(testserver)...)
	resource.Test(t, testcase)
}

// Check the remote outcome even for APIs that intentionally retain the object.
//
//nolint:gocognit // Check the four distinct remote destroy contracts in one test hook.
func checkBrazeDestroy(t *testing.T, server *httptest.Server) resource.TestCheckFunc {
	t.Helper()

	return func(state *terraform.State) error {
		for _, item := range state.RootModule().Resources {
			var endpoint string

			status := http.StatusNotFound

			switch item.Type {
			case "braze_catalog":
				endpoint = "/catalogs"
				status = http.StatusOK
			case "braze_catalog_item":
				endpoint = "/catalogs/" + url.PathEscape(item.Primary.Attributes["catalog_name"]) + "/items/" + url.PathEscape(item.Primary.Attributes["item_id"])
			case "braze_content_block":
				endpoint = "/content_blocks/info?content_block_id=" + url.QueryEscape(item.Primary.ID)
				status = http.StatusOK
			case "braze_email_template":
				endpoint = "/templates/email/info?email_template_id=" + url.QueryEscape(item.Primary.ID)
				status = http.StatusOK
			default:
				continue
			}

			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+endpoint, nil)
			if err != nil {
				return fmt.Errorf("build destroy check: %w", err)
			}

			request.Header.Set("Authorization", "Bearer 12345")

			response, err := server.Client().Do(request)
			if err != nil {
				return fmt.Errorf("read remote object after destroy: %w", err)
			}

			body, readErr := io.ReadAll(response.Body)

			closeErr := response.Body.Close()
			if readErr != nil || closeErr != nil {
				return fmt.Errorf("read destroy response: %w", errors.Join(readErr, closeErr))
			}

			if response.StatusCode != status {
				return fmt.Errorf("%w: %s destroy expected HTTP %d, got %d", errUnexpectedDestroyOutcome, item.Type, status, response.StatusCode)
			}

			if item.Type == "braze_catalog" {
				var result struct {
					Catalogs []struct {
						Name string `json:"name"`
					} `json:"catalogs"`
				}

				err := json.Unmarshal(body, &result)
				if err != nil {
					return fmt.Errorf("decode catalogs after destroy: %w", err)
				}

				for _, catalog := range result.Catalogs {
					if catalog.Name == item.Primary.Attributes["name"] {
						return fmt.Errorf("%w: catalog %s still exists after destroy", errUnexpectedDestroyOutcome, catalog.Name)
					}
				}
			}
		}

		return nil
	}
}

func BrazeProviderOptionsWithHTTPTestServer(testserver *httptest.Server) []BrazeProviderOption {
	if testserver == nil {
		return nil
	}

	return []BrazeProviderOption{
		WithBaseURL(testserver.URL),
		WithHTTPClient(testserver.Client()),
		WithAPIKey("12345"),
	}
}

var errUnexpectedDestroyOutcome = errors.New("unexpected remote destroy outcome")
