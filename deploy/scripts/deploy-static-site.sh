#!/usr/bin/env sh
set -eu

bucket_name="$(cat terraform/site_bucket_name.txt)"
distribution_id="$(cat terraform/cloudfront_distribution_id.txt)"

aws s3 sync dist/ "s3://${bucket_name}" \
  --delete \
  --exclude "*.html" \
  --cache-control "public, max-age=31536000, immutable"

aws s3 sync dist/ "s3://${bucket_name}" \
  --delete \
  --exclude "*" \
  --include "*.html" \
  --cache-control "public, max-age=60" \
  --content-type "text/html; charset=utf-8"

invalidation_id="$(
  aws cloudfront create-invalidation \
  --distribution-id "${distribution_id}" \
  --paths "/*" \
  --query "Invalidation.Id" \
  --output text
)"

aws cloudfront wait invalidation-completed \
  --distribution-id "${distribution_id}" \
  --id "${invalidation_id}"
