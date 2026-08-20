#!/usr/bin/env sh
set -eu

cd src/backend
go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...
