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
    ]
  })
}

resource "aws_iam_role_policy_attachment" "jenkins_deploy" {
  role       = "mrembiasz-blog-jenkins-deploy"
  policy_arn = aws_iam_policy.jenkins_deploy.arn
}
