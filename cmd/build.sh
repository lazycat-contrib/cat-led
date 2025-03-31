#!/bin/bash

cd "$(dirname "$0")/.."
GOOS=linux GOARCH=amd64 go build -o dist/cat-led ./cmd
