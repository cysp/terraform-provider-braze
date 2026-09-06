package provider_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	brazeclienttesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
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

	server, err := brazeclienttesting.NewBrazeServer()
	require.NoError(t, err)

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

func TestAccBrazeSDKAuthenticationKeysLifecycle(t *testing.T) {
	t.Parallel()
	fixture := newPluralSDKFixture(t)
	keys := []pluralSDKMember{pluralSDKKey("A", true), pluralSDKKey("B", false)}
	config := pluralSDKConfig(pluralSDKAppID, keys...)

	var retained []brazeclient.SDKAuthenticationKey

	BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
		CheckDestroy: func(_ *terraform.State) error {
			assert.ElementsMatch(t, retained, fixture.remote(t.Context(), pluralSDKAppID))
			fixture.expectMethods()

			return nil
		},
		Steps: []resource.TestStep{
			{Config: config, ConfigStateChecks: fixture.state(keys, func(remote []brazeclient.SDKAuthenticationKey) {
				retained = remote

				fixture.expectMethods(http.MethodPost, http.MethodPost)
			})},
			{
				PreConfig: func() { fixture.resetWrites(); fixture.reverse.Store(true) }, Config: config,
				ConfigPlanChecks:  resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				ConfigStateChecks: fixture.state(keys, func(_ []brazeclient.SDKAuthenticationKey) { fixture.expectMethods() }),
			},
			{Config: config, ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "app_id"},
			{Config: config, ResourceName: pluralSDKAddress, ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity},
			{Config: config, ConfigStateChecks: fixture.state(keys, func(remote []brazeclient.SDKAuthenticationKey) {
				assert.ElementsMatch(t, retained, remote)
				fixture.expectMethods()
			})},
		},
	})
}

func TestAccBrazeSDKAuthenticationKeysPromotionAndDrift(t *testing.T) {
	t.Parallel()
	fixture := newPluralSDKFixture(t)
	before := []pluralSDKMember{pluralSDKKey("A", true), pluralSDKKey("B", false)}
	after := []pluralSDKMember{pluralSDKKey("B", true), pluralSDKKey("A", false)}

	var initial map[string]string

	var remoteKeys []brazeclient.SDKAuthenticationKey

	checkPromotion := func(remote []brazeclient.SDKAuthenticationKey) {
		assert.Equal(t, initial, pluralSDKIDs(remote))
		fixture.expectMethods(http.MethodPut)
		requests := fixture.requests()
		require.Len(t, requests, 1)
		assert.Equal(t, initial["public material B\x00B"], requests[0].body["key_id"])

		remoteKeys = remote
	}
	BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{Steps: []resource.TestStep{
		{Config: pluralSDKConfig(pluralSDKAppID, before...), ConfigStateChecks: fixture.state(before, func(remote []brazeclient.SDKAuthenticationKey) { initial = pluralSDKIDs(remote) })},
		{PreConfig: fixture.resetWrites, Config: pluralSDKConfig(pluralSDKAppID, after...), ConfigStateChecks: fixture.state(after, checkPromotion)},
		{PreConfig: func() {
			fixture.resetWrites()

			for _, key := range remoteKeys {
				if key.Description == "A" {
					key.IsPrimary = true
					fixture.server.SetSDKAuthenticationKey(pluralSDKAppID, key)
				}
			}
		}, Config: pluralSDKConfig(pluralSDKAppID, after...), ConfigStateChecks: fixture.state(after, checkPromotion)},
	}})
}

func TestAccBrazeSDKAuthenticationKeysImmutableMembers(t *testing.T) {
	t.Parallel()

	for _, field := range []string{"description", "rsa_public_key"} {
		t.Run(field, func(t *testing.T) {
			t.Parallel()
			fixture := newPluralSDKFixture(t)
			before := []pluralSDKMember{pluralSDKKey("A", true), pluralSDKKey("B", false)}

			after := slices.Clone(before)
			if field == "description" {
				after[1].description = "Replacement B"
			} else {
				after[1].material = "replacement public material B"
			}

			var initial map[string]string

			BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{Steps: []resource.TestStep{
				{Config: pluralSDKConfig(pluralSDKAppID, before...), ConfigStateChecks: fixture.state(before, func(remote []brazeclient.SDKAuthenticationKey) { initial = pluralSDKIDs(remote) })},
				{PreConfig: fixture.resetWrites, Config: pluralSDKConfig(pluralSDKAppID, after...), ConfigStateChecks: fixture.state(after, func(remote []brazeclient.SDKAuthenticationKey) {
					ids := pluralSDKIDs(remote)
					assert.Equal(t, initial["public material A\x00A"], ids["public material A\x00A"])

					for _, key := range remote {
						assert.NotEqual(t, initial["public material B\x00B"], key.ID)
					}

					fixture.expectMethods(http.MethodPost, http.MethodDelete)
					requests := fixture.requests()
					require.Len(t, requests, 2)
					assert.Equal(t, initial["public material B\x00B"], requests[1].body["key_id"])
				})},
			}})
		})
	}
}

func TestAccBrazeSDKAuthenticationKeysMembershipDrift(t *testing.T) {
	t.Parallel()
	fixture := newPluralSDKFixture(t)
	keys := []pluralSDKMember{pluralSDKKey("A", true), pluralSDKKey("B", false)}

	var initial []brazeclient.SDKAuthenticationKey

	config := pluralSDKConfig(pluralSDKAppID, keys...)
	BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{Steps: []resource.TestStep{
		{Config: config, ConfigStateChecks: fixture.state(keys, func(remote []brazeclient.SDKAuthenticationKey) { initial = remote })},
		{
			PreConfig: func() { fixture.resetWrites(); fixture.seed("unexpected", pluralSDKKey("C", false)) }, Config: config,
			ConfigStateChecks: fixture.state(keys, func(remote []brazeclient.SDKAuthenticationKey) {
				assert.ElementsMatch(t, initial, remote)
				fixture.expectMethods(http.MethodDelete)
				require.Len(t, fixture.requests(), 1)
				assert.Equal(t, "unexpected", fixture.requests()[0].body["key_id"])
			}),
		},
		{PreConfig: func() {
			fixture.resetWrites()
			fixture.server.ResetSDKAuthenticationKeys(pluralSDKAppID)

			for _, key := range initial {
				if key.IsPrimary {
					fixture.server.SetSDKAuthenticationKey(pluralSDKAppID, key)
				}
			}
		}, Config: config, ConfigStateChecks: fixture.state(keys, func(remote []brazeclient.SDKAuthenticationKey) {
			fixture.expectMethods(http.MethodPost)

			old := pluralSDKIDs(initial)
			ids := pluralSDKIDs(remote)
			assert.Equal(t, old["public material A\x00A"], ids["public material A\x00A"])
			assert.NotEqual(t, old["public material B\x00B"], ids["public material B\x00B"])
		})},
		// A refresh-only step must preserve the aggregate even when Braze returns no keys.
		{
			PreConfig: func() { fixture.resetWrites(); fixture.server.ResetSDKAuthenticationKeys(pluralSDKAppID) }, RefreshState: true, ExpectNonEmptyPlan: true,
			Check: func(state *terraform.State) error {
				fixture.expectMethods()
				require.Empty(t, fixture.remote(t.Context(), pluralSDKAppID))

				item := state.RootModule().Resources[pluralSDKAddress]
				require.NotNil(t, item, "empty remote collection must preserve resource state")
				assert.Equal(t, pluralSDKAppID, item.Primary.Attributes["app_id"])
				assert.Equal(t, "0", item.Primary.Attributes["keys.#"])

				return nil
			},
		},
		{Config: config, ConfigStateChecks: fixture.state(keys, func(remote []brazeclient.SDKAuthenticationKey) {
			fixture.expectMethods(http.MethodPost, http.MethodPost)

			for _, key := range remote {
				for _, old := range initial {
					assert.NotEqual(t, old.ID, key.ID)
				}
			}
		})},
	}})
}

func TestAccBrazeSDKAuthenticationKeysCapacityOrdering(t *testing.T) {
	t.Parallel()
	fixture := newPluralSDKFixture(t)
	before := []pluralSDKMember{pluralSDKKey("A", true), pluralSDKKey("B", false), pluralSDKKey("C", false)}
	after := []pluralSDKMember{pluralSDKKey("B", false), pluralSDKKey("D", true)}

	for _, key := range before {
		fixture.seed(key.description, key)
	}

	config := pluralSDKConfig(pluralSDKAppID, before...)
	BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{Steps: []resource.TestStep{
		{Config: config, ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStatePersist: true},
		{PreConfig: fixture.resetWrites, Config: pluralSDKConfig(pluralSDKAppID, after...), ConfigStateChecks: fixture.state(after, func(remote []brazeclient.SDKAuthenticationKey) {
			fixture.expectMethods(http.MethodDelete, http.MethodPost, http.MethodDelete)
			requests := fixture.requests()
			require.Len(t, requests, 3)
			assert.Equal(t, "C", requests[0].body["key_id"])
			assert.Equal(t, true, requests[1].body["make_primary"])
			assert.Equal(t, "A", requests[2].body["key_id"])
			assert.Equal(t, "B", pluralSDKIDs(remote)["public material B\x00B"])
		})},
	}})
}

func TestAccBrazeSDKAuthenticationKeysExplicitIntermediatePrimary(t *testing.T) {
	t.Parallel()
	fixture := newPluralSDKFixture(t)
	before := []pluralSDKMember{pluralSDKKey("A", true), pluralSDKKey("B", false), pluralSDKKey("C", false)}
	intermediate := []pluralSDKMember{pluralSDKKey("A", false), pluralSDKKey("B", true), pluralSDKKey("C", false)}
	after := []pluralSDKMember{pluralSDKKey("B", false), pluralSDKKey("C", false), pluralSDKKey("D", true)}

	for _, key := range before {
		fixture.seed(key.description, key)
	}

	BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{Steps: []resource.TestStep{
		{Config: pluralSDKConfig(pluralSDKAppID, before...), ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStatePersist: true},
		{PreConfig: fixture.resetWrites, Config: pluralSDKConfig(pluralSDKAppID, after...), PlanOnly: true, ExpectError: regexp.MustCompile(`Cannot reconcile SDK Authentication Keys`)},
		{
			PreConfig: func() { fixture.expectMethods(); fixture.resetWrites() }, Config: pluralSDKConfig(pluralSDKAppID, intermediate...),
			ConfigStateChecks: fixture.state(intermediate, func(remote []brazeclient.SDKAuthenticationKey) {
				fixture.expectMethods(http.MethodPut)
				require.Len(t, fixture.requests(), 1)
				assert.Equal(t, "B", fixture.requests()[0].body["key_id"])

				for _, key := range remote {
					assert.Equal(t, key.Description, key.ID)
				}
			}),
		},
		{PreConfig: fixture.resetWrites, Config: pluralSDKConfig(pluralSDKAppID, after...), ConfigStateChecks: fixture.state(after, func(remote []brazeclient.SDKAuthenticationKey) {
			fixture.expectMethods(http.MethodDelete, http.MethodPost)
			require.Len(t, fixture.requests(), 2)
			assert.Equal(t, "A", fixture.requests()[0].body["key_id"])
			assert.Equal(t, true, fixture.requests()[1].body["make_primary"])

			for _, key := range remote {
				if key.Description != "D" {
					assert.Equal(t, key.Description, key.ID)
				}
			}
		})},
	}})
}

func TestAccBrazeSDKAuthenticationKeysDuplicateConfiguration(t *testing.T) {
	t.Parallel()
	fixture := newPluralSDKFixture(t)
	first := pluralSDKKey("A", true)
	duplicate := first
	duplicate.primary = false
	distinct := pluralSDKKey("B", false)
	distinct.description = first.description
	keys := []pluralSDKMember{first, distinct}
	BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{Steps: []resource.TestStep{
		{Config: pluralSDKConfig(pluralSDKAppID, first, duplicate), PlanOnly: true, ExpectError: regexp.MustCompile(`(?i)(duplicate|ambiguous)`)},
		{PreConfig: func() { fixture.expectMethods() }, Config: pluralSDKConfig(pluralSDKAppID, keys...), ConfigStateChecks: fixture.state(keys, func(remote []brazeclient.SDKAuthenticationKey) {
			require.Len(t, remote, 2)
			assert.NotEqual(t, remote[0].ID, remote[1].ID)
			fixture.expectMethods(http.MethodPost, http.MethodPost)
		})},
	}})
}

func TestAccBrazeSDKAuthenticationKeysNonemptyCreateRequiresImport(t *testing.T) {
	t.Parallel()
	fixture := newPluralSDKFixture(t)
	keys := []pluralSDKMember{pluralSDKKey("A", true)}
	fixture.seed("existing-A", keys[0])
	config := pluralSDKConfig(pluralSDKAppID, keys...)
	BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{Steps: []resource.TestStep{
		{Config: config, ExpectError: regexp.MustCompile(`SDK Authentication Keys require import`)},
		{PreConfig: func() { fixture.expectMethods() }, Config: config, ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStatePersist: true},
		{Config: config, ConfigStateChecks: fixture.state(keys, func(remote []brazeclient.SDKAuthenticationKey) {
			fixture.expectMethods()
			assert.Equal(t, "existing-A", remote[0].ID)
		})},
	}})
}

func TestAccBrazeSDKAuthenticationKeysAppChangeRetainsAll(t *testing.T) {
	t.Parallel()
	fixture := newPluralSDKFixture(t)

	const nextAppID = "fedcba98-7654-3210-fedc-ba9876543210"

	keys := []pluralSDKMember{pluralSDKKey("A", true), pluralSDKKey("B", false)}

	var oldKeys, newKeys []brazeclient.SDKAuthenticationKey

	BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{
		CheckDestroy: func(_ *terraform.State) error {
			assert.ElementsMatch(t, oldKeys, fixture.remote(t.Context(), pluralSDKAppID))
			assert.ElementsMatch(t, newKeys, fixture.remote(t.Context(), nextAppID))
			fixture.expectMethods()

			return nil
		},
		Steps: []resource.TestStep{
			{Config: pluralSDKConfig(pluralSDKAppID, keys...), ConfigStateChecks: fixture.state(keys, func(remote []brazeclient.SDKAuthenticationKey) { oldKeys = remote })},
			{
				PreConfig: fixture.resetWrites, Config: pluralSDKConfig(nextAppID, keys...),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(pluralSDKAddress, plancheck.ResourceActionDestroyBeforeCreate)}},
				ConfigStateChecks: []statecheck.StateCheck{pluralSDKStateCheck{fixture: fixture, appID: nextAppID, keys: keys, after: func(remote []brazeclient.SDKAuthenticationKey) {
					newKeys = remote
					assert.ElementsMatch(t, oldKeys, fixture.remote(t.Context(), pluralSDKAppID))
					fixture.expectMethods(http.MethodPost, http.MethodPost)

					for _, request := range fixture.requests() {
						assert.Equal(t, nextAppID, request.body["app_id"])
					}

					fixture.resetWrites()
				}}},
			},
		},
	})
}

func TestAccBrazeSDKAuthenticationKeysCreateVerificationRecovery(t *testing.T) {
	t.Parallel()
	fixture := newPluralSDKFixture(t)

	var failedVerification atomic.Bool
	failedVerification.Store(true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet && failedVerification.Load() && len(fixture.requests()) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"verification unavailable"}`))

			return
		}

		fixture.ServeHTTP(w, req)
	})
	keys := []pluralSDKMember{pluralSDKKey("A", true)}
	config := pluralSDKConfig(pluralSDKAppID, keys...)

	var confirmed []brazeclient.SDKAuthenticationKey

	BrazeProviderMockedResourceTest(t, handler, resource.TestCase{
		CheckDestroy: func(_ *terraform.State) error {
			assert.ElementsMatch(t, confirmed, fixture.remote(t.Context(), pluralSDKAppID))
			fixture.expectMethods(http.MethodPost)

			return nil
		},
		Steps: []resource.TestStep{
			{Config: config, ExpectError: regexp.MustCompile(`Failed to create SDK Authentication Keys`)},
			// Failed Create taints the aggregate. Replacement relinquishes it, but must
			// refuse to adopt the confirmed key or send a second create request.
			{PreConfig: func() {
				confirmed = fixture.remote(t.Context(), pluralSDKAppID)
				require.Len(t, confirmed, 1)
				fixture.expectMethods(http.MethodPost)
				failedVerification.Store(false)
			}, Config: config, ExpectError: regexp.MustCompile(`SDK Authentication Keys require import`)},
			{PreConfig: func() {
				fixture.expectMethods(http.MethodPost)
				assert.ElementsMatch(t, confirmed, fixture.remote(t.Context(), pluralSDKAppID))
			}, Config: `
provider "braze" {}
removed {
 from = braze_sdk_authentication_keys.test
 lifecycle { destroy = false }
}
`},
			{Config: config, ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStatePersist: true},
			{Config: config, ConfigStateChecks: fixture.state(keys, func(remote []brazeclient.SDKAuthenticationKey) {
				assert.ElementsMatch(t, confirmed, remote)
				fixture.expectMethods(http.MethodPost)
			})},
		},
	})
}

type pluralSDKPlanHook struct{ run func() }

func (hook pluralSDKPlanHook) CheckPlan(_ context.Context, _ plancheck.CheckPlanRequest, _ *plancheck.CheckPlanResponse) {
	hook.run()
}

func TestAccBrazeSDKAuthenticationKeysConcurrentChangeBeforeApply(t *testing.T) {
	t.Parallel()
	fixture := newPluralSDKFixture(t)
	before := []pluralSDKMember{pluralSDKKey("A", true), pluralSDKKey("B", false)}
	after := []pluralSDKMember{pluralSDKKey("A", false), pluralSDKKey("B", true)}

	for _, key := range before {
		fixture.seed(key.description, key)
	}

	BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{Steps: []resource.TestStep{
		{Config: pluralSDKConfig(pluralSDKAppID, before...), ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStatePersist: true},
		{Config: pluralSDKConfig(pluralSDKAppID, after...), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{pluralSDKPlanHook{run: func() { fixture.seed("concurrent", pluralSDKKey("C", false)) }}}}, ExpectError: regexp.MustCompile(`SDK Authentication Keys changed since planning`)},
		{PreConfig: func() { fixture.expectMethods() }, Config: pluralSDKConfig(pluralSDKAppID, after...), ConfigStateChecks: fixture.state(after, func(_ []brazeclient.SDKAuthenticationKey) {
			fixture.expectMethods(http.MethodPut, http.MethodDelete)
			require.Len(t, fixture.requests(), 2)
			assert.Equal(t, "B", fixture.requests()[0].body["key_id"])
			assert.Equal(t, "concurrent", fixture.requests()[1].body["key_id"])
		})},
	}})
}

func TestAccBrazeSDKAuthenticationKeysConcurrentChangeAfterCreate(t *testing.T) {
	t.Parallel()
	fixture := newPluralSDKFixture(t)
	fixture.seed("A", pluralSDKKey("A", true))

	var inject atomic.Bool
	inject.Store(true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fixture.ServeHTTP(w, req)

		if req.Method == http.MethodPost && inject.CompareAndSwap(true, false) {
			fixture.seed("concurrent", pluralSDKKey("X", false))
		}
	})
	before := []pluralSDKMember{pluralSDKKey("A", true)}
	after := []pluralSDKMember{pluralSDKKey("A", true), pluralSDKKey("B", false), pluralSDKKey("C", false)}
	BrazeProviderMockedResourceTest(t, handler, resource.TestCase{Steps: []resource.TestStep{
		{Config: pluralSDKConfig(pluralSDKAppID, before...), ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStatePersist: true},
		{Config: pluralSDKConfig(pluralSDKAppID, after...), ExpectError: regexp.MustCompile(`Failed to update SDK Authentication Keys`)},
		{PreConfig: func() {
			fixture.expectMethods(http.MethodPost)
			assert.Len(t, fixture.remote(t.Context(), pluralSDKAppID), 3)
			fixture.resetWrites()
		}, Config: pluralSDKConfig(pluralSDKAppID, after...), ConfigStateChecks: fixture.state(after, func(_ []brazeclient.SDKAuthenticationKey) {
			fixture.expectMethods(http.MethodDelete, http.MethodPost)
			require.Len(t, fixture.requests(), 2)
			assert.Equal(t, "concurrent", fixture.requests()[0].body["key_id"])
		})},
	}})
}

func TestAccBrazeSDKAuthenticationKeysImportWithoutPrimary(t *testing.T) {
	t.Parallel()
	fixture := newPluralSDKFixture(t)
	fixture.seed("A", pluralSDKKey("A", false))
	keys := []pluralSDKMember{pluralSDKKey("A", true)}
	config := pluralSDKConfig(pluralSDKAppID, keys...)
	BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{Steps: []resource.TestStep{
		{
			Config: config, ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStatePersist: true,
			ImportStateCheck: func(states []*terraform.InstanceState) error {
				require.Len(t, states, 1)
				assert.Equal(t, "false", states[0].Attributes["keys.0.primary"])
				fixture.expectMethods()

				return nil
			},
		},
		{Config: config, ConfigStateChecks: fixture.state(keys, func(remote []brazeclient.SDKAuthenticationKey) {
			fixture.expectMethods(http.MethodPut)
			assert.Equal(t, "A", remote[0].ID)
		})},
	}})
}

func TestAccBrazeSDKAuthenticationKeysAmbiguousRemoteRetainDestroy(t *testing.T) {
	t.Parallel()
	fixture := newPluralSDKFixture(t)
	fixture.seed("A", pluralSDKKey("A", true))
	fixture.seed("duplicate-A", pluralSDKKey("A", false))
	keys := []pluralSDKMember{pluralSDKKey("A", true)}
	config := pluralSDKConfig(pluralSDKAppID, keys...)
	retained := fixture.remote(t.Context(), pluralSDKAppID)
	BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{
		CheckDestroy: func(_ *terraform.State) error {
			fixture.expectMethods()
			assert.ElementsMatch(t, retained, fixture.remote(t.Context(), pluralSDKAppID))

			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: config, ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					require.Len(t, states, 1)
					assert.Equal(t, "2", states[0].Attributes["keys.#"])

					return nil
				},
			},
			{Config: config, PlanOnly: true, ExpectError: regexp.MustCompile(`(?i)(duplicate|ambiguous)`)},
			{PreConfig: func() { fixture.expectMethods() }, Config: `provider "braze" {}`},
		},
	})
}

func TestAccBrazeSDKAuthenticationKeysMalformedReadPreservesState(t *testing.T) {
	t.Parallel()

	for _, scenario := range []string{"duplicate IDs", "multiple primaries", "HTTP 404"} {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			fixture := newPluralSDKFixture(t)

			var corrupt atomic.Bool

			handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet || !corrupt.Load() {
					fixture.ServeHTTP(w, req)

					return
				}

				w.Header().Set("Content-Type", "application/json")

				if scenario == "HTTP 404" {
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"message":"endpoint unavailable"}`))

					return
				}

				keys := fixture.remote(req.Context(), pluralSDKAppID)
				if scenario == "duplicate IDs" {
					keys = append(keys, keys[0])
				} else {
					for index := range keys {
						keys[index].IsPrimary = true
					}
				}

				assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{"keys": keys}))
			})
			keys := []pluralSDKMember{pluralSDKKey("A", true), pluralSDKKey("B", false)}
			config := pluralSDKConfig(pluralSDKAppID, keys...)

			var retained []brazeclient.SDKAuthenticationKey

			BrazeProviderMockedResourceTest(t, handler, resource.TestCase{Steps: []resource.TestStep{
				{Config: config, ConfigStateChecks: fixture.state(keys, func(remote []brazeclient.SDKAuthenticationKey) { retained = remote })},
				{PreConfig: func() { fixture.resetWrites(); corrupt.Store(true) }, Config: config, PlanOnly: true, ExpectError: regexp.MustCompile(`Failed to read SDK Authentication Keys`)},
				{
					PreConfig: func() { corrupt.Store(false); fixture.expectMethods() }, Config: config,
					ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
					ConfigStateChecks: fixture.state(keys, func(remote []brazeclient.SDKAuthenticationKey) {
						assert.ElementsMatch(t, retained, remote)
						fixture.expectMethods()
					}),
				},
			}})
		})
	}
}

// The harness ImportBlockWithResourceIdentity mode only plans an import. Apply
// the identity import block as ordinary configuration to verify persisted state.
func TestAccBrazeSDKAuthenticationKeysIdentityImport(t *testing.T) {
	t.Parallel()
	fixture := newPluralSDKFixture(t)

	keys := []pluralSDKMember{pluralSDKKey("A", true), pluralSDKKey("B", false)}
	for _, key := range keys {
		fixture.seed(key.description, key)
	}

	fixture.reverse.Store(true)
	imported := fixture.remote(t.Context(), pluralSDKAppID)
	config := pluralSDKConfig(pluralSDKAppID, keys...)
	BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
		Steps: []resource.TestStep{
			{
				Config: config + fmt.Sprintf("\nimport {\n to = braze_sdk_authentication_keys.test\n identity = {app_id = %q}\n}\n", pluralSDKAppID),
				ConfigStateChecks: fixture.state(keys, func(remote []brazeclient.SDKAuthenticationKey) {
					fixture.expectMethods()
					assert.ElementsMatch(t, imported, remote)
				}),
			},
			{
				Config: config, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				ConfigStateChecks: fixture.state(keys, func(remote []brazeclient.SDKAuthenticationKey) {
					fixture.expectMethods()
					assert.ElementsMatch(t, imported, remote)
				}),
			},
		},
	})
}

func TestAccBrazeSDKAuthenticationKeysLostCreateResponseRecovery(t *testing.T) {
	t.Parallel()

	fixture := newPluralSDKFixture(t)

	var loseResponse atomic.Bool
	loseResponse.Store(true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost || !loseResponse.CompareAndSwap(true, false) {
			fixture.ServeHTTP(w, req)

			return
		}

		// The API accepts the POST, but the provider never receives its generated ID.
		recorder := httptest.NewRecorder()
		fixture.ServeHTTP(recorder, req)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"message":"create response lost"}`))
	})
	keys := []pluralSDKMember{pluralSDKKey("A", true)}
	config := pluralSDKConfig(pluralSDKAppID, keys...)

	var created []brazeclient.SDKAuthenticationKey

	BrazeProviderMockedResourceTest(t, handler, resource.TestCase{Steps: []resource.TestStep{
		{Config: config, ExpectError: regexp.MustCompile(`Failed to create SDK Authentication Keys`)},
		{PreConfig: func() {
			fixture.expectMethods(http.MethodPost)
			created = fixture.remote(t.Context(), pluralSDKAppID)
			require.Len(t, created, 1)
		}, Config: `
provider "braze" {}
removed {
 from = braze_sdk_authentication_keys.test
 lifecycle { destroy = false }
}
`},
		{Config: config, ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStatePersist: true},
		{Config: config, ConfigStateChecks: fixture.state(keys, func(remote []brazeclient.SDKAuthenticationKey) {
			fixture.expectMethods(http.MethodPost)
			assert.ElementsMatch(t, created, remote)
		})},
	}})
}

func TestAccBrazeSDKAuthenticationKeysLostMutationResponses(t *testing.T) {
	t.Parallel()

	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			fixture := newPluralSDKFixture(t)

			before := []pluralSDKMember{pluralSDKKey("A", true), pluralSDKKey("B", false)}
			for _, key := range before {
				fixture.seed(key.description, key)
			}

			after := []pluralSDKMember{pluralSDKKey("A", true)}
			if method == http.MethodPut {
				after = []pluralSDKMember{pluralSDKKey("A", false), pluralSDKKey("B", true)}
			}

			var loseResponse atomic.Bool
			loseResponse.Store(true)

			handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != method || !loseResponse.CompareAndSwap(true, false) {
					fixture.ServeHTTP(w, req)

					return
				}

				recorder := httptest.NewRecorder()
				fixture.ServeHTTP(recorder, req)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"message":"mutation response unavailable"}`))
			})

			BrazeProviderMockedResourceTest(t, handler, resource.TestCase{Steps: []resource.TestStep{
				{Config: pluralSDKConfig(pluralSDKAppID, before...), ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStatePersist: true},
				{Config: pluralSDKConfig(pluralSDKAppID, after...), ExpectError: regexp.MustCompile(`Failed to update SDK Authentication Keys`)},
				{
					PreConfig: func() { fixture.expectMethods(method) }, Config: pluralSDKConfig(pluralSDKAppID, after...),
					ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
					ConfigStateChecks: fixture.state(after, func(remote []brazeclient.SDKAuthenticationKey) {
						fixture.expectMethods(method)

						for _, key := range remote {
							assert.Equal(t, key.Description, key.ID)
						}
					}),
				},
			}})
		})
	}
}

func TestAccBrazeSDKAuthenticationKeysForcedReplacementRetainsAll(t *testing.T) {
	t.Parallel()

	fixture := newPluralSDKFixture(t)
	keys := []pluralSDKMember{pluralSDKKey("A", true), pluralSDKKey("B", false)}
	config := pluralSDKConfig(pluralSDKAppID, keys...)

	var retained []brazeclient.SDKAuthenticationKey

	BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{Steps: []resource.TestStep{
		{Config: config, ConfigStateChecks: fixture.state(keys, func(remote []brazeclient.SDKAuthenticationKey) { retained = remote })},
		{PreConfig: fixture.resetWrites, Config: config, Taint: []string{pluralSDKAddress}, ExpectError: regexp.MustCompile(`SDK Authentication Keys require import`)},
		{PreConfig: func() {
			fixture.expectMethods()
			assert.ElementsMatch(t, retained, fixture.remote(t.Context(), pluralSDKAppID))
		}, Config: config, ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStatePersist: true},
		{Config: config, ConfigStateChecks: fixture.state(keys, func(remote []brazeclient.SDKAuthenticationKey) {
			fixture.expectMethods()
			assert.ElementsMatch(t, retained, remote)
		})},
	}})
}

func TestAccBrazeSDKAuthenticationKeysUnknownPrimary(t *testing.T) {
	t.Parallel()

	fixture := newPluralSDKFixture(t)
	keys := []pluralSDKMember{pluralSDKKey("A", true), pluralSDKKey("B", false)}
	config := strings.Replace(pluralSDKConfig(pluralSDKAppID, keys...), "primary = true", "primary = terraform_data.primary.output", 1) + `
resource "terraform_data" "primary" { input = true }
`

	BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{Steps: []resource.TestStep{
		{
			Config: config, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectUnknownValue("terraform_data.primary", tfjsonpath.New("output"))}},
			ConfigStateChecks: fixture.state(keys, func(_ []brazeclient.SDKAuthenticationKey) {
				fixture.expectMethods(http.MethodPost, http.MethodPost)
				require.Len(t, fixture.requests(), 2)
				assert.Equal(t, true, fixture.requests()[0].body["make_primary"])
			}),
		},
		{
			PreConfig: fixture.resetWrites, Config: config, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			ConfigStateChecks: fixture.state(keys, func(_ []brazeclient.SDKAuthenticationKey) { fixture.expectMethods() }),
		},
	}})
}

func TestAccBrazeSDKAuthenticationKeysUnknownPrimaryRejectsAtApply(t *testing.T) {
	t.Parallel()

	fixture := newPluralSDKFixture(t)

	before := []pluralSDKMember{pluralSDKKey("A", true), pluralSDKKey("B", false), pluralSDKKey("C", false)}
	for _, key := range before {
		fixture.seed(key.description, key)
	}

	after := []pluralSDKMember{pluralSDKKey("B", false), pluralSDKKey("C", false), pluralSDKKey("D", true)}
	config := strings.Replace(pluralSDKConfig(pluralSDKAppID, after...), "primary = true", "primary = terraform_data.primary.output", 1) + `
resource "terraform_data" "primary" { input = true }
`

	var reachedApply bool

	BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{Steps: []resource.TestStep{
		{Config: pluralSDKConfig(pluralSDKAppID, before...), ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStatePersist: true},
		{Config: config, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
			plancheck.ExpectUnknownValue("terraform_data.primary", tfjsonpath.New("output")),
			pluralSDKPlanHook{run: func() { reachedApply = true }},
		}}, ExpectError: regexp.MustCompile(`(Cannot reconcile|Failed to update) SDK Authentication Keys`)},
		{
			PreConfig: func() {
				require.True(t, reachedApply, "known validation must defer while final primary is unknown")
				fixture.expectMethods()
			},
			Config: pluralSDKConfig(pluralSDKAppID, before...), ConfigStateChecks: fixture.state(before, func(remote []brazeclient.SDKAuthenticationKey) {
				fixture.expectMethods()

				for _, key := range remote {
					assert.Equal(t, key.Description, key.ID)
				}
			}),
		},
	}})
}
