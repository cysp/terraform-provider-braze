variable "app_id" {
  type = string
}

variable "sdk_authentication_keys" {
  type = map(object({
    rsa_public_key = string
    description    = string
  }))
}

variable "primary_key" {
  type = string
}

resource "braze_sdk_authentication_key" "this" {
  for_each = var.sdk_authentication_keys

  app_id         = var.app_id
  rsa_public_key = each.value.rsa_public_key
  description    = each.value.description

  # A false value would imply a direct demotion operation, which Braze does
  # not provide. Omit the claim with null on every other key.
  primary = each.key == var.primary_key ? true : null
}
