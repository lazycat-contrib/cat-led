#!/bin/bash
echo "prepare go mod"
go mod tidy && go mod download
echo "building dist"
GOOS=linux GOARCH=amd64 go build -o dist/cat-led ./cmd/cat-led
echo "ensure permission"
chmod +x dist/cat-led