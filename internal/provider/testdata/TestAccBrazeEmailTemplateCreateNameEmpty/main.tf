resource "braze_email_template" "test" {
  template_name = var.email_template_name
  subject       = var.email_template_subject
  body          = var.email_template_body
}
