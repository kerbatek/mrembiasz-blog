variable "aws_region" {
  description = "AWS region for app-specific resources."
  type        = string
  default     = "eu-central-1"
}

variable "site_bucket_name" {
  description = "S3 bucket for built Astro assets."
  type        = string
  default     = "mrembiasz-blog"
}

variable "tags" {
  description = "Tags applied to resources that support tags."
  type        = map(string)
  default = {
    Project = "mrembiasz-blog"
  }
}
