#!/usr/bin/env sh
set -eu

: "${RELEASE_ARTIFACT_SHA256:?RELEASE_ARTIFACT_SHA256 is required}"

archive="release-artifacts.tar.gz"

printf '%s  %s\n' "$RELEASE_ARTIFACT_SHA256" "$archive" | sha256sum -c -
tar -xzf "$archive"
