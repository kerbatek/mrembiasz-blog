resource "aws_apigatewayv2_api" "analytics" {
  name          = "mrembiasz-blog-analytics"
  protocol_type = "HTTP"
  tags          = local.tags

  cors_configuration {
    allow_headers = ["content-type"]
    allow_methods = ["POST", "OPTIONS"]
    allow_origins = ["https://${local.domain_name}"]
    max_age       = 300
  }
}

resource "aws_apigatewayv2_integration" "analytics_validator" {
  api_id                 = aws_apigatewayv2_api.analytics.id
  integration_type       = "AWS_PROXY"
  integration_method     = "POST"
  integration_uri        = aws_lambda_function.analytics_validator.invoke_arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_route" "post_view" {
  api_id    = aws_apigatewayv2_api.analytics.id
  route_key = "POST /api/views/{slug}"
  target    = "integrations/${aws_apigatewayv2_integration.analytics_validator.id}"
}

resource "aws_apigatewayv2_stage" "analytics" {
  api_id      = aws_apigatewayv2_api.analytics.id
  name        = "$default"
  auto_deploy = true
  tags        = local.tags
}

resource "aws_lambda_permission" "analytics_api_gateway" {
  statement_id  = "AllowAnalyticsHttpApi"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.analytics_validator.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.analytics.execution_arn}/*/*"
}
