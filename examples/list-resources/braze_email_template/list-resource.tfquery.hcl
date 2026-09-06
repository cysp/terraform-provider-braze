# Use alongside a .tf file containing required_providers and provider configuration.
# Terraform 1.14 or later.
list "braze_email_template" "existing" {
  provider         = braze
  include_resource = true
  limit            = 100
}
