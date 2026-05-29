resource "braze_email_template" "test" {
  template_name     = var.email_template_name
  subject           = var.email_template_subject
  body              = var.email_template_body
  plaintext_body    = var.email_template_plaintext_body
  preheader         = var.email_template_preheader
  tags              = var.email_template_tags
  should_inline_css = var.email_template_should_inline_css
}
