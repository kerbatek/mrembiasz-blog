variable "environment" {
  description = "Deployment environment."
  type        = string
  default     = "production"

  validation {
    condition     = contains(["production", "integration"], var.environment)
    error_message = "environment must be production or integration."
  }
}

variable "resource_prefix" {
  description = "Prefix used for environment-specific AWS resources."
  type        = string
  default     = "mrembiasz-blog"
}

variable "domain_name" {
  description = "Public hostname for the environment."
  type        = string
  default     = "blog.mrembiasz.pl"
}

variable "jenkins_deploy_role_name" {
  description = "Existing Jenkins role that deploys this environment."
  type        = string
  default     = "mrembiasz-blog-jenkins-deploy"
}

variable "api_log_retention_days" {
  description = "API Gateway log retention."
  type        = number
  default     = 30
}

variable "cloudfront_log_retention_days" {
  description = "CloudFront and S3 access log retention."
  type        = number
  default     = 90
}

variable "raw_analytics_retention_days" {
  description = "Raw analytics retention, or null to retain indefinitely."
  type        = number
  default     = null
  nullable    = true
}
