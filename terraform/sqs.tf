resource "aws_sqs_queue" "aggregate_post_views_dlq" {
  name = "mrembiasz-blog-aggregate-post-views-dlq"
  tags = local.tags
}

resource "aws_sqs_queue" "aggregate_post_views" {
  name = "mrembiasz-blog-aggregate-post-views"
  tags = local.tags

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.aggregate_post_views_dlq.arn
    maxReceiveCount     = 5
  })
}
