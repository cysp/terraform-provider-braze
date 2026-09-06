# Terraform 1.12 or later. Configure the provider for the owning workspace.
import {
  to = braze_sdk_authentication_key.this["current"]
  identity = {
    app_id = "01234567-89ab-cdef-0123-456789abcdef"
    id     = "fedcba98-7654-3210-fedc-ba9876543210"
  }
}
