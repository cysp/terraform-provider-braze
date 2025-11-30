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

func TestOrphanedContentBlock(t *testing.T) {
	t.Parallel()

	server, err := brazetesting.NewBrazeServer()
	if err != nil {
		t.Fatalf("NewBrazeServer() error = %v", err)
	}

	// Set up a valid content block and an orphaned one
	server.SetContentBlock("valid-id", "valid-block", "<p>Content</p>", "", []string{})
	server.SetOrphanedContentBlock("orphaned-id", "orphaned-block", []string{})

	ts := httptest.NewServer(server)
	defer ts.Close()

	// List should return both blocks
	listReq, err := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/content_blocks/list", nil)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext() error = %v", err)
	}

	listResp, err := http.DefaultClient.Do(listReq)
	if err != nil {
		t.Fatalf("http.DefaultClient.Do() error = %v", err)
	}
	defer listResp.Body.Close()

	if listResp.StatusCode != http.StatusOK {
		t.Errorf("list expected status code %d, got %d", http.StatusOK, listResp.StatusCode)
	}

	// Get valid block should succeed
	validReq, err := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/content_blocks/info?content_block_id=valid-id", nil)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext() error = %v", err)
	}
	validReq.Header.Set("Authorization", "Bearer test-key")

	validResp, err := http.DefaultClient.Do(validReq)
	if err != nil {
		t.Fatalf("http.DefaultClient.Do() error = %v", err)
	}
	defer validResp.Body.Close()

	if validResp.StatusCode != http.StatusOK {
		t.Errorf("valid block expected status code %d, got %d", http.StatusOK, validResp.StatusCode)
	}

	// Get orphaned block should fail with 404
	orphanedReq, err := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/content_blocks/info?content_block_id=orphaned-id", nil)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext() error = %v", err)
	}
	orphanedReq.Header.Set("Authorization", "Bearer test-key")

	orphanedResp, err := http.DefaultClient.Do(orphanedReq)
	if err != nil {
		t.Fatalf("http.DefaultClient.Do() error = %v", err)
	}
	defer orphanedResp.Body.Close()

	t.Logf("orphanedResp.StatusCode: %d", orphanedResp.StatusCode)
	if orphanedResp.StatusCode != http.StatusNotFound {
		t.Errorf("orphaned block expected status code %d, got %d", http.StatusNotFound, orphanedResp.StatusCode)
	}
}
