package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccBrazeEmailTemplate(t *testing.T) {
	t.Parallel()

	server := newBrazeTestServer(t)

	configVariables1 := config.Variables{
		"email_template_name":    config.StringVariable("test-email-template"),
		"email_template_subject": config.StringVariable("Welcome"),
		"email_template_body":    config.StringVariable("<p>Hello</p>"),
	}

	configVariables2 := config.Variables{
		"email_template_name":              config.StringVariable("test-email-template"),
		"email_template_subject":           config.StringVariable("Welcome"),
		"email_template_body":              config.StringVariable("<p>Hello</p>"),
		"email_template_plaintext_body":    config.StringVariable("Hello"),
		"email_template_preheader":         config.StringVariable("Preview text"),
		"email_template_tags":              config.ListVariable(config.StringVariable("tag1"), config.StringVariable("tag2")),
		"email_template_should_inline_css": config.BoolVariable(true),
	}

	configVariables3 := config.Variables{
		"email_template_name":              config.StringVariable("test-email-template"),
		"email_template_subject":           config.StringVariable("Welcome"),
		"email_template_body":              config.StringVariable("<p>Hello</p>"),
		"email_template_tags":              config.ListVariable(),
		"email_template_should_inline_css": config.BoolVariable(false),
	}

	configVariables4 := config.Variables{
		"email_template_name":    config.StringVariable("test-email-template"),
		"email_template_subject": config.StringVariable("Welcome"),
		"email_template_body":    config.StringVariable("<p>Hello</p>"),
	}

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ConfigDirectory: config.TestNameDirectory(),
				ConfigVariables: configVariables1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("braze_email_template.test", "template_name", "test-email-template"),
					resource.TestCheckResourceAttr("braze_email_template.test", "subject", "Welcome"),
					resource.TestCheckResourceAttr("braze_email_template.test", "body", "<p>Hello</p>"),
					resource.TestCheckNoResourceAttr("braze_email_template.test", "plaintext_body"),
					resource.TestCheckNoResourceAttr("braze_email_template.test", "preheader"),
					resource.TestCheckNoResourceAttr("braze_email_template.test", "tags"),
					resource.TestCheckNoResourceAttr("braze_email_template.test", "should_inline_css"),
				),
			},
			{
				ConfigDirectory:   config.TestNameDirectory(),
				ImportState:       true,
				ImportStateVerify: true,
				ResourceName:      "braze_email_template.test",
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
					resource.TestCheckResourceAttr("braze_email_template.test", "subject", "Welcome"),
					resource.TestCheckResourceAttr("braze_email_template.test", "body", "<p>Hello</p>"),
					resource.TestCheckResourceAttr("braze_email_template.test", "plaintext_body", "Hello"),
					resource.TestCheckResourceAttr("braze_email_template.test", "preheader", "Preview text"),
					resource.TestCheckResourceAttr("braze_email_template.test", "tags.#", "2"),
					resource.TestCheckResourceAttr("braze_email_template.test", "tags.0", "tag1"),
					resource.TestCheckResourceAttr("braze_email_template.test", "tags.1", "tag2"),
					resource.TestCheckResourceAttr("braze_email_template.test", "should_inline_css", "true"),
				),
			},
			{
				ConfigDirectory: config.TestNameDirectory(),
				ConfigVariables: configVariables3,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("braze_email_template.test", "template_name", "test-email-template"),
					resource.TestCheckResourceAttr("braze_email_template.test", "subject", "Welcome"),
					resource.TestCheckResourceAttr("braze_email_template.test", "body", "<p>Hello</p>"),
					resource.TestCheckNoResourceAttr("braze_email_template.test", "plaintext_body"),
					resource.TestCheckNoResourceAttr("braze_email_template.test", "preheader"),
					resource.TestCheckResourceAttr("braze_email_template.test", "tags.#", "0"),
					resource.TestCheckResourceAttr("braze_email_template.test", "should_inline_css", "false"),
				),
			},
			{
				ConfigDirectory: config.TestNameDirectory(),
				ConfigVariables: configVariables4,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("braze_email_template.test", "template_name", "test-email-template"),
					resource.TestCheckResourceAttr("braze_email_template.test", "subject", "Welcome"),
					resource.TestCheckResourceAttr("braze_email_template.test", "body", "<p>Hello</p>"),
					resource.TestCheckNoResourceAttr("braze_email_template.test", "plaintext_body"),
					resource.TestCheckNoResourceAttr("braze_email_template.test", "preheader"),
					resource.TestCheckNoResourceAttr("braze_email_template.test", "tags"),
					resource.TestCheckResourceAttr("braze_email_template.test", "should_inline_css", "false"),
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
