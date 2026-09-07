package provider_test

import (
	"net/http"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	brazeclienttesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSDKAuthenticationKeyConfig = `
provider "braze" {}

resource "braze_sdk_authentication_key" "test" {
  app_id         = "01234567-89ab-cdef-0123-456789abcdef"
  rsa_public_key = <<-PEM
    -----BEGIN PUBLIC KEY-----
    MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAknOke8rjbXy2LD6DVEGu
    w8V6k+VsRINnGROwXFlrIXVUQ1NY4Dd6vb+0kydgwrH9/zB5nJgRuFkXUH1Iigbl
    CI/Bo9m7JVMqICRpvuKzCCzTn3qzgPSE+7TDwQJfLomo12DDxqcykIR11Y0Nx6mJ
    nOZnCEDgppNtZpdnNOzwE8WGKyNd/JI613mekBByrkmc3boGzAESxoBLAMwQIRgp
    k6+XJ5i/dPWUbk33Lt8QjFH+aZ+0hLKx0IcPGYKFsF87ZQ7b8dpARu/D5i/VhV5n
    7Q7wvzZwt9NMQ8SLzSXGrE7H3wf8/ag7TySmMsANLYIMCsopTXcHdaqJe3QRyPUH
    LQIDAQAB
    -----END PUBLIC KEY-----
  PEM
  description    = "Terraform-managed SDK Authentication key"
}
`

func TestAccBrazeSDKAuthenticationKey(t *testing.T) {
	t.Parallel()

	const (
		appID   = "01234567-89ab-cdef-0123-456789abcdef"
		address = "braze_sdk_authentication_key.test"
	)

	server, err := brazeclienttesting.NewBrazeServer()
	require.NoError(t, err)

	external := brazeclient.SDKAuthenticationKey{ID: "external", RsaPublicKey: "external public key", Description: "External key", IsPrimary: true}
	server.SetSDKAuthenticationKey(appID, external)

	var creates, promotions, deletions atomic.Int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodPost:
			creates.Add(1)
		case http.MethodPut:
			promotions.Add(1)
		case http.MethodDelete:
			deletions.Add(1)
		}

		server.ServeHTTP(w, req)
	})
	id := statecheck.CompareValue(compare.ValuesSame())
	stateChecks := func(primary bool) []statecheck.StateCheck {
		return []statecheck.StateCheck{
			statecheck.ExpectKnownValue(address, tfjsonpath.New("id"), knownvalue.NotNull()),
			id.AddStateValue(address, tfjsonpath.New("id")),
			statecheck.ExpectKnownValue(address, tfjsonpath.New("app_id"), knownvalue.StringExact(appID)),
			statecheck.ExpectKnownValue(address, tfjsonpath.New("primary"), knownvalue.Bool(primary)),
		}
	}

	var managedID string

	assertRemote := func(primary bool, expectedPromotions int32) {
		t.Helper()

		response, err := server.Handler().ListSDKAuthenticationKeys(t.Context(), brazeclient.ListSDKAuthenticationKeysParams{AppID: appID})
		require.NoError(t, err)
		require.Len(t, response.Response.Keys, 2)

		for _, key := range response.Response.Keys {
			if key.ID == external.ID {
				assert.Equal(t, !primary, key.IsPrimary)
				assert.Equal(t, external.RsaPublicKey, key.RsaPublicKey)

				continue
			}

			if managedID == "" {
				managedID = key.ID
			}

			assert.Equal(t, managedID, key.ID, "promotion and import must preserve the existing key")
			assert.Equal(t, primary, key.IsPrimary)
			assert.Equal(t, "Terraform-managed SDK Authentication key", key.Description)
		}

		assert.EqualValues(t, 1, creates.Load())
		assert.Equal(t, expectedPromotions, promotions.Load())
		assert.Zero(t, deletions.Load())
	}
	promoteExternal := func() {
		_, err := server.Handler().SetPrimarySDKAuthenticationKey(t.Context(), &brazeclient.SetPrimarySDKAuthenticationKeyRequest{AppID: appID, KeyID: external.ID})
		require.NoError(t, err)
	}
	primaryConfig := testSDKAuthenticationKeyPrimaryConfig("Terraform-managed SDK Authentication key")
	update := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)}}

	BrazeProviderMockedResourceTest(t, handler, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
		Steps: []resource.TestStep{
			{Config: testSDKAuthenticationKeyConfig, ConfigStateChecks: stateChecks(false), PostApplyFunc: func() { assertRemote(false, 0) }},
			{Config: testSDKAuthenticationKeyConfig, ResourceName: address, ImportState: true, ImportStateIdPrefix: appID + "/", ImportStateVerify: true},
			{Config: testSDKAuthenticationKeyConfig, ResourceName: address, ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity},
			{Config: primaryConfig, ConfigPlanChecks: update, ConfigStateChecks: stateChecks(true), PostApplyFunc: func() { assertRemote(true, 1) }},
			{PreConfig: promoteExternal, Config: primaryConfig, ConfigPlanChecks: update, ConfigStateChecks: stateChecks(true), PostApplyFunc: func() { assertRemote(true, 2) }},
			// Relinquish the primary claim before destroy; the managed secondary
			// must be deleted while the external primary remains untouched.
			{
				PreConfig: promoteExternal, Config: testSDKAuthenticationKeyConfig,
				ConfigPlanChecks:  resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				ConfigStateChecks: stateChecks(false), PostApplyFunc: func() { assertRemote(false, 2) },
			},
		},
	})

	response, err := server.Handler().ListSDKAuthenticationKeys(t.Context(), brazeclient.ListSDKAuthenticationKeysParams{AppID: appID})
	require.NoError(t, err)
	assert.Equal(t, []brazeclient.SDKAuthenticationKey{external}, response.Response.Keys)
	assert.EqualValues(t, 1, deletions.Load())
}

func TestAccBrazeSDKAuthenticationKeyRejectsPrimaryDestroy(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()
	primaryConfig := testSDKAuthenticationKeyPrimaryConfig("Terraform-managed SDK Authentication key")

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: primaryConfig,
			},
			{
				Config:      `provider "braze" {}`,
				ExpectError: regexp.MustCompile(`Cannot delete currently primary SDK Authentication Key`),
			},
			{
				Config: primaryConfig,
				PostApplyFunc: func() {
					server.ResetSDKAuthenticationKeys("01234567-89ab-cdef-0123-456789abcdef")
				},
			},
			{
				Config: `provider "braze" {}`,
			},
		},
	})
}

func TestAccBrazeSDKAuthenticationKeyRejectsPrimaryReplacement(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()
	primaryConfig := testSDKAuthenticationKeyPrimaryConfig("Terraform-managed SDK Authentication key")

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: primaryConfig,
			},
			{
				Config:      testSDKAuthenticationKeyPrimaryConfig("Replacement SDK Authentication key"),
				ExpectError: regexp.MustCompile(`Cannot replace currently primary SDK Authentication Key`),
			},
			{
				Config: primaryConfig,
				PostApplyFunc: func() {
					server.ResetSDKAuthenticationKeys("01234567-89ab-cdef-0123-456789abcdef")
				},
			},
			{
				Config: `provider "braze" {}`,
			},
		},
	})
}

func TestAccBrazeSDKAuthenticationKeyRotation(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()
	oldPrimary := testSDKAuthenticationKeyNamedConfig(
		"old",
		"Previous SDK Authentication key",
		true,
	)
	overlap := testSDKAuthenticationKeyNamedConfig(
		"old",
		"Previous SDK Authentication key",
		false,
	) + testSDKAuthenticationKeyResourceBlock(
		"next",
		"Replacement SDK Authentication key",
		true,
	)
	nextPrimary := "provider \"braze\" {}\n\n" + testSDKAuthenticationKeyResourceBlock(
		"next",
		"Replacement SDK Authentication key",
		true,
	)

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: oldPrimary,
			},
			{
				Config: overlap,
				Check: resource.TestCheckResourceAttr(
					"braze_sdk_authentication_key.next",
					"primary",
					"true",
				),
			},
			{
				Config: overlap,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"braze_sdk_authentication_key.old",
						"primary",
						"false",
					),
					resource.TestCheckResourceAttr(
						"braze_sdk_authentication_key.next",
						"primary",
						"true",
					),
				),
			},
			{
				Config: nextPrimary,
				Check: func(_ *terraform.State) error {
					response, err := server.Handler().ListSDKAuthenticationKeys(t.Context(), brazeclient.ListSDKAuthenticationKeysParams{AppID: "01234567-89ab-cdef-0123-456789abcdef"})
					require.NoError(t, err)
					require.Len(t, response.Response.Keys, 1)
					assert.Equal(t, "Replacement SDK Authentication key", response.Response.Keys[0].Description)
					assert.True(t, response.Response.Keys[0].IsPrimary)

					return nil
				},
				PostApplyFunc: func() {
					server.ResetSDKAuthenticationKeys("01234567-89ab-cdef-0123-456789abcdef")
				},
			},
			{
				Config: `provider "braze" {}`,
			},
		},
	})
}

func TestAccBrazeSDKAuthenticationKeyValidation(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: `
provider "braze" {}

resource "braze_sdk_authentication_key" "test" {
  app_id         = "01234567-89ab-cdef-0123-456789abcdef"
  rsa_public_key = "public key"
  description    = ""
}
`,
				ExpectError: regexp.MustCompile(`description value must not be empty`),
			},
			{
				Config: `
provider "braze" {}

resource "braze_sdk_authentication_key" "test" {
  app_id         = "01234567-89ab-cdef-0123-456789abcdef"
  rsa_public_key = "public key"
  description    = "   "
}
`,
				ExpectError: regexp.MustCompile(`description value must not be empty`),
			},
			{
				Config: `
provider "braze" {}

resource "braze_sdk_authentication_key" "test" {
  app_id         = "01234567-89ab-cdef-0123-456789abcdef"
  rsa_public_key = "public key"
  description    = "Key"
  primary        = false
}
`,
				ExpectError: regexp.MustCompile(`does not support directly unsetting`),
			},
		},
	})
}

func testSDKAuthenticationKeyPrimaryConfig(description string) string {
	return testSDKAuthenticationKeyNamedConfig("test", description, true)
}

func testSDKAuthenticationKeyNamedConfig(name, description string, claimPrimary bool) string {
	config := strings.Replace(
		testSDKAuthenticationKeyConfig,
		`resource "braze_sdk_authentication_key" "test"`,
		`resource "braze_sdk_authentication_key" "`+name+`"`,
		1,
	)

	config = strings.Replace(
		config,
		`description    = "Terraform-managed SDK Authentication key"`,
		`description    = "`+description+`"`,
		1,
	)
	if claimPrimary {
		config = strings.Replace(config, "\n}\n", "\n  primary        = true\n}\n", 1)
	}

	return config
}

func testSDKAuthenticationKeyResourceBlock(name, description string, claimPrimary bool) string {
	return strings.TrimPrefix(
		testSDKAuthenticationKeyNamedConfig(name, description, claimPrimary),
		"\nprovider \"braze\" {}\n\n",
	)
}

func TestAccBrazeSDKAuthenticationKeyDisappears(t *testing.T) {
	t.Parallel()

	server, err := brazeclienttesting.NewBrazeServer()
	require.NoError(t, err)

	var oldID string

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: testSDKAuthenticationKeyConfig, Check: func(state *terraform.State) error {
				oldID = state.RootModule().Resources["braze_sdk_authentication_key.test"].Primary.ID

				return nil
			}},
			{PreConfig: func() { server.ResetSDKAuthenticationKeys("01234567-89ab-cdef-0123-456789abcdef") }, Config: testSDKAuthenticationKeyConfig, Check: func(state *terraform.State) error {
				assert.NotEqual(t, oldID, state.RootModule().Resources["braze_sdk_authentication_key.test"].Primary.ID)

				return nil
			}},
		},
	})
}

func TestAccBrazeSDKAuthenticationKeyCreateVerificationRecovery(t *testing.T) {
	t.Parallel()

	server, err := brazeclienttesting.NewBrazeServer()
	require.NoError(t, err)

	var (
		verificationUnavailable atomic.Bool
		creates                 atomic.Int32
	)

	verificationUnavailable.Store(true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			creates.Add(1)
		}

		if req.Method == http.MethodGet && verificationUnavailable.Load() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"permission denied"}`))

			return
		}

		server.ServeHTTP(w, req)
	})
	config := testSDKAuthenticationKeyPrimaryConfig("Terraform-managed SDK Authentication key")
	BrazeProviderMockedResourceTest(t, handler, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: config, ExpectError: regexp.MustCompile(`Failed to create SDK Authentication Key`)},
			// A create error taints the key. Once reads recover, replacement still
			// cannot delete it while it is primary. Recovery must adopt the same ID.
			{PreConfig: func() { verificationUnavailable.Store(false) }, Config: config, ExpectError: regexp.MustCompile(`Failed to delete SDK Authentication Key`)},
			{Config: `
provider "braze" {}
removed {
 from = braze_sdk_authentication_key.test
 lifecycle { destroy = false }
}
`},
			{Config: config, ResourceName: "braze_sdk_authentication_key.test", ImportState: true, ImportStatePersist: true, ImportStateIdFunc: func(_ *terraform.State) (string, error) {
				response, err := server.Handler().ListSDKAuthenticationKeys(t.Context(), brazeclient.ListSDKAuthenticationKeysParams{AppID: "01234567-89ab-cdef-0123-456789abcdef"})
				require.NoError(t, err)
				require.Len(t, response.Response.Keys, 1)

				return "01234567-89ab-cdef-0123-456789abcdef/" + response.Response.Keys[0].ID, nil
			}},
			{Config: config, Check: func(_ *terraform.State) error {
				assert.EqualValues(t, 1, creates.Load(), "Recovery must reuse the created key")

				return nil
			}, PostApplyFunc: func() { server.ResetSDKAuthenticationKeys("01234567-89ab-cdef-0123-456789abcdef") }},
			{Config: `provider "braze" {}`},
		},
	})
}

func TestAccBrazeSDKAuthenticationKeyImmutableAttributes(t *testing.T) {
	t.Parallel()

	const appID = "01234567-89ab-cdef-0123-456789abcdef"
	for name, replacement := range map[string]struct{ old, next string }{
		"app_id":         {appID, "fedcba98-7654-3210-fedc-ba9876543210"},
		"description":    {"Terraform-managed SDK Authentication key", "Replacement description"},
		"rsa_public_key": {"knOke8rjbXy2LD6DVEGu", "knOke8rjbXy2LD6DVEGv"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server, err := brazeclienttesting.NewBrazeServer()
			require.NoError(t, err)

			var oldID string

			BrazeProviderMockedResourceTest(t, server, resource.TestCase{
				Steps: []resource.TestStep{
					{Config: testSDKAuthenticationKeyConfig, Check: func(state *terraform.State) error {
						oldID = state.RootModule().Resources["braze_sdk_authentication_key.test"].Primary.ID

						return nil
					}},
					{Config: strings.Replace(testSDKAuthenticationKeyConfig, replacement.old, replacement.next, 1), Check: func(state *terraform.State) error {
						assert.NotEqual(t, oldID, state.RootModule().Resources["braze_sdk_authentication_key.test"].Primary.ID)
						response, err := server.Handler().ListSDKAuthenticationKeys(t.Context(), brazeclient.ListSDKAuthenticationKeysParams{AppID: appID})
						require.NoError(t, err)

						for _, key := range response.Response.Keys {
							assert.NotEqual(t, oldID, key.ID, "Replacement must delete the old remote key")
						}

						return nil
					}},
				},
			})
		})
	}
}
