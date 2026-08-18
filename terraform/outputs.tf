output "site_bucket_name" {
  description = "S3 bucket that stores built Astro assets."
  value       = aws_s3_bucket.site.bucket
}

output "cloudfront_distribution_id" {
  description = "CloudFront distribution ID."
  value       = aws_cloudfront_distribution.site.id
}

output "cloudfront_domain_name" {
  description = "CloudFront distribution domain name."
  value       = aws_cloudfront_distribution.site.domain_name
}

output "analytics_api_endpoint" {
  description = "HTTP API endpoint for analytics ingestion."
  value       = aws_apigatewayv2_api.analytics.api_endpoint
}

output "raw_analytics_bucket_name" {
  description = "S3 bucket that stores raw analytics events in Parquet format."
  value       = aws_s3_bucket.raw_analytics.bucket
}

output "acm_validation_records" {
  description = "DNS CNAME records required to validate the ACM certificate."
  value = {
    for option in aws_acm_certificate.site.domain_validation_options :
    option.domain_name => {
      name  = option.resource_record_name
      type  = option.resource_record_type
      value = option.resource_record_value
    }
  }
}
