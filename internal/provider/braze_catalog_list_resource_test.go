package provider_test

import (
	"encoding/json"
	"fmt"
	"testing"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	brazeclienttesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck/queryfilter"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccBrazeCatalogList(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()
	server.SetCatalog("centres", "Centre metadata", []brazeclient.CatalogField{
		{Name: "id", Type: brazeclient.CatalogFieldTypeString},
		{Name: "name", Type: brazeclient.CatalogFieldTypeString},
	})

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Query: true,
				Config: `
				provider "braze" {}

				list "braze_catalog" "test" {
					provider = braze

					include_resource = true
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("braze_catalog.test", "name", "centres"),
					resource.TestCheckResourceAttr("braze_catalog.test", "description", "Centre metadata"),
					resource.TestCheckResourceAttr("braze_catalog.test", "fields.#", "2"),
					resource.TestCheckResourceAttr("braze_catalog.test", "fields.0.name", "id"),
					resource.TestCheckResourceAttr("braze_catalog.test", "fields.0.type", "string"),
				),
			},
		},
	})
}

func TestAccBrazeCatalogListLimitAndIdentity(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()
	server.SetCatalog("centres", "Centre metadata", []brazeclient.CatalogField{
		{Name: "id", Type: brazeclient.CatalogFieldTypeString},
	})

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Query: true,
				Config: `
				provider "braze" {}

				list "braze_catalog" "test" {
					provider = braze

					limit = 1
				}
				`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("braze_catalog.test", 1),
					querycheck.ExpectIdentity("braze_catalog.test", map[string]knownvalue.Check{
						"name": knownvalue.StringExact("centres"),
					}),
					querycheck.ExpectResourceDisplayName(
						"braze_catalog.test",
						queryfilter.ByResourceIdentity(map[string]knownvalue.Check{
							"name": knownvalue.StringExact("centres"),
						}),
						knownvalue.StringExact("centres"),
					),
				},
			},
		},
	})
}

func TestAccBrazeCatalogListZeroLimit(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()
	server.SetCatalog("centres", "Centre metadata", []brazeclient.CatalogField{
		{Name: "id", Type: brazeclient.CatalogFieldTypeString},
	})

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Query: true,
				Config: `
				provider "braze" {}

				list "braze_catalog" "test" {
					provider = braze

					limit = 0
				}
				`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("braze_catalog.test", 0),
				},
			},
		},
	})
}

func TestAccBrazeCatalogItemList(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()
	server.SetCatalog("centres", "Centre metadata", []brazeclient.CatalogField{
		{Name: "id", Type: brazeclient.CatalogFieldTypeString},
		{Name: "name", Type: brazeclient.CatalogFieldTypeString},
	})
	server.SetCatalogItem("centres", "airportwest", map[string]json.RawMessage{
		"name": json.RawMessage(`"Airport West"`),
	})

	BrazeProviderMockedResourceTest(t, server, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Query: true,
				Config: `
				provider "braze" {}

				list "braze_catalog_item" "test" {
					provider = braze
					config {
						catalog_name = "centres"
					}
					include_resource = true
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("braze_catalog_item.test", "id", "centres/airportwest"),
					resource.TestCheckResourceAttr("braze_catalog_item.test", "catalog_name", "centres"),
					resource.TestCheckResourceAttr("braze_catalog_item.test", "item_id", "airportwest"),
					resource.TestCheckResourceAttr("braze_catalog_item.test", "values_json", `{"name":"Airport West"}`),
				),
			},
			{
				Query: true,
				Config: `
				provider "braze" {}

				list "braze_catalog_item" "test" {
					provider = braze
					config {
						catalog_name = "centres"
					}
					limit = 1
				}
				`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("braze_catalog_item.test", 1),
					querycheck.ExpectIdentity("braze_catalog_item.test", map[string]knownvalue.Check{
						"id": knownvalue.StringExact("centres/airportwest"),
					}),
					querycheck.ExpectResourceDisplayName(
						"braze_catalog_item.test",
						queryfilter.ByResourceIdentity(map[string]knownvalue.Check{
							"id": knownvalue.StringExact("centres/airportwest"),
						}),
						knownvalue.StringExact("airportwest"),
					),
				},
			},
			{
				Query: true,
				Config: `
				provider "braze" {}

				list "braze_catalog_item" "test" {
					provider = braze
					config {
						catalog_name = "centres"
					}
					limit = 0
				}
				`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("braze_catalog_item.test", 0),
				},
			},
		},
	})
}

func TestAccBrazeCatalogItemListPagination(t *testing.T) {
	t.Parallel()

	server, _ := brazeclienttesting.NewBrazeServer()
	server.SetCatalog("centres", "Centre metadata", []brazeclient.CatalogField{
		{Name: "id", Type: brazeclient.CatalogFieldTypeString},
		{Name: "name", Type: brazeclient.CatalogFieldTypeString},
	})

	for i := range 55 {
		id := fmt.Sprintf("centre-%03d", i)

		server.SetCatalogItem("centres", id, map[string]json.RawMessage{
			"name": json.RawMessage(fmt.Sprintf("%q", id)),
		})
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

				list "braze_catalog_item" "test" {
					provider = braze
					config {
						catalog_name = "centres"
					}
					limit = 55
				}
				`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("braze_catalog_item.test", 55),
				},
			},
			{
				Query: true,
				Config: `
				provider "braze" {}

				list "braze_catalog_item" "test" {
					provider = braze
					config {
						catalog_name = "centres"
					}
					limit = 51
				}
				`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("braze_catalog_item.test", 51),
				},
			},
		},
	})
}
