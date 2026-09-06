package provider_test

import (
	"fmt"
	"testing"

	brazeclienttesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck/queryfilter"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
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
				QueryResultChecks: []querycheck.QueryResultCheck{querycheck.ExpectLength("braze_email_template.test", 1), querycheck.ExpectResourceKnownValues("braze_email_template.test", nil, []querycheck.KnownValueCheck{
					{Path: tfjsonpath.New("id"), KnownValue: knownvalue.StringExact("email-template-id")},
					{Path: tfjsonpath.New("template_name"), KnownValue: knownvalue.StringExact("test-email-template")},
					{Path: tfjsonpath.New("subject"), KnownValue: knownvalue.StringExact("Welcome")},
					{Path: tfjsonpath.New("body"), KnownValue: knownvalue.StringExact("<p>Hello</p>")},
					{Path: tfjsonpath.New("plaintext_body"), KnownValue: knownvalue.StringExact("Hello")},
					{Path: tfjsonpath.New("preheader"), KnownValue: knownvalue.StringExact("Preview text")},
					{Path: tfjsonpath.New("tags"), KnownValue: knownvalue.ListSizeExact(1)},
					{Path: tfjsonpath.New("tags").AtSliceIndex(0), KnownValue: knownvalue.StringExact("tag1")},
					{Path: tfjsonpath.New("should_inline_css"), KnownValue: knownvalue.Bool(true)},
				})},
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
