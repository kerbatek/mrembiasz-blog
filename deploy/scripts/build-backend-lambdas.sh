#!/usr/bin/env sh
set -eu

for lambda_dir in \
  src/backend/aggregate_views \
  src/backend/analytics_validator \
  src/backend/get_views
do
  (
    cd "$lambda_dir"
    GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-s -w" -o bootstrap .
  )
done
