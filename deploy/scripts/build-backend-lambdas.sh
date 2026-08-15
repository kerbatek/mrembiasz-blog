#!/usr/bin/env sh
set -eu

cd src/backend/analytics_validator
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-s -w" -o bootstrap .
