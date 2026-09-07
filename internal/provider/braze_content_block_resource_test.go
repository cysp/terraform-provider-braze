package provider_test

import (
	"regexp"
	"testing"

	brazeclienttesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccBrazeContentBlock(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()

	configVariables1 := config.Variables{
		"content_block_name":    config.StringVariable("test-content-block"),
		"content_block_content": config.StringVariable("lorem ipsum"),
	}

	configVariables2 := config.Variables{
		"content_block_name":    config.StringVariable("test-content-block"),
		"content_block_content": config.StringVariable("lorem ipsum"),
		"content_block_tags":    config.ListVariable(config.StringVariable("tag1"), config.StringVariable("tag2")),
	}

	configVariables3 := config.Variables{
		"content_block_name":    config.StringVariable("test-content-block"),
		"content_block_content": config.StringVariable("lorem ipsum"),
		"content_block_tags":    config.ListVariable(),
	}

	configVariables4 := config.Variables{
		"content_block_name":    config.StringVariable("test-content-block"),
		"content_block_content": config.StringVariable("lorem ipsum"),
	}

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ConfigDirectory: config.TestNameDirectory(),
				ConfigVariables: configVariables1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("braze_content_block.test", "name", "test-content-block"),
					resource.TestCheckNoResourceAttr("braze_content_block.test", "description"),
					resource.TestCheckResourceAttr("braze_content_block.test", "content", "lorem ipsum"),
					resource.TestCheckNoResourceAttr("braze_content_block.test", "tags"),
				),
			},
			{
				ConfigDirectory:   config.TestNameDirectory(),
				ImportState:       true,
				ImportStateVerify: true,
				ResourceName:      "braze_content_block.test",
			},
			{
				ConfigDirectory: config.TestNameDirectory(),
				ConfigVariables: configVariables1,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("braze_content_block.test", plancheck.ResourceActionNoop),
					},
				},
			},
			{
				ConfigDirectory: config.TestNameDirectory(),
				ConfigVariables: configVariables2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("braze_content_block.test", "name", "test-content-block"),
					resource.TestCheckNoResourceAttr("braze_content_block.test", "description"),
					resource.TestCheckResourceAttr("braze_content_block.test", "content", "lorem ipsum"),
					resource.TestCheckResourceAttr("braze_content_block.test", "tags.#", "2"),
					resource.TestCheckResourceAttr("braze_content_block.test", "tags.0", "tag1"),
					resource.TestCheckResourceAttr("braze_content_block.test", "tags.1", "tag2"),
				),
			},
			{
				ConfigDirectory: config.TestNameDirectory(),
				ConfigVariables: configVariables3,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("braze_content_block.test", "name", "test-content-block"),
					resource.TestCheckNoResourceAttr("braze_content_block.test", "description"),
					resource.TestCheckResourceAttr("braze_content_block.test", "content", "lorem ipsum"),
					resource.TestCheckResourceAttr("braze_content_block.test", "tags.#", "0"),
				),
			},
			{
				ConfigDirectory: config.TestNameDirectory(),
				ConfigVariables: configVariables4,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("braze_content_block.test", "name", "test-content-block"),
					resource.TestCheckNoResourceAttr("braze_content_block.test", "description"),
					resource.TestCheckResourceAttr("braze_content_block.test", "content", "lorem ipsum"),
					resource.TestCheckNoResourceAttr("braze_content_block.test", "tags"),
				),
			},
			{
				ConfigDirectory: config.TestNameDirectory(),
				Destroy:         true,
				ResourceName:    "braze_content_block.test",
			},
		},
	})
}

func TestAccBrazeTagsRejectNullElementsDuringPlan(t *testing.T) {
	t.Parallel()

	for name, configuration := range map[string]string{
		"content block": `resource "braze_content_block" "test" {
name = "welcome"
content = "Hello"
tags = [null]
 }`,
		"email template": `resource "braze_email_template" "test" {
template_name = "welcome"
subject = "Welcome"
body = "Hello"
tags = [null]
 }`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server, _ := brazeclienttesting.NewBrazeServer()
			BrazeProviderMockedResourceTest(t, server, resource.TestCase{Steps: []resource.TestStep{{
				Config:      configuration,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("Invalid tags"),
			}}})
		})
	}
}
