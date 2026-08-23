locals {
  # ponytail: Lambda environment variables have a 4 KB ceiling; move this allowlist to DynamoDB if the blog outgrows it.
  valid_post_slugs = sort([
    for file in setunion(
      fileset("${path.module}/../src/frontend/content/blog", "**/*.md"),
      fileset("${path.module}/../src/frontend/content/blog", "**/*.mdx"),
    ) : trimsuffix(trimsuffix(file, ".mdx"), ".md")
  ])

  backend_lambdas = {
    aggregate_views = {
      role_arn = aws_iam_role.aggregate_views_lambda.arn
      environment = {
        POST_VIEWS_TABLE_NAME = aws_dynamodb_table.aggregate_post_views.name
      }
    }
    analytics_validator = {
      role_arn = aws_iam_role.analytics_validator_lambda.arn
      environment = {
        ANALYTICS_ALLOWED_ORIGIN = "https://${local.domain_name}"
        ANALYTICS_ORIGIN_SECRET  = random_password.analytics_origin.result
        ANALYTICS_TOPIC_ARN      = aws_sns_topic.analytics_events.arn
        VALID_POST_SLUGS         = jsonencode(local.valid_post_slugs)
      }
    }
    get_views = {
      role_arn = aws_iam_role.get_views_lambda.arn
      environment = {
        ANALYTICS_ORIGIN_SECRET = random_password.analytics_origin.result
        POST_VIEWS_TABLE_NAME   = aws_dynamodb_table.aggregate_post_views.name
        VALID_POST_SLUGS        = jsonencode(local.valid_post_slugs)
      }
    }
  }
}

resource "random_password" "analytics_origin" {
  length  = 32
  special = false
}

resource "aws_lambda_function" "backend_lambda" {
  for_each = local.backend_lambdas

  function_name    = "${var.resource_prefix}-${replace(each.key, "_", "-")}"
  role             = each.value.role_arn
  handler          = "bootstrap"
  runtime          = "provided.al2023"
  filename         = "${path.module}/../deploy/backend-lambdas/${each.key}.zip"
  source_code_hash = filebase64sha256("${path.module}/../deploy/backend-lambdas/${each.key}.zip")
  architectures    = ["arm64"]
  memory_size      = 512
  timeout          = 10
  tags             = local.tags

  environment {
    variables = each.value.environment
  }

  lifecycle {
    ignore_changes = [
      filename,
      source_code_hash,
    ]
  }
}


resource "aws_lambda_event_source_mapping" "aggregate_views" {
  event_source_arn        = aws_sqs_queue.aggregate_post_views.arn
  function_name           = aws_lambda_function.backend_lambda["aggregate_views"].arn
  batch_size              = 10
  function_response_types = ["ReportBatchItemFailures"]

  depends_on = [
    aws_iam_role_policy_attachment.aggregate_views_lambda,
  ]
}
