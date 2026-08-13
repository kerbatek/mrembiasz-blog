#!/usr/bin/env sh
set -eu

terraform -chdir=terraform apply -input=false tfplan
terraform -chdir=terraform output -raw site_bucket_name > terraform/site_bucket_name.txt
terraform -chdir=terraform output -raw cloudfront_distribution_id > terraform/cloudfront_distribution_id.txt
