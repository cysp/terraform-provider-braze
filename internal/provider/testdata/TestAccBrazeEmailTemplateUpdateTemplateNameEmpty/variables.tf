variable "template_name" {
  type    = string
  default = "test-email-template"
}

variable "subject" {
  type    = string
  default = "Test Subject"
}

variable "body" {
  type    = string
  default = "<h1>Test Body</h1>"
}
