resource "braze_content_block" "example" {
  name        = "My Content Block"
  description = "An example content block for email campaigns"
  content     = "<p>This is <strong>HTML</strong> content for the block.</p>"
  tags        = ["example", "html"]
}
