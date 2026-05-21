resource "braze_email_template" "test" {
  template_name = var.template_name
  subject       = var.subject
  body          = var.body
}
