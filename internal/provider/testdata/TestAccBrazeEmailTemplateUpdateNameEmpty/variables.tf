variable "email_template_name" {
  type    = string
  default = "test-email-template"
}

variable "email_template_subject" {
  type    = string
  default = "Welcome"
}

variable "email_template_body" {
  type    = string
  default = "<p>Hello</p>"
}
