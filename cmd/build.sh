#!/bin/bash

cd "$(dirname "$0")/.."
GOOS=linux go build -o dist/cat-led ./cmd
