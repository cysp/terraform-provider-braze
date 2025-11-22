package provider_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	. "github.com/cysp/terraform-provider-braze/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func BrazeProviderMockedResourceTest(t *testing.T, server http.Handler, testcase resource.TestCase) {
	t.Helper()

	brazeProviderMockableResourceTest(t, server, true, testcase)
}

func BrazeProviderMockableResourceTest(t *testing.T, server http.Handler, testcase resource.TestCase) {
	t.Helper()

	brazeProviderMockableResourceTest(t, server, false, testcase)
}

func brazeProviderMockableResourceTest(t *testing.T, handler http.Handler, alwaysMock bool, testcase resource.TestCase) {
	t.Helper()

	switch {
	case alwaysMock || os.Getenv("TF_ACC_MOCKED") != "":
		if testcase.ProtoV6ProviderFactories != nil {
			t.Fatal("tc.ProtoV6ProviderFactories must be nil")
		}

		var testserver *httptest.Server
		if handler != nil {
			testserver = httptest.NewServer(handler)
			t.Cleanup(testserver.Close)
		}

		testcase.ProtoV6ProviderFactories = makeTestAccProtoV6ProviderFactories(BrazeProviderOptionsWithHTTPTestServer(testserver)...)
		resource.Test(t, testcase)

	default:
		if testcase.ProtoV6ProviderFactories == nil {
			testcase.ProtoV6ProviderFactories = testAccProtoV6ProviderFactories
		}

		resource.Test(t, testcase)
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
