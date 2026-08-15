#!/usr/bin/env sh
set -eu

deploy_lambda() {
  function_name="$1"
  package_path="$2"

  aws lambda update-function-code \
    --function-name "$function_name" \
    --zip-file "fileb://${package_path}" \
    >/dev/null

  aws lambda wait function-updated --function-name "$function_name"
}

deploy_lambda "mrembiasz-blog-aggregate-views" "deploy/backend-lambdas/aggregate_views.zip"
deploy_lambda "mrembiasz-blog-analytics-validator" "deploy/backend-lambdas/analytics_validator.zip"
deploy_lambda "mrembiasz-blog-get-views" "deploy/backend-lambdas/get_views.zip"
