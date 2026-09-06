resource "braze_catalog" "products" {
  name        = "products"
  description = "Product information for personalized messages"
  fields = [
    { name = "id", type = "string" },
    { name = "name", type = "string" },
    { name = "active", type = "boolean" },
  ]
}

resource "braze_catalog_item" "example" {
  catalog_name = braze_catalog.products.name
  item_id      = "product_1"
  values_json = jsonencode({
    name   = "Coffee"
    active = true
  })
  lifecycle {
    # Recreate the item if replacing its catalog removes all contained items.
    replace_triggered_by = [braze_catalog.products]
  }
}
