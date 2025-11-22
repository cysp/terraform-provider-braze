package provider_test

import (
	"testing"

	brazeclienttesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccBrazeContentBlockList(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()

	server.SetContentBlock("content-block-id", "test-content-block", "<p>This is <strong>HTML</strong> content</p>", "", []string{})

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Query: true,
				Config: `
				provider "braze" {}

				list "braze_content_block" "test" {
					provider = braze
					config {
						modified_after = "1970-01-01T00:00:00Z"
						modified_before = "9999-01-01T00:00:00Z"
					}
					include_resource = true
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("braze_content_block.test", "id", "content-block-id"),
					resource.TestCheckResourceAttr("braze_content_block.test", "name", "test-content-block"),
					resource.TestCheckResourceAttr("braze_content_block.test", "description", ""),
					resource.TestCheckResourceAttr("braze_content_block.test", "content", "<p>This is <strong>HTML</strong> content</p>"),
					resource.TestCheckResourceAttr("braze_content_block.test", "tags.#", "0"),
				),
			},
		},
	})
}
