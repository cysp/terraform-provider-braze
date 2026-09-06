resource "braze_catalog" "products" {
  name        = "products"
  description = "Product information for personalized messages"
  fields = [
    { name = "id", type = "string" },
    { name = "name", type = "string" },
    { name = "active", type = "boolean" },
  ]
}
