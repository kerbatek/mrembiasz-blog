#!/usr/bin/env sh
set -eu

url="https://blog.mrembiasz.pl/"
expected="name=\"deploy-id\" content=\"${GIT_COMMIT:?GIT_COMMIT is required}\""

wget -qO- "$url" | grep -Fq "$expected"
