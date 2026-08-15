locals {
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
        ANALYTICS_TOPIC_ARN = aws_sns_topic.analytics_events.arn
      }
    }
    get_views = {
      role_arn = aws_iam_role.get_views_lambda.arn
      environment = {
        POST_VIEWS_TABLE_NAME = aws_dynamodb_table.aggregate_post_views.name
      }
    }
  }
}

data "archive_file" "backend_lambda" {
  for_each = local.backend_lambdas

  type        = "zip"
  source_file = "${path.module}/../src/backend/lambdas/${each.key}/bootstrap"
  output_path = "${path.module}/${each.key}_lambda.zip"
}

resource "aws_lambda_function" "backend_lambda" {
  for_each = local.backend_lambdas

  function_name    = "mrembiasz-blog-${replace(each.key, "_", "-")}"
  role             = each.value.role_arn
  handler          = "bootstrap"
  runtime          = "provided.al2023"
  filename         = data.archive_file.backend_lambda[each.key].output_path
  source_code_hash = data.archive_file.backend_lambda[each.key].output_base64sha256
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

moved {
  from = aws_lambda_function.aggregate_views
  to   = aws_lambda_function.backend_lambda["aggregate_views"]
}

moved {
  from = aws_lambda_function.analytics_validator
  to   = aws_lambda_function.backend_lambda["analytics_validator"]
}

moved {
  from = aws_lambda_function.get_views
  to   = aws_lambda_function.backend_lambda["get_views"]
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
