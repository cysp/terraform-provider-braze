package provider_test

import (
	"fmt"
	"testing"

	brazeclienttesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck/queryfilter"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccBrazeEmailTemplateList(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()

	shouldInlineCSS := true
	server.SetEmailTemplate("email-template-id", "test-email-template", "Welcome", "<p>Hello</p>", "Hello", "Preview text", []string{"tag1"}, &shouldInlineCSS)

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Query: true,
				Config: `
				provider "braze" {}

				list "braze_email_template" "test" {
					provider = braze
					config {
						modified_after = "1970-01-01T00:00:00Z"
						modified_before = "9999-01-01T00:00:00Z"
					}
					include_resource = true
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("braze_email_template.test", "id", "email-template-id"),
					resource.TestCheckResourceAttr("braze_email_template.test", "template_name", "test-email-template"),
					resource.TestCheckResourceAttr("braze_email_template.test", "subject", "Welcome"),
					resource.TestCheckResourceAttr("braze_email_template.test", "body", "<p>Hello</p>"),
					resource.TestCheckResourceAttr("braze_email_template.test", "plaintext_body", "Hello"),
					resource.TestCheckResourceAttr("braze_email_template.test", "preheader", "Preview text"),
					resource.TestCheckResourceAttr("braze_email_template.test", "tags.#", "1"),
					resource.TestCheckResourceAttr("braze_email_template.test", "tags.0", "tag1"),
					resource.TestCheckResourceAttr("braze_email_template.test", "should_inline_css", "true"),
				),
			},
			{
				Query: true,
				Config: `
				provider "braze" {}

				list "braze_email_template" "test" {
					provider = braze

					limit = 1
				}
				`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("braze_email_template.test", 1),
					querycheck.ExpectIdentity("braze_email_template.test", map[string]knownvalue.Check{
						"id": knownvalue.StringExact("email-template-id"),
					}),
					querycheck.ExpectResourceDisplayName(
						"braze_email_template.test",
						queryfilter.ByResourceIdentity(map[string]knownvalue.Check{
							"id": knownvalue.StringExact("email-template-id"),
						}),
						knownvalue.StringExact("test-email-template"),
					),
				},
			},
			{
				Query: true,
				Config: `
				provider "braze" {}

				list "braze_email_template" "test" {
					provider = braze

					limit = 0
				}
				`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("braze_email_template.test", 0),
				},
			},
		},
	})
}

func TestAccBrazeEmailTemplateListPagination(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()

	for i := range 101 {
		id := fmt.Sprintf("email-template-pagination-%03d", i)

		server.SetEmailTemplate(id, id, "subject", "body", "", "", []string{}, nil)
	}

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Query: true,
				Config: `
				provider "braze" {}

				list "braze_email_template" "test" {
					provider = braze

					limit = 101
				}
				`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("braze_email_template.test", 101),
				},
			},
		},
	})
}
