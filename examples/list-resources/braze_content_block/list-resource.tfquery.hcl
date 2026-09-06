# Use alongside a .tf file containing required_providers and provider configuration.
# Terraform 1.14 or later.
list "braze_content_block" "existing" {
  provider         = braze
  include_resource = true
  limit            = 100
}
