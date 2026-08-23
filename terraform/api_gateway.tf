resource "aws_apigatewayv2_api" "analytics" {
  name          = "${var.resource_prefix}-analytics"
  protocol_type = "HTTP"
  tags          = local.tags

  cors_configuration {
    allow_headers = ["content-type"]
    allow_methods = ["GET", "POST", "OPTIONS"]
    allow_origins = ["https://${local.domain_name}"]
    max_age       = 300
  }
}

resource "aws_apigatewayv2_integration" "analytics_validator" {
  api_id                 = aws_apigatewayv2_api.analytics.id
  integration_type       = "AWS_PROXY"
  integration_method     = "POST"
  integration_uri        = aws_lambda_function.backend_lambda["analytics_validator"].invoke_arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_integration" "get_views" {
  api_id                 = aws_apigatewayv2_api.analytics.id
  integration_type       = "AWS_PROXY"
  integration_method     = "POST"
  integration_uri        = aws_lambda_function.backend_lambda["get_views"].invoke_arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_route" "post_view" {
  api_id    = aws_apigatewayv2_api.analytics.id
  route_key = "POST /api/views/{slug+}"
  target    = "integrations/${aws_apigatewayv2_integration.analytics_validator.id}"
}

resource "aws_apigatewayv2_route" "get_views" {
  api_id    = aws_apigatewayv2_api.analytics.id
  route_key = "GET /api/views/{slug+}"
  target    = "integrations/${aws_apigatewayv2_integration.get_views.id}"
}

resource "aws_apigatewayv2_stage" "analytics" {
  api_id      = aws_apigatewayv2_api.analytics.id
  name        = "$default"
  auto_deploy = true
  tags        = local.tags

  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.analytics_api.arn
    format = jsonencode({
      requestId      = "$context.requestId"
      ip             = "$context.identity.sourceIp"
      requestTime    = "$context.requestTime"
      httpMethod     = "$context.httpMethod"
      routeKey       = "$context.routeKey"
      status         = "$context.status"
      responseLength = "$context.responseLength"
    })
  }

  default_route_settings {
    throttling_burst_limit = 10
    throttling_rate_limit  = 5
  }

  route_settings {
    route_key              = aws_apigatewayv2_route.get_views.route_key
    throttling_burst_limit = 20
    throttling_rate_limit  = 10
  }
}

resource "aws_cloudwatch_log_group" "analytics_api" {
  name              = "/aws/apigateway/${var.resource_prefix}-analytics"
  retention_in_days = var.api_log_retention_days
  tags              = local.tags
}

resource "aws_lambda_permission" "analytics_api_gateway" {
  statement_id  = "AllowAnalyticsHttpApi"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.backend_lambda["analytics_validator"].function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.analytics.execution_arn}/*/*"
}

resource "aws_lambda_permission" "get_views_api_gateway" {
  statement_id  = "AllowGetViewsHttpApi"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.backend_lambda["get_views"].function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.analytics.execution_arn}/*/*"
}
