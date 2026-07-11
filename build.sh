#!/bin/bash
set -euo pipefail

echo "prepare go mod"
go mod download
echo "building dist"
mkdir -p dist
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o dist/cat-led ./cmd/cat-led
echo "ensure permission"
chmod +x dist/cat-led
