package provider_test

import (
	"maps"
	"testing"

	brazeclienttesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccBrazeContentBlock(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()

	configVariables := config.Variables{
		"content_block_name": config.StringVariable("test-content-block"),
	}

	configVariables1 := maps.Clone(configVariables)
	configVariables1["content_block_content"] = config.StringVariable(".")

	configVariables2 := maps.Clone(configVariables1)
	configVariables2["content_block_tags"] = config.ListVariable(config.StringVariable("tag1"), config.StringVariable("tag2"))

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ConfigDirectory: config.TestNameDirectory(),
				ConfigVariables: configVariables1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("braze_content_block.test", "name", "test-content-block"),
					resource.TestCheckResourceAttr("braze_content_block.test", "tags.#", "0"),
				),
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
					resource.TestCheckResourceAttr("braze_content_block.test", "tags.#", "2"),
				),
			},
		},
	})
}
