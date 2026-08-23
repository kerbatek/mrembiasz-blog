resource "aws_sqs_queue" "aggregate_post_views_dlq" {
  name = "${var.resource_prefix}-aggregate-post-views-dlq"
  tags = local.tags
}

resource "aws_sqs_queue" "aggregate_post_views" {
  name = "${var.resource_prefix}-aggregate-post-views"
  tags = local.tags

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.aggregate_post_views_dlq.arn
    maxReceiveCount     = 5
  })
}

data "aws_iam_policy_document" "aggregate_post_views_queue" {
  statement {
    actions   = ["sqs:SendMessage"]
    resources = [aws_sqs_queue.aggregate_post_views.arn]

    principals {
      type        = "Service"
      identifiers = ["sns.amazonaws.com"]
    }

    condition {
      test     = "ArnEquals"
      variable = "aws:SourceArn"
      values   = [aws_sns_topic.analytics_events.arn]
    }
  }
}

resource "aws_sqs_queue_policy" "aggregate_post_views" {
  queue_url = aws_sqs_queue.aggregate_post_views.id
  policy    = data.aws_iam_policy_document.aggregate_post_views_queue.json
}
