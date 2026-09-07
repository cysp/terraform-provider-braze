package provider_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	brazeclienttesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pluralSDKAppID   = "01234567-89ab-cdef-0123-456789abcdef"
	pluralSDKAddress = "braze_sdk_authentication_keys.test"
)

type pluralSDKMember struct {
	material    string
	description string
	primary     bool
}

func pluralSDKKey(name string, primary bool) pluralSDKMember {
	return pluralSDKMember{material: "public material " + name, description: name, primary: primary}
}

func pluralSDKConfig(appID string, keys ...pluralSDKMember) string {
	var config strings.Builder
	fmt.Fprintf(&config, "provider \"braze\" {}\nresource \"braze_sdk_authentication_keys\" \"test\" {\n app_id = %q\n keys = [\n", appID)

	for _, key := range keys {
		fmt.Fprintf(&config, "{rsa_public_key = %q, description = %q, primary = %t},\n", key.material, key.description, key.primary)
	}

	config.WriteString("]\n}\n")

	return config.String()
}

type pluralSDKRequest struct {
	method string
	body   map[string]any
}

type pluralSDKFixture struct {
	t       *testing.T
	server  *brazeclienttesting.Server
	mu      sync.Mutex
	writes  []pluralSDKRequest
	reverse atomic.Bool
}

func newPluralSDKFixture(t *testing.T) *pluralSDKFixture {
	t.Helper()

	server := newBrazeTestServer(t)

	return &pluralSDKFixture{t: t, server: server}
}

func (fixture *pluralSDKFixture) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		body, err := io.ReadAll(req.Body)
		assert.NoError(fixture.t, err)

		var payload map[string]any
		assert.NoError(fixture.t, json.Unmarshal(body, &payload))
		req.Body = io.NopCloser(bytes.NewReader(body))

		fixture.mu.Lock()
		fixture.writes = append(fixture.writes, pluralSDKRequest{method: req.Method, body: payload})
		fixture.mu.Unlock()
	}

	if req.Method == http.MethodGet && fixture.reverse.Load() {
		recorder := httptest.NewRecorder()
		fixture.server.ServeHTTP(recorder, req)

		var response struct {
			Keys []brazeclient.SDKAuthenticationKey `json:"keys"`
		}
		assert.NoError(fixture.t, json.Unmarshal(recorder.Body.Bytes(), &response))
		slices.Reverse(response.Keys)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(recorder.Code)
		assert.NoError(fixture.t, json.NewEncoder(w).Encode(response))

		return
	}

	fixture.server.ServeHTTP(w, req)
}

func (fixture *pluralSDKFixture) resetWrites() {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()

	fixture.writes = nil
}

func (fixture *pluralSDKFixture) requests() []pluralSDKRequest {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()

	return slices.Clone(fixture.writes)
}

func (fixture *pluralSDKFixture) expectMethods(methods ...string) {
	fixture.t.Helper()

	got := make([]string, 0, len(fixture.requests()))
	for _, request := range fixture.requests() {
		got = append(got, request.method)
	}

	assert.Equal(fixture.t, strings.Join(methods, ","), strings.Join(got, ","))
}

func (fixture *pluralSDKFixture) remote(ctx context.Context, appID string) []brazeclient.SDKAuthenticationKey {
	fixture.t.Helper()
	response, err := fixture.server.Handler().ListSDKAuthenticationKeys(ctx, brazeclient.ListSDKAuthenticationKeysParams{AppID: appID})
	require.NoError(fixture.t, err)

	return response.Response.Keys
}

func (fixture *pluralSDKFixture) seed(id string, key pluralSDKMember) {
	fixture.server.SetSDKAuthenticationKey(pluralSDKAppID, brazeclient.SDKAuthenticationKey{
		ID: id, RsaPublicKey: key.material, Description: key.description, IsPrimary: key.primary,
	})
}

// Compare full returned objects with Terraform state independently of set order;
// retain IDs in this oracle so duplicate response members cannot disappear.
type pluralSDKStateCheck struct {
	fixture *pluralSDKFixture
	appID   string
	keys    []pluralSDKMember
	after   func([]brazeclient.SDKAuthenticationKey)
}

func (check pluralSDKStateCheck) CheckState(ctx context.Context, req statecheck.CheckStateRequest, _ *statecheck.CheckStateResponse) {
	t := check.fixture.t
	t.Helper()
	require.NotNil(t, req.State)
	require.NotNil(t, req.State.Values)
	require.NotNil(t, req.State.Values.RootModule)

	var attributes map[string]any

	for _, item := range req.State.Values.RootModule.Resources {
		if item.Address == pluralSDKAddress {
			attributes = item.AttributeValues
		}
	}

	require.NotNil(t, attributes, "empty remote collection must not remove resource from state")
	assert.Equal(t, check.appID, attributes["app_id"])
	remote := check.fixture.remote(ctx, check.appID)
	expected := make([]any, 0, len(remote))

	observed := make([]pluralSDKMember, 0, len(remote))
	for _, key := range remote {
		require.NotEmpty(t, key.ID)
		expected = append(expected, map[string]any{"id": key.ID, "rsa_public_key": key.RsaPublicKey, "description": key.Description, "primary": key.IsPrimary})
		observed = append(observed, pluralSDKMember{material: key.RsaPublicKey, description: key.Description, primary: key.IsPrimary})
	}

	assert.ElementsMatch(t, check.keys, observed)
	assert.ElementsMatch(t, expected, attributes["keys"])

	if check.after != nil {
		check.after(remote)
	}
}

func (fixture *pluralSDKFixture) state(keys []pluralSDKMember, after func([]brazeclient.SDKAuthenticationKey)) []statecheck.StateCheck {
	return []statecheck.StateCheck{pluralSDKStateCheck{fixture: fixture, appID: pluralSDKAppID, keys: keys, after: after}}
}

func pluralSDKIDs(keys []brazeclient.SDKAuthenticationKey) map[string]string {
	ids := make(map[string]string, len(keys))
	for _, key := range keys {
		ids[key.RsaPublicKey+"\x00"+key.Description] = key.ID
	}

	return ids
}
