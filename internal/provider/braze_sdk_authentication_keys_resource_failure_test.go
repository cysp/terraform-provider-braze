package provider_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fail one response after its API operation has taken effect. GETs before the
// first mutation are preflight reads, not part of the mutation sequence.
type pluralSDKBoundaryFailure struct {
	fixture *pluralSDKFixture
	failAt  int64
	active  atomic.Bool
	started atomic.Bool
	calls   atomic.Int64
	failed  atomic.Bool
}

func (failure *pluralSDKBoundaryFailure) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !failure.active.Load() || req.Method == http.MethodGet && !failure.started.Load() {
		failure.fixture.ServeHTTP(w, req)

		return
	}

	failure.started.Store(true)

	boundary := failure.calls.Add(1) - 1
	if boundary != failure.failAt {
		failure.fixture.ServeHTTP(w, req)

		return
	}

	recorder := httptest.NewRecorder()
	failure.fixture.ServeHTTP(recorder, req)
	assert.Equal(failure.fixture.t, http.StatusOK, recorder.Code, "the injected failure must follow a successful fixture operation")
	failure.failed.Store(true)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"message":"mutation boundary response unavailable"}`))
}

type pluralSDKFailurePlanCheck struct {
	run func(*tfjson.State)
}

func (check pluralSDKFailurePlanCheck) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, _ *plancheck.CheckPlanResponse) {
	check.run(req.Plan.PriorState)
}

// These expectations come from the accepted-response boundary, not from the
// implementation's partial-state model. IDs accepted by an earlier POST must
// remain; a POST whose response was lost cannot contribute an unconfirmed ID.
func pluralSDKAssertFailureState(t *testing.T, state *tfjson.State, ids map[string]string, names []string, primary string, unverified bool) {
	t.Helper()
	require.NotNil(t, state)
	require.NotNil(t, state.Values)
	require.NotNil(t, state.Values.RootModule)

	var attributes map[string]any

	for _, item := range state.Values.RootModule.Resources {
		if item.Address == pluralSDKAddress {
			attributes = item.AttributeValues
		}
	}

	require.NotNil(t, attributes)
	assert.Equal(t, pluralSDKAppID, attributes["app_id"])

	expected := make([]any, 0, len(names))
	for _, name := range names {
		key := pluralSDKKey(name, name == primary)

		var primaryValue any = key.primary
		if unverified {
			primaryValue = nil
		}

		require.NotEmpty(t, ids[name], "missing independent ID for %s", name)
		expected = append(expected, map[string]any{"id": ids[name], "rsa_public_key": key.material, "description": key.description, "primary": primaryValue})
	}

	assert.ElementsMatch(t, expected, attributes["keys"])
}

func pluralSDKFailureIDs(keys []brazeclient.SDKAuthenticationKey) map[string]string {
	ids := make(map[string]string, len(keys))
	for _, key := range keys {
		ids[key.Description] = key.ID
	}

	return ids
}

func pluralSDKFailureMembers(keys []brazeclient.SDKAuthenticationKey) ([]string, string) {
	names := make([]string, 0, len(keys))

	var primary string

	for _, key := range keys {
		names = append(names, key.Description)
		if key.IsPrimary {
			primary = key.Description
		}
	}

	return names, primary
}

type pluralSDKBoundaryCase struct {
	name          string
	expectedNames []string
	primary       string
	unverified    bool
}

func TestAccBrazeSDKAuthenticationKeysCapacityFailureBoundaries(t *testing.T) {
	t.Parallel()

	before := []pluralSDKMember{pluralSDKKey("A", true), pluralSDKKey("B", false), pluralSDKKey("C", false)}
	desired := []pluralSDKMember{pluralSDKKey("B", false), pluralSDKKey("D", true), pluralSDKKey("E", false)}
	// Selecting B in the inspection-only plan keeps a diff even when the last
	// operation succeeded remotely. That plan is never applied.
	inspection := []pluralSDKMember{pluralSDKKey("B", true), pluralSDKKey("D", false), pluralSDKKey("E", false)}
	calls := []string{http.MethodDelete, http.MethodPost, http.MethodGet, http.MethodDelete, http.MethodPost, http.MethodGet}

	cases := []pluralSDKBoundaryCase{
		{name: "delete obsolete C", expectedNames: []string{"A", "B", "C"}, primary: "A"},
		{name: "create primary D", expectedNames: []string{"A", "B"}, unverified: true},
		{name: "verify primary D", expectedNames: []string{"A", "B", "D"}, unverified: true},
		{name: "delete former primary A", expectedNames: []string{"A", "B", "D"}, primary: "D"},
		{name: "create secondary E", expectedNames: []string{"B", "D"}, primary: "D"},
		{name: "verify secondary E", expectedNames: []string{"B", "D", "E"}, primary: "D"},
	}
	for boundary, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			pluralSDKRunUpdateBoundary(t, before, desired, inspection, calls, int64(boundary), scenario)
		})
	}
}

func TestAccBrazeSDKAuthenticationKeysPromotionFailureBoundaries(t *testing.T) {
	t.Parallel()

	before := []pluralSDKMember{pluralSDKKey("A", true), pluralSDKKey("B", false), pluralSDKKey("C", false)}
	desired := []pluralSDKMember{pluralSDKKey("B", true), pluralSDKKey("C", false), pluralSDKKey("D", false)}
	inspection := []pluralSDKMember{pluralSDKKey("B", false), pluralSDKKey("C", true), pluralSDKKey("D", false)}
	calls := []string{http.MethodPut, http.MethodDelete, http.MethodPost, http.MethodGet}

	cases := []pluralSDKBoundaryCase{
		{name: "promote B", expectedNames: []string{"A", "B", "C"}, unverified: true},
		{name: "delete obsolete A", expectedNames: []string{"A", "B", "C"}, primary: "B"},
		{name: "create secondary D", expectedNames: []string{"B", "C"}, primary: "B"},
		{name: "verify secondary D", expectedNames: []string{"B", "C", "D"}, primary: "B"},
	}
	for boundary, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			pluralSDKRunUpdateBoundary(t, before, desired, inspection, calls, int64(boundary), scenario)
		})
	}
}

func pluralSDKRunUpdateBoundary(t *testing.T, before, desired, inspection []pluralSDKMember, calls []string, boundary int64, scenario pluralSDKBoundaryCase) {
	t.Helper()

	fixture := newPluralSDKFixture(t)
	for _, key := range before {
		fixture.seed(key.description, key)
	}

	ids := pluralSDKFailureIDs(fixture.remote(t.Context(), pluralSDKAppID))
	failure := &pluralSDKBoundaryFailure{fixture: fixture, failAt: boundary}

	var stopped []brazeclient.SDKAuthenticationKey

	var inspected, refreshed bool

	expectedMethods := make([]string, 0, len(calls))
	for _, method := range calls[:boundary+1] {
		if method != http.MethodGet {
			expectedMethods = append(expectedMethods, method)
		}
	}

	BrazeProviderMockedResourceTest(t, failure, resource.TestCase{Steps: []resource.TestStep{
		{Config: pluralSDKConfig(pluralSDKAppID, before...), ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStatePersist: true},
		{PreConfig: func() { failure.active.Store(true) }, Config: pluralSDKConfig(pluralSDKAppID, desired...), ExpectError: regexp.MustCompile(`Failed to update SDK Authentication Keys`)},
		{
			PreConfig: func() {
				require.True(t, failure.failed.Load())
				assert.Equal(t, boundary+1, failure.calls.Load(), "executor continued after failed boundary")
				fixture.expectMethods(expectedMethods...)
				failure.active.Store(false)

				stopped = fixture.remote(t.Context(), pluralSDKAppID)
				for _, key := range stopped {
					ids[key.Description] = key.ID
				}
			}, Config: pluralSDKConfig(pluralSDKAppID, inspection...), PlanOnly: true, ExpectNonEmptyPlan: true,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PostApplyPreRefresh: []plancheck.PlanCheck{pluralSDKFailurePlanCheck{run: func(state *tfjson.State) {
					inspected = true

					pluralSDKAssertFailureState(t, state, ids, scenario.expectedNames, scenario.primary, scenario.unverified)
				}}},
				PostApplyPostRefresh: []plancheck.PlanCheck{pluralSDKFailurePlanCheck{run: func(state *tfjson.State) {
					refreshed = true
					names, primary := pluralSDKFailureMembers(stopped)
					pluralSDKAssertFailureState(t, state, ids, names, primary, false)
				}}},
			},
		},
		// Explicitly relinquish/import after the remote outcome has been inspected;
		// this covers ambiguous POST outcomes without treating retry as recovery.
		{PreConfig: func() {
			require.True(t, inspected)
			require.True(t, refreshed)
			fixture.expectMethods(expectedMethods...)
		}, Config: `
provider "braze" {}
removed {
 from = braze_sdk_authentication_keys.test
 lifecycle { destroy = false }
}
`},
		{Config: pluralSDKConfig(pluralSDKAppID, desired...), ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStatePersist: true},
		{Config: pluralSDKConfig(pluralSDKAppID, desired...), ConfigStateChecks: fixture.state(desired, func(remote []brazeclient.SDKAuthenticationKey) {
			for _, key := range remote {
				if confirmedID := ids[key.Description]; confirmedID != "" {
					assert.Equal(t, confirmedID, key.ID)
				}
			}
			// Every actual creation in the full intended sequence occurs once, even
			// when its response was lost. No rollback deletes a newly generated ID.
			created := make(map[string]int)

			for _, request := range fixture.requests() {
				if request.method == http.MethodPost {
					description, ok := request.body["description"].(string)
					require.True(t, ok)

					created[description]++
				}

				if request.method == http.MethodDelete {
					assert.Contains(t, []string{"A", "C"}, request.body["key_id"])
				}
			}

			for _, key := range desired {
				if key.description == "D" || key.description == "E" {
					assert.Equal(t, 1, created[key.description])
				}
			}
		})},
	}})
}

func TestAccBrazeSDKAuthenticationKeysInitialCreateFailureBoundaries(t *testing.T) {
	t.Parallel()

	cases := []pluralSDKBoundaryCase{
		{name: "create primary A", unverified: true},
		{name: "verify primary A", expectedNames: []string{"A"}, unverified: true},
		{name: "create secondary B", expectedNames: []string{"A"}, primary: "A"},
		{name: "verify secondary B", expectedNames: []string{"A", "B"}, primary: "A"},
		{name: "create secondary C", expectedNames: []string{"A", "B"}, primary: "A"},
		{name: "verify secondary C", expectedNames: []string{"A", "B", "C"}, primary: "A"},
	}
	for boundary, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			fixture := newPluralSDKFixture(t)
			failure := &pluralSDKBoundaryFailure{fixture: fixture, failAt: int64(boundary)}
			failure.active.Store(true)

			desired := []pluralSDKMember{pluralSDKKey("A", true), pluralSDKKey("B", false), pluralSDKKey("C", false)}
			config := pluralSDKConfig(pluralSDKAppID, desired...)
			ids := make(map[string]string)

			var stopped []brazeclient.SDKAuthenticationKey

			var inspected, refreshed bool

			expectedMethods := make([]string, boundary/2+1)
			for index := range expectedMethods {
				expectedMethods[index] = http.MethodPost
			}

			BrazeProviderMockedResourceTest(t, failure, resource.TestCase{Steps: []resource.TestStep{
				{Config: config, ExpectError: regexp.MustCompile(`Failed to create SDK Authentication Keys`)},
				{
					PreConfig: func() {
						require.True(t, failure.failed.Load())
						assert.EqualValues(t, boundary+1, failure.calls.Load())
						fixture.expectMethods(expectedMethods...)
						failure.active.Store(false)

						stopped = fixture.remote(t.Context(), pluralSDKAppID)
						ids = pluralSDKFailureIDs(stopped)
					}, Config: config, PlanOnly: true, ExpectNonEmptyPlan: true,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPreRefresh: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(pluralSDKAddress, plancheck.ResourceActionDestroyBeforeCreate),
							pluralSDKFailurePlanCheck{run: func(state *tfjson.State) {
								inspected = true

								pluralSDKAssertFailureState(t, state, ids, scenario.expectedNames, scenario.primary, scenario.unverified)
							}},
						},
						PostApplyPostRefresh: []plancheck.PlanCheck{pluralSDKFailurePlanCheck{run: func(state *tfjson.State) {
							refreshed = true
							names, primary := pluralSDKFailureMembers(stopped)
							pluralSDKAssertFailureState(t, state, ids, names, primary, false)
						}}},
					},
				},
				{PreConfig: func() {
					require.True(t, inspected)
					require.True(t, refreshed)
					fixture.expectMethods(expectedMethods...)
				}, Config: `
provider "braze" {}
removed {
 from = braze_sdk_authentication_keys.test
 lifecycle { destroy = false }
}
`},
				{Config: config, ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStatePersist: true},
				{Config: config, ConfigStateChecks: fixture.state(desired, func(remote []brazeclient.SDKAuthenticationKey) {
					fixture.expectMethods(http.MethodPost, http.MethodPost, http.MethodPost)

					created := make(map[string]int)

					for _, request := range fixture.requests() {
						description, ok := request.body["description"].(string)
						require.True(t, ok)

						created[description]++
					}

					assert.Equal(t, map[string]int{"A": 1, "B": 1, "C": 1}, created)

					for _, key := range remote {
						if confirmedID := ids[key.Description]; confirmedID != "" {
							assert.Equal(t, confirmedID, key.ID)
						}
					}
				})},
			}})
		})
	}
}

func TestAccBrazeSDKAuthenticationKeysImportEmptyCollection(t *testing.T) {
	t.Parallel()

	fixture := newPluralSDKFixture(t)
	desired := []pluralSDKMember{pluralSDKKey("A", true)}
	config := pluralSDKConfig(pluralSDKAppID, desired...)

	var checkedImport bool

	BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{Steps: []resource.TestStep{
		{
			Config: config, ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStatePersist: true,
			ImportStateCheck: func(states []*terraform.InstanceState) error {
				checkedImport = true

				require.Len(t, states, 1, "empty collection must remain an imported management resource")
				assert.Equal(t, pluralSDKAppID, states[0].Attributes["app_id"])
				assert.Equal(t, "0", states[0].Attributes["keys.#"])
				fixture.expectMethods()

				return nil
			},
		},
		{PreConfig: func() { require.True(t, checkedImport) }, Config: config, ConfigStateChecks: fixture.state(desired, func(_ []brazeclient.SDKAuthenticationKey) { fixture.expectMethods(http.MethodPost) })},
	}})
}

func TestAccBrazeSDKAuthenticationKeysImportEmptyDescription(t *testing.T) {
	t.Parallel()

	fixture := newPluralSDKFixture(t)
	// This is a deliberately permissive response fixture, not a claim that the
	// live dashboard permits this object. Import must preserve what was returned.
	observed := pluralSDKKey("A", true)
	observed.description = ""
	fixture.seed("dashboard-empty-description", observed)

	desired := []pluralSDKMember{pluralSDKKey("A", true)}
	config := pluralSDKConfig(pluralSDKAppID, desired...)

	var checkedImport bool

	BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{Steps: []resource.TestStep{
		{
			Config: config, ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStatePersist: true,
			ImportStateCheck: func(states []*terraform.InstanceState) error {
				checkedImport = true

				require.Len(t, states, 1)
				assert.Equal(t, "1", states[0].Attributes["keys.#"])

				var foundDescription, foundID bool

				for path, value := range states[0].Attributes {
					if strings.HasSuffix(path, ".description") {
						foundDescription = true

						assert.Empty(t, value)
					}

					if strings.HasSuffix(path, ".id") {
						foundID = true

						assert.Equal(t, "dashboard-empty-description", value)
					}
				}

				require.True(t, foundDescription, "import must retain the empty description verbatim")
				require.True(t, foundID)
				fixture.expectMethods()

				return nil
			},
		},
		{PreConfig: func() { require.True(t, checkedImport) }, Config: config, ConfigStateChecks: fixture.state(desired, func(remote []brazeclient.SDKAuthenticationKey) {
			fixture.expectMethods(http.MethodPost, http.MethodDelete)
			require.Len(t, remote, 1)
			assert.NotEqual(t, "dashboard-empty-description", remote[0].ID)

			requests := fixture.requests()
			require.Len(t, requests, 2)
			assert.Equal(t, "A", requests[0].body["description"])
			assert.Equal(t, true, requests[0].body["make_primary"])
			assert.Equal(t, "dashboard-empty-description", requests[1].body["key_id"])
		})},
	}})
}
