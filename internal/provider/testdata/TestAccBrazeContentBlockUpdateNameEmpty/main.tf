resource "braze_content_block" "test" {
  name        = var.content_block_name
  description = var.content_block_description
  content     = var.content_block_content
  tags        = var.content_block_tags
}
