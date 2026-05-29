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

variable "email_template_plaintext_body" {
  type    = string
  default = null
}

variable "email_template_preheader" {
  type    = string
  default = null
}

variable "email_template_tags" {
  type    = list(string)
  default = null
}

variable "email_template_should_inline_css" {
  type    = bool
  default = null
}
