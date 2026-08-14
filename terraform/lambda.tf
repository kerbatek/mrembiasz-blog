data "archive_file" "aggregate_views_lambda" {
  type        = "zip"
  source_file = "${path.module}/../src/backend/aggregate_views/handler.py"
  output_path = "${path.module}/aggregate_views_lambda.zip"
}

resource "aws_lambda_function" "aggregate_views" {
  function_name    = "mrembiasz-blog-aggregate-views"
  role             = aws_iam_role.aggregate_views_lambda.arn
  handler          = "handler.lambda_handler"
  runtime          = "python3.12"
  filename         = data.archive_file.aggregate_views_lambda.output_path
  source_code_hash = data.archive_file.aggregate_views_lambda.output_base64sha256
  timeout          = 10
  tags             = local.tags

  environment {
    variables = {
      POST_VIEWS_TABLE_NAME = aws_dynamodb_table.aggregate_post_views.name
    }
  }
}

resource "aws_lambda_event_source_mapping" "aggregate_views" {
  event_source_arn        = aws_sqs_queue.aggregate_post_views.arn
  function_name           = aws_lambda_function.aggregate_views.arn
  batch_size              = 10
  function_response_types = ["ReportBatchItemFailures"]

  depends_on = [
    aws_iam_role_policy_attachment.aggregate_views_lambda,
  ]
}
