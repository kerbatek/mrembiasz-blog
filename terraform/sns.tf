resource "aws_sns_topic" "analytics_events" {
  name = "${var.resource_prefix}-analytics-events"
  tags = local.tags
}

resource "aws_sns_topic_subscription" "aggregate_post_views" {
  topic_arn            = aws_sns_topic.analytics_events.arn
  protocol             = "sqs"
  endpoint             = aws_sqs_queue.aggregate_post_views.arn
  raw_message_delivery = true
}

resource "aws_sns_topic_subscription" "raw_analytics" {
  topic_arn             = aws_sns_topic.analytics_events.arn
  protocol              = "firehose"
  endpoint              = aws_kinesis_firehose_delivery_stream.raw_analytics.arn
  subscription_role_arn = aws_iam_role.analytics_events_sns_firehose.arn
  raw_message_delivery  = true
  depends_on            = [aws_iam_role_policy_attachment.analytics_events_sns_firehose]
}
