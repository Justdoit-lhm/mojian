#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
mkdir -p internal/remotebin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' \
  -o internal/remotebin/mojian-server-linux-amd64 ./cmd/mojian-server
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags='-s -w' \
  -o internal/remotebin/mojian-server-linux-arm64 ./cmd/mojian-server
