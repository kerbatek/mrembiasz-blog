#!/usr/bin/env sh
set -eu

archive="release-artifacts.tar.gz"

tar -czf "$archive" \
  dist \
  deploy/backend-lambdas \
  terraform/tfplan

sha256sum "$archive" | cut -d ' ' -f 1
