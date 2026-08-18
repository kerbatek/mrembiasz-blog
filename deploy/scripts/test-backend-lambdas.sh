#!/usr/bin/env sh
set -eu

cd src/backend
go test ./...
