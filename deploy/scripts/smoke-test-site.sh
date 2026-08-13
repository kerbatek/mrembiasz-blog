#!/usr/bin/env sh
set -eu

url="https://blog.mrembiasz.pl/"
expected="<title>Mateusz Rembiasz Blog</title>"

wget -qO- "$url" | grep -Fq "$expected"
