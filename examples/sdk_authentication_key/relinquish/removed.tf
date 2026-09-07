# Remove the resource block while relinquishing these keys from Terraform state.
removed {
  from = braze_sdk_authentication_key.this

  lifecycle {
    destroy = false
  }
}
