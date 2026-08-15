#!/usr/bin/env bash
set -eu

lambdas=(
  "mrembiasz-blog-aggregate-views=deploy/backend-lambdas/aggregate_views.zip"
  "mrembiasz-blog-analytics-validator=deploy/backend-lambdas/analytics_validator.zip"
  "mrembiasz-blog-get-views=deploy/backend-lambdas/get_views.zip"
)

update_lambda() {
  function_name="$1"
  package_path="$2"

  aws lambda update-function-code \
    --function-name "$function_name" \
    --zip-file "fileb://${package_path}" \
    >/dev/null
}

wait_lambda() {
  function_name="$1"
  aws lambda wait function-updated --function-name "$function_name"
}

for lambda in "${lambdas[@]}"; do
  function_name="${lambda%%=*}"
  package_path="${lambda#*=}"
  update_lambda "$function_name" "$package_path"
done

for lambda in "${lambdas[@]}"; do
  function_name="${lambda%%=*}"
  wait_lambda "$function_name"
done
