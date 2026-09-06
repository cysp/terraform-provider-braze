# Starting from current (primary) and backup, create next as primary and remove
# current in one apply. A third slot must be free before creating next.
resource "braze_sdk_authentication_keys" "this" {
  app_id = "01234567-89ab-cdef-0123-456789abcdef"

  keys = [
    {
      rsa_public_key = file("backup.pem")
      description    = "Backup signing key"
      primary        = false
    },
    {
      rsa_public_key = file("next.pem")
      description    = "Next signing key"
      primary        = true
    },
  ]
}
