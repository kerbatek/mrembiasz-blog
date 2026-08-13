resource "aws_acm_certificate" "site" {
  provider          = aws.use1
  domain_name       = local.domain_name
  validation_method = "DNS"
  tags              = local.tags

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_acm_certificate_validation" "site" {
  provider        = aws.use1
  certificate_arn = aws_acm_certificate.site.arn
}
