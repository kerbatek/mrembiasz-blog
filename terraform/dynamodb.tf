resource "aws_dynamodb_table" "aggregate_post_views" {
  name         = "${var.resource_prefix}-aggregate-post-views"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "post_slug"
  tags         = local.tags

  attribute {
    name = "post_slug"
    type = "S"
  }
}
