#!/usr/bin/env sh
set -eu

for lambda_dir in src/backend/lambdas/*
do
  (
    cd "$lambda_dir"
    GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-s -w" -o bootstrap .
  )
  lambda_name="$(basename "$lambda_dir")"
  mkdir -p deploy/backend-lambdas
  zip -j -q "deploy/backend-lambdas/${lambda_name}.zip" "$lambda_dir/bootstrap"
done
