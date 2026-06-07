package provider_test

import (
	"regexp"
	"testing"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	brazeclienttesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

const testCatalogAndCatalogItemConfig = `
provider "braze" {}

resource "braze_catalog" "test" {
  name        = "centres"
  description = "Centre metadata"

  fields = [
    {
      name = "id"
      type = "string"
    },
    {
      name = "name"
      type = "string"
    },
    {
      name = "active"
      type = "boolean"
    },
  ]
}

resource "braze_catalog_item" "test" {
  catalog_name = braze_catalog.test.name
  item_id      = "airportwest"
  values_json  = jsonencode({ name = "Airport West", active = true })
}
`

const testCatalogImportConfig = `
provider "braze" {}

resource "braze_catalog" "test" {
  name        = "centres"
  description = "Centre metadata"

  fields = [
    {
      name = "id"
      type = "string"
    },
    {
      name = "name"
      type = "string"
    },
  ]
}
`

func TestAccBrazeCatalogAndCatalogItem(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: testCatalogAndCatalogItemConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("braze_catalog.test", "name", "centres"),
					resource.TestCheckResourceAttr("braze_catalog.test", "description", "Centre metadata"),
					resource.TestCheckResourceAttr("braze_catalog.test", "fields.#", "3"),
					resource.TestCheckResourceAttr("braze_catalog.test", "fields.0.name", "id"),
					resource.TestCheckResourceAttr("braze_catalog.test", "fields.0.type", "string"),
					resource.TestCheckResourceAttr("braze_catalog_item.test", "id", "centres/airportwest"),
					resource.TestCheckResourceAttr("braze_catalog_item.test", "catalog_name", "centres"),
					resource.TestCheckResourceAttr("braze_catalog_item.test", "item_id", "airportwest"),
					resource.TestCheckResourceAttr("braze_catalog_item.test", "values_json", `{"active":true,"name":"Airport West"}`),
				),
			},
			{
				Config: testCatalogAndCatalogItemConfig,
			},
		},
	})
}

func TestAccBrazeCatalogImport(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()
	server.SetCatalog("centres", "Centre metadata", []brazeclient.CatalogField{
		{Name: "id", Type: brazeclient.CatalogFieldTypeString},
		{Name: "name", Type: brazeclient.CatalogFieldTypeString},
	})

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:        testCatalogImportConfig,
				ResourceName:  "braze_catalog.test",
				ImportState:   true,
				ImportStateId: "centres",
			},
		},
	})
}

func TestAccBrazeCatalogItemImport(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testCatalogAndCatalogItemConfig,
			},
			{
				Config:          testCatalogAndCatalogItemConfig,
				ResourceName:    "braze_catalog_item.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func TestAccBrazeCatalogValidation(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: `
provider "braze" {}

resource "braze_catalog" "test" {
  name        = "centres"
  description = "Centre metadata"

  fields = [
    {
      name = "name"
      type = "string"
    },
  ]
}
`,
				ExpectError: regexp.MustCompile(`first catalog field`),
			},
			{
				Config: `
provider "braze" {}

resource "braze_catalog" "test" {
  name        = "centres"
  description = "Centre metadata"

  fields = [
    {
      name = "id"
      type = "string"
    },
    {
      name = "name"
      type = "unsupported"
    },
  ]
}
`,
				ExpectError: regexp.MustCompile(`Braze catalog field type must be one of`),
			},
			{
				Config: `
provider "braze" {}

resource "braze_catalog_item" "test" {
  catalog_name = "centres"
  item_id      = "airportwest"
  values_json  = "not json"
}
`,
				ExpectError: regexp.MustCompile(`Invalid JSON String Value`),
			},
			{
				Config: `
provider "braze" {}

resource "braze_catalog_item" "test" {
  catalog_name = "centres"
  item_id      = "airportwest"
  values_json  = "null"
}
`,
				ExpectError: regexp.MustCompile(`values_json must be a JSON object`),
			},
			{
				Config: `
provider "braze" {}

resource "braze_catalog_item" "test" {
  catalog_name = "centres"
  item_id      = "airportwest"
  values_json  = jsonencode({ id = "airportwest", name = "Airport West" })
}
`,
				ExpectError: regexp.MustCompile(`values_json must not include id`),
			},
		},
	})
}
