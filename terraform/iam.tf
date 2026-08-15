data "aws_iam_role" "jenkins_deploy" {
  name = "mrembiasz-blog-jenkins-deploy"
}

resource "aws_iam_policy" "jenkins_deploy" {
  name = "mrembiasz-blog-deploy"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:ListBucket",
        ]
        Resource = aws_s3_bucket.site.arn
      },
      {
        Effect = "Allow"
        Action = [
          "s3:DeleteObject",
          "s3:GetObject",
          "s3:PutObject",
        ]
        Resource = "${aws_s3_bucket.site.arn}/*"
      },
      {
        Effect = "Allow"
        Action = [
          "cloudfront:CreateInvalidation",
        ]
        Resource = aws_cloudfront_distribution.site.arn
      },
      {
        Effect = "Allow"
        Action = [
          "lambda:GetFunctionConfiguration",
          "lambda:UpdateFunctionCode",
        ]
        Resource = [
          for lambda in aws_lambda_function.backend_lambda : lambda.arn
        ]
      },
    ]
  })
}

resource "aws_iam_role_policy_attachment" "jenkins_deploy" {
  role       = data.aws_iam_role.jenkins_deploy.name
  policy_arn = aws_iam_policy.jenkins_deploy.arn
}

data "aws_iam_policy_document" "lambda_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "aggregate_views_lambda" {
  name               = "mrembiasz-blog-aggregate-views-lambda"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json
  tags               = local.tags
}

data "aws_iam_policy_document" "aggregate_views_lambda" {
  statement {
    actions = [
      "sqs:ChangeMessageVisibility",
      "sqs:DeleteMessage",
      "sqs:GetQueueAttributes",
      "sqs:ReceiveMessage",
    ]

    resources = [aws_sqs_queue.aggregate_post_views.arn]
  }

  statement {
    actions   = ["dynamodb:UpdateItem"]
    resources = [aws_dynamodb_table.aggregate_post_views.arn]
  }
}

resource "aws_iam_policy" "aggregate_views_lambda" {
  name   = "mrembiasz-blog-aggregate-views-lambda"
  policy = data.aws_iam_policy_document.aggregate_views_lambda.json
}

resource "aws_iam_role_policy_attachment" "aggregate_views_lambda" {
  role       = aws_iam_role.aggregate_views_lambda.name
  policy_arn = aws_iam_policy.aggregate_views_lambda.arn
}

resource "aws_iam_role_policy_attachment" "aggregate_views_lambda_logs" {
  role       = aws_iam_role.aggregate_views_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role" "analytics_validator_lambda" {
  name               = "mrembiasz-blog-analytics-validator-lambda"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json
  tags               = local.tags
}

data "aws_iam_policy_document" "analytics_validator_lambda" {
  statement {
    actions   = ["sns:Publish"]
    resources = [aws_sns_topic.analytics_events.arn]
  }
}

resource "aws_iam_policy" "analytics_validator_lambda" {
  name   = "mrembiasz-blog-analytics-validator-lambda"
  policy = data.aws_iam_policy_document.analytics_validator_lambda.json
}

resource "aws_iam_role_policy_attachment" "analytics_validator_lambda" {
  role       = aws_iam_role.analytics_validator_lambda.name
  policy_arn = aws_iam_policy.analytics_validator_lambda.arn
}

resource "aws_iam_role_policy_attachment" "analytics_validator_lambda_logs" {
  role       = aws_iam_role.analytics_validator_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role" "get_views_lambda" {
  name               = "mrembiasz-blog-get-views-lambda"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json
  tags               = local.tags
}

data "aws_iam_policy_document" "get_views_lambda" {
  statement {
    actions   = ["dynamodb:GetItem"]
    resources = [aws_dynamodb_table.aggregate_post_views.arn]
  }
}

resource "aws_iam_policy" "get_views_lambda" {
  name   = "mrembiasz-blog-get-views-lambda"
  policy = data.aws_iam_policy_document.get_views_lambda.json
}

resource "aws_iam_role_policy_attachment" "get_views_lambda" {
  role       = aws_iam_role.get_views_lambda.name
  policy_arn = aws_iam_policy.get_views_lambda.arn
}

resource "aws_iam_role_policy_attachment" "get_views_lambda_logs" {
  role       = aws_iam_role.get_views_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}
