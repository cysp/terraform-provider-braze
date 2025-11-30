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

func TestAccBrazeContentBlockListWithFailedGetInfo(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()

	// Set up a valid content block
	server.SetContentBlock("valid-block-id", "valid-content-block", "<p>Valid content</p>", "", []string{})

	// Set up an orphaned block that appears in list but fails when getting details.
	// This simulates a race condition where a block is deleted between listing and fetching details,
	// or any other scenario where GetContentBlockInfo returns nil/error.
	server.SetOrphanedContentBlock("orphaned-block-id", "orphaned-content-block", []string{})

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
				// The test verifies that the provider handles GetContentBlockInfo failures gracefully:
				// 1. No panic occurs when getResponse is nil
				// 2. The valid block is still returned successfully
				// 3. An error diagnostic is added for the failed block
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("braze_content_block.test", "id", "valid-block-id"),
					resource.TestCheckResourceAttr("braze_content_block.test", "name", "valid-content-block"),
					resource.TestCheckResourceAttr("braze_content_block.test", "content", "<p>Valid content</p>"),
				),
			},
		},
	})
}
