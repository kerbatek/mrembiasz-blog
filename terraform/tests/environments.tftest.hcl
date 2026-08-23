mock_provider "aws" {
  mock_data "aws_iam_policy_document" {
    defaults = {
      json = "{\"Version\":\"2012-10-17\",\"Statement\":[]}"
    }
  }
}

mock_provider "aws" {
  alias = "use1"
}

mock_provider "random" {}

run "production_names_stay_unchanged" {
  command = plan

  assert {
    condition     = aws_s3_bucket.site.bucket == "mrembiasz-blog"
    error_message = "The production site bucket name changed."
  }

  assert {
    condition     = aws_dynamodb_table.aggregate_post_views.name == "mrembiasz-blog-aggregate-post-views"
    error_message = "The production DynamoDB table name changed."
  }

  assert {
    condition     = aws_lambda_function.backend_lambda["get_views"].function_name == "mrembiasz-blog-get-views"
    error_message = "The production Lambda prefix changed."
  }

  assert {
    condition = alltrue([
      aws_s3_bucket.cloudfront_logs.bucket == "mrembiasz-blog-cloudfront-logs",
      aws_s3_bucket.raw_analytics.bucket == "mrembiasz-blog-raw-analytics",
      aws_apigatewayv2_api.analytics.name == "mrembiasz-blog-analytics",
      aws_sns_topic.analytics_events.name == "mrembiasz-blog-analytics-events",
      aws_sqs_queue.aggregate_post_views.name == "mrembiasz-blog-aggregate-post-views",
      aws_sqs_queue.aggregate_post_views_dlq.name == "mrembiasz-blog-aggregate-post-views-dlq",
      aws_kinesis_firehose_delivery_stream.raw_analytics.name == "mrembiasz-blog-raw-analytics",
      aws_glue_catalog_database.analytics.name == "mrembiasz_blog_analytics",
      aws_iam_role.aggregate_views_lambda.name == "mrembiasz-blog-aggregate-views-lambda",
      aws_iam_role.analytics_validator_lambda.name == "mrembiasz-blog-analytics-validator-lambda",
      aws_iam_role.get_views_lambda.name == "mrembiasz-blog-get-views-lambda",
    ])
    error_message = "One or more production resource names changed."
  }

  assert {
    condition     = length(aws_s3_bucket_lifecycle_configuration.raw_analytics) == 0
    error_message = "Production raw analytics must remain unexpired."
  }
}

run "integration_is_isolated" {
  command = plan

  variables {
    environment                   = "integration"
    resource_prefix               = "mrembiasz-blog-integration"
    domain_name                   = "test.blog.mrembiasz.pl"
    jenkins_deploy_role_name      = "mrembiasz-blog-jenkins-integration"
    api_log_retention_days        = 7
    cloudfront_log_retention_days = 7
    raw_analytics_retention_days  = 7
  }

  assert {
    condition     = aws_s3_bucket.site.bucket == "mrembiasz-blog-integration"
    error_message = "The integration site bucket is not isolated."
  }

  assert {
    condition     = aws_apigatewayv2_api.analytics.name == "mrembiasz-blog-integration-analytics"
    error_message = "The integration API is not isolated."
  }

  assert {
    condition     = aws_lambda_function.backend_lambda["get_views"].function_name == "mrembiasz-blog-integration-get-views"
    error_message = "The integration Lambda prefix is not isolated."
  }

  assert {
    condition     = one(aws_s3_bucket_lifecycle_configuration.raw_analytics).rule[0].expiration[0].days == 7
    error_message = "Integration raw analytics must expire after seven days."
  }
}
