resource "braze_email_template" "example" {
  template_name  = "Example email template"
  subject        = "Welcome"
  body           = "<p>Hello {{${first_name}}}</p>"
  plaintext_body = "Hello {{${first_name}}}"
  preheader      = "Welcome to our newsletter"
  tags           = ["example"]
}
