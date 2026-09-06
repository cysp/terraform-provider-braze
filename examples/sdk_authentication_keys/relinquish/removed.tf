# Replace the resource block with this block, then apply to relinquish management.
# All registered keys remain in Braze.
removed {
  from = braze_sdk_authentication_keys.this
  lifecycle {
    destroy = false
  }
}
