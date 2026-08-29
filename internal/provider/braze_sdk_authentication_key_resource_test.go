package provider_test

import (
	"regexp"
	"strings"
	"testing"

	brazeclienttesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
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

	server, _ := brazeclienttesting.NewBrazeServer()

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: testSDKAuthenticationKeyConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("braze_sdk_authentication_key.test", "id"),
					resource.TestCheckResourceAttr(
						"braze_sdk_authentication_key.test",
						"app_id",
						"01234567-89ab-cdef-0123-456789abcdef",
					),
					resource.TestCheckResourceAttr(
						"braze_sdk_authentication_key.test",
						"rsa_public_key",
						"-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAknOke8rjbXy2LD6DVEGu\nw8V6k+VsRINnGROwXFlrIXVUQ1NY4Dd6vb+0kydgwrH9/zB5nJgRuFkXUH1Iigbl\nCI/Bo9m7JVMqICRpvuKzCCzTn3qzgPSE+7TDwQJfLomo12DDxqcykIR11Y0Nx6mJ\nnOZnCEDgppNtZpdnNOzwE8WGKyNd/JI613mekBByrkmc3boGzAESxoBLAMwQIRgp\nk6+XJ5i/dPWUbk33Lt8QjFH+aZ+0hLKx0IcPGYKFsF87ZQ7b8dpARu/D5i/VhV5n\n7Q7wvzZwt9NMQ8SLzSXGrE7H3wf8/ag7TySmMsANLYIMCsopTXcHdaqJe3QRyPUH\nLQIDAQAB\n-----END PUBLIC KEY-----\n",
					),
					resource.TestCheckResourceAttr(
						"braze_sdk_authentication_key.test",
						"description",
						"Terraform-managed SDK Authentication key",
					),
					resource.TestCheckResourceAttrSet("braze_sdk_authentication_key.test", "primary"),
				),
			},
			{
				Config: testSDKAuthenticationKeyConfig,
			},
		},
	})
}

func TestAccBrazeSDKAuthenticationKeyImport(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testSDKAuthenticationKeyConfig,
			},
			{
				Config:          testSDKAuthenticationKeyConfig,
				ResourceName:    "braze_sdk_authentication_key.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func TestAccBrazeSDKAuthenticationKeyCompositeImport(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: testSDKAuthenticationKeyConfig,
			},
			{
				Config:              testSDKAuthenticationKeyConfig,
				ResourceName:        "braze_sdk_authentication_key.test",
				ImportState:         true,
				ImportStateIdPrefix: "01234567-89ab-cdef-0123-456789abcdef/",
				ImportStateVerify:   true,
			},
		},
	})
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
