#!/usr/bin/env sh
set -eu

[ -z "${GOMODCACHE:-}" ] || mkdir -p "$GOMODCACHE"
[ -z "${GOCACHE:-}" ] || mkdir -p "$GOCACHE"

for lambda_dir in src/backend/*
do
  (
    cd "$lambda_dir"
    go mod download
  )
done
