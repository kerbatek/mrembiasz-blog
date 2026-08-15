#!/usr/bin/env bash
set -eu

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

lambda_dirs=(src/backend/lambdas/*)

for lambda_dir in "${lambda_dirs[@]}"; do
  lambda_name="$(basename "$lambda_dir")"
  function_name="mrembiasz-blog-${lambda_name//_/-}"
  update_lambda "$function_name" "deploy/backend-lambdas/${lambda_name}.zip"
done

for lambda_dir in "${lambda_dirs[@]}"; do
  lambda_name="$(basename "$lambda_dir")"
  function_name="mrembiasz-blog-${lambda_name//_/-}"
  wait_lambda "$function_name"
done
