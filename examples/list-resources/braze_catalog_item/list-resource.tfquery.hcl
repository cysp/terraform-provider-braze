# Use alongside a .tf file containing required_providers and provider configuration.
# Terraform 1.14 or later.
list "braze_catalog_item" "existing" {
  provider         = braze
  include_resource = true
  limit            = 100
  config { catalog_name = "products" }
}
