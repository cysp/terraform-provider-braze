package provider_test

import (
	"maps"
	"regexp"
	"testing"

	brazeclienttesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccBrazeEmailTemplate(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()

	configVariables1 := config.Variables{
		"template_name": config.StringVariable("test-email-template"),
		"subject":       config.StringVariable("Test Subject"),
		"body":          config.StringVariable("<h1>Test Body</h1>"),
	}

	configVariables2 := config.Variables{
		"template_name": config.StringVariable("test-email-template"),
		"subject":       config.StringVariable("Test Subject"),
		"body":          config.StringVariable("<h1>Test Body</h1>"),
		"tags":          config.ListVariable(config.StringVariable("tag1"), config.StringVariable("tag2")),
	}

	configVariables3 := config.Variables{
		"template_name": config.StringVariable("test-email-template"),
		"subject":       config.StringVariable("Test Subject"),
		"body":          config.StringVariable("<h1>Test Body</h1>"),
		"tags":          config.ListVariable(),
	}

	configVariables4 := config.Variables{
		"template_name": config.StringVariable("test-email-template"),
		"subject":       config.StringVariable("Updated Subject"),
		"body":          config.StringVariable("<h1>Updated Body</h1>"),
	}

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ConfigDirectory: config.TestNameDirectory(),
				ConfigVariables: configVariables1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("braze_email_template.test", "template_name", "test-email-template"),
					resource.TestCheckResourceAttr("braze_email_template.test", "subject", "Test Subject"),
					resource.TestCheckResourceAttr("braze_email_template.test", "body", "<h1>Test Body</h1>"),
					resource.TestCheckNoResourceAttr("braze_email_template.test", "description"),
					resource.TestCheckNoResourceAttr("braze_email_template.test", "preheader"),
					resource.TestCheckNoResourceAttr("braze_email_template.test", "plaintext_body"),
					resource.TestCheckNoResourceAttr("braze_email_template.test", "should_inline_css"),
					resource.TestCheckNoResourceAttr("braze_email_template.test", "tags"),
				),
			},
			{
				ConfigDirectory: config.TestNameDirectory(),
				ImportState:     true,
				ResourceName:    "braze_email_template.test",
			},
			{
				ConfigDirectory: config.TestNameDirectory(),
				ConfigVariables: configVariables1,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("braze_email_template.test", plancheck.ResourceActionNoop),
					},
				},
			},
			{
				ConfigDirectory: config.TestNameDirectory(),
				ConfigVariables: configVariables2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("braze_email_template.test", "template_name", "test-email-template"),
					resource.TestCheckResourceAttr("braze_email_template.test", "subject", "Test Subject"),
					resource.TestCheckResourceAttr("braze_email_template.test", "body", "<h1>Test Body</h1>"),
					resource.TestCheckResourceAttr("braze_email_template.test", "tags.#", "2"),
					resource.TestCheckResourceAttr("braze_email_template.test", "tags.0", "tag1"),
					resource.TestCheckResourceAttr("braze_email_template.test", "tags.1", "tag2"),
				),
			},
			{
				ConfigDirectory: config.TestNameDirectory(),
				ConfigVariables: configVariables3,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("braze_email_template.test", "template_name", "test-email-template"),
					resource.TestCheckResourceAttr("braze_email_template.test", "subject", "Test Subject"),
					resource.TestCheckResourceAttr("braze_email_template.test", "body", "<h1>Test Body</h1>"),
					resource.TestCheckResourceAttr("braze_email_template.test", "tags.#", "0"),
				),
			},
			{
				ConfigDirectory: config.TestNameDirectory(),
				ConfigVariables: configVariables4,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("braze_email_template.test", "template_name", "test-email-template"),
					resource.TestCheckResourceAttr("braze_email_template.test", "subject", "Updated Subject"),
					resource.TestCheckResourceAttr("braze_email_template.test", "body", "<h1>Updated Body</h1>"),
					resource.TestCheckNoResourceAttr("braze_email_template.test", "tags"),
				),
			},
			{
				ConfigDirectory: config.TestNameDirectory(),
				Destroy:         true,
				ResourceName:    "braze_email_template.test",
			},
		},
	})
}

func TestAccBrazeEmailTemplateCreateTemplateNameEmpty(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()

	configVariables := config.Variables{
		"template_name": config.StringVariable(""),
		"subject":       config.StringVariable("Test Subject"),
		"body":          config.StringVariable("<h1>Test Body</h1>"),
	}

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ConfigDirectory: config.TestNameDirectory(),
				ConfigVariables: configVariables,
				ExpectError:     regexp.MustCompile("Failed to create Email Template"),
			},
		},
	})
}

func TestAccBrazeEmailTemplateUpdateTemplateNameEmpty(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()

	configVariables1 := config.Variables{
		"template_name": config.StringVariable("initial-name"),
		"subject":       config.StringVariable("Test Subject"),
		"body":          config.StringVariable("<h1>Test Body</h1>"),
	}

	configVariables2 := maps.Clone(configVariables1)
	configVariables2["template_name"] = config.StringVariable("")

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ConfigDirectory: config.TestNameDirectory(),
				ConfigVariables: configVariables1,
			},
			{
				ConfigDirectory: config.TestNameDirectory(),
				ConfigVariables: configVariables2,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("braze_email_template.test", plancheck.ResourceActionUpdate),
					},
				},
				ExpectError: regexp.MustCompile("Failed to update Email Template"),
			},
		},
	})
}
