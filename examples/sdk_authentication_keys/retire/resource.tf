resource "braze_sdk_authentication_keys" "this" {
  app_id = "01234567-89ab-cdef-0123-456789abcdef"

  keys = [
    {
      rsa_public_key = file("next.pem")
      description    = "Next signing key"
      primary        = true
    },
  ]
}
