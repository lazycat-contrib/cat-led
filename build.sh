#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")"
version="${VERSION:-$(awk '/^version:/ {gsub(/[\047\042]/, "", $2); print $2; exit}' package.yml)}"
if [[ ! "$version" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.+-]+)?$ ]]; then
    echo "Invalid application version: $version" >&2
    exit 1
fi

echo "prepare go mod"
go mod download
echo "building dist"
mkdir -p dist
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
    -ldflags "-X cat-led/internal/buildinfo.Version=$version" \
    -o dist/cat-led ./cmd/cat-led
echo "ensure permission"
chmod +x dist/cat-led
