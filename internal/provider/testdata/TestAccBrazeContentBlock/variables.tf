variable "content_block_name" {
  type    = string
  default = "test-content-block"
}

variable "content_block_description" {
  type    = string
  default = null
}

variable "content_block_content" {
  type    = string
  default = "<p>This is <strong>HTML</strong> content</p>"
}

variable "content_block_tags" {
  type    = list(string)
  default = null
}
