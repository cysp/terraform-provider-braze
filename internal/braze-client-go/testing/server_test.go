package testing_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	brazetesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
)

func TestNewBrazeServer(t *testing.T) {
	t.Parallel()

	server, err := brazetesting.NewBrazeServer()
	if err != nil {
		t.Fatalf("NewBrazeServer() error = %v", err)
	}

	if server == nil {
		t.Fatal("NewBrazeServer() returned nil server")
	}

	if server.Handler() == nil {
		t.Fatal("server.Handler() returned nil handler")
	}
}

func TestServerServeHTTP(t *testing.T) {
	t.Parallel()

	server, err := brazetesting.NewBrazeServer()
	if err != nil {
		t.Fatalf("NewBrazeServer() error = %v", err)
	}

	ts := httptest.NewServer(server)
	defer ts.Close()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/nonexistent", nil)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext() error = %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http.DefaultClient.Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status code %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}
