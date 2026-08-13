#!/usr/bin/env sh
set -eu

npm ci
npm run lint:frontend
npm run build
