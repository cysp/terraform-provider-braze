variable "template_name" {
  type    = string
  default = "test-email-template"
}

variable "description" {
  type    = string
  default = null
}

variable "subject" {
  type    = string
  default = "Test Subject"
}

variable "preheader" {
  type    = string
  default = null
}

variable "body" {
  type    = string
  default = "<h1>Test Body</h1>"
}

variable "plaintext_body" {
  type    = string
  default = null
}

variable "should_inline_css" {
  type    = bool
  default = null
}

variable "tags" {
  type    = list(string)
  default = null
}
