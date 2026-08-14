resource "aws_sns_topic" "analytics_events" {
  name = "mrembiasz-blog-analytics-events"
  tags = local.tags
}

resource "aws_sns_topic_subscription" "aggregate_post_views" {
  topic_arn            = aws_sns_topic.analytics_events.arn
  protocol             = "sqs"
  endpoint             = aws_sqs_queue.aggregate_post_views.arn
  raw_message_delivery = true
}
