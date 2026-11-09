#!/bin/sh
# Builds the Linux mojian-server binaries embedded in the app. It runs from
# `go generate ./internal/remotebin`, or directly as sh internal/remotebin/build.sh.
set -eu
cd "$(dirname "$0")/../.."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' \
  -o internal/remotebin/mojian-server-linux-amd64 ./cmd/mojian-server
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags='-s -w' \
  -o internal/remotebin/mojian-server-linux-arm64 ./cmd/mojian-server
