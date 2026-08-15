#!/usr/bin/env sh
set -eu

for lambda_dir in src/backend/lambdas/*
do
  (
    cd "$lambda_dir"
    go test ./...
  )
done
