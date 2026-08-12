output "site_bucket_name" {
  description = "S3 bucket that stores built Astro assets."
  value       = aws_s3_bucket.site.bucket
}
