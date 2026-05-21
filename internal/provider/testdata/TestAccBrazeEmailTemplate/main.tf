resource "braze_email_template" "test" {
  template_name    = var.template_name
  description      = var.description
  subject          = var.subject
  preheader        = var.preheader
  body             = var.body
  plaintext_body   = var.plaintext_body
  should_inline_css = var.should_inline_css
  tags             = var.tags
}
