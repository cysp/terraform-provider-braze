package provider_test

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
)

// Ignoring keys preserves observed state even when it is not a valid desired
// collection. The configuration itself remains valid in every scenario.
func TestAccBrazeSDKAuthenticationKeysIgnoreChanges(t *testing.T) {
	t.Parallel()

	emptyDescription := pluralSDKKey("A", true)
	emptyDescription.description = ""

	cases := []struct {
		name     string
		observed []pluralSDKMember
		methods  []string
		planErr  *regexp.Regexp
	}{
		{name: "empty collection", methods: []string{http.MethodPost}},
		{name: "no primary", observed: []pluralSDKMember{pluralSDKKey("A", false)}, methods: []string{http.MethodPut}},
		{name: "empty description", observed: []pluralSDKMember{emptyDescription}, methods: []string{http.MethodPost, http.MethodDelete}},
		{name: "duplicate immutable pairs", observed: []pluralSDKMember{pluralSDKKey("A", true), pluralSDKKey("A", false)}, planErr: regexp.MustCompile(`(?i)(duplicate|ambiguous)`)},
	}
	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			fixture := newPluralSDKFixture(t)
			for index, key := range scenario.observed {
				fixture.seed(fmt.Sprintf("observed-%d", index), key)
			}

			retained := fixture.remote(t.Context(), pluralSDKAppID)

			var expectedMethods []string

			desired := []pluralSDKMember{pluralSDKKey("A", true)}
			config := pluralSDKConfig(pluralSDKAppID, desired...)
			ignored := strings.Replace(config, " keys = [", " lifecycle { ignore_changes = [keys] }\n keys = [", 1)
			steps := []resource.TestStep{
				{Config: ignored, ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStatePersist: true},
				{
					Config:           ignored,
					ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
					ConfigStateChecks: fixture.state(scenario.observed, func(remote []brazeclient.SDKAuthenticationKey) {
						fixture.expectMethods()
						assert.ElementsMatch(t, retained, remote)
					}),
				},
			}

			// Removing ignore_changes restores reconciliation, including its
			// refusal to match two observed IDs to one immutable pair.
			if scenario.planErr != nil {
				steps = append(steps, resource.TestStep{Config: config, PlanOnly: true, ExpectError: scenario.planErr})
			} else {
				steps = append(steps, resource.TestStep{Config: config, ConfigStateChecks: fixture.state(desired, func(remote []brazeclient.SDKAuthenticationKey) {
					fixture.expectMethods(scenario.methods...)

					expectedMethods = scenario.methods
					if scenario.name == "no primary" {
						assert.Equal(t, retained[0].ID, remote[0].ID, "promotion must preserve the observed key ID")
					}

					retained = remote
				})})
			}

			BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{
				CheckDestroy: func(_ *terraform.State) error {
					fixture.expectMethods(expectedMethods...)
					assert.ElementsMatch(t, retained, fixture.remote(t.Context(), pluralSDKAppID))

					return nil
				},
				Steps: steps,
			})
		})
	}
}

func TestAccBrazeSDKAuthenticationKeysIgnoreChangesCreate(t *testing.T) {
	t.Parallel()

	fixture := newPluralSDKFixture(t)
	desired := []pluralSDKMember{pluralSDKKey("A", true)}
	config := strings.Replace(pluralSDKConfig(pluralSDKAppID, desired...), " keys = [", " lifecycle { ignore_changes = [keys] }\n keys = [", 1)

	BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{Steps: []resource.TestStep{
		{Config: config, ConfigStateChecks: fixture.state(desired, func(_ []brazeclient.SDKAuthenticationKey) {
			fixture.expectMethods(http.MethodPost)
		})},
	}})
}

func TestAccBrazeSDKAuthenticationKeysIgnoreChangesAppReplacement(t *testing.T) {
	t.Parallel()

	const nextAppID = "fedcba98-7654-3210-fedc-ba9876543210"

	emptyDescription := pluralSDKKey("A", true)
	emptyDescription.description = ""

	for name, observed := range map[string][]pluralSDKMember{
		"empty collection":          {},
		"no primary":                {pluralSDKKey("A", false)},
		"empty description":         {emptyDescription},
		"duplicate immutable pairs": {pluralSDKKey("A", true), pluralSDKKey("A", false)},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fixture := newPluralSDKFixture(t)
			for index, key := range observed {
				fixture.seed(fmt.Sprintf("observed-%d", index), key)
			}

			oldKeys := fixture.remote(t.Context(), pluralSDKAppID)
			desired := []pluralSDKMember{pluralSDKKey("next", true)}
			config := strings.Replace(pluralSDKConfig(pluralSDKAppID, desired...), " keys = [", " lifecycle { ignore_changes = [keys] }\n keys = [", 1)

			var newKeys []brazeclient.SDKAuthenticationKey

			BrazeProviderMockedResourceTest(t, fixture, resource.TestCase{
				CheckDestroy: func(_ *terraform.State) error {
					assert.ElementsMatch(t, oldKeys, fixture.remote(t.Context(), pluralSDKAppID))
					assert.ElementsMatch(t, newKeys, fixture.remote(t.Context(), nextAppID))
					fixture.expectMethods()

					return nil
				},
				Steps: []resource.TestStep{
					{Config: config, ResourceName: pluralSDKAddress, ImportState: true, ImportStateId: pluralSDKAppID, ImportStatePersist: true},
					{
						Config:           config,
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
						ConfigStateChecks: fixture.state(observed, func(remote []brazeclient.SDKAuthenticationKey) {
							assert.ElementsMatch(t, oldKeys, remote)
							fixture.expectMethods()
						}),
					},
					{
						Config:           strings.Replace(config, pluralSDKAppID, nextAppID, 1),
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(pluralSDKAddress, plancheck.ResourceActionDestroyBeforeCreate)}},
						ConfigStateChecks: []statecheck.StateCheck{pluralSDKStateCheck{fixture: fixture, appID: nextAppID, keys: desired, after: func(remote []brazeclient.SDKAuthenticationKey) {
							newKeys = remote
							assert.ElementsMatch(t, oldKeys, fixture.remote(t.Context(), pluralSDKAppID))
							fixture.expectMethods(http.MethodPost)

							for _, request := range fixture.requests() {
								assert.Equal(t, nextAppID, request.body["app_id"])
							}

							fixture.resetWrites()
						}}},
					},
				},
			})
		})
	}
}
