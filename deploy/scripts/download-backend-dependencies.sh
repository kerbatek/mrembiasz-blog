#!/usr/bin/env sh
set -eu

[ -z "${GOMODCACHE:-}" ] || mkdir -p "$GOMODCACHE"
[ -z "${GOCACHE:-}" ] || mkdir -p "$GOCACHE"

cd src/backend
go mod download
