#!/usr/bin/env bash
set -euo pipefail

: "${REDIS_ADDR:?REDIS_ADDR is required}"

module_directory="$(cd "$(dirname "$0")/.." && pwd)"
consumer="$(mktemp -d "${TMPDIR:-/tmp}/tenancy-redis.XXXXXX")"
cleanup() {
    find "${consumer}" -depth -delete
}
trap cleanup EXIT HUP INT TERM

cd "${consumer}"
GOWORK=off go mod init example.com/tenancy-redis
GOWORK=off go mod edit \
    -require=github.com/faustbrian/go-queue@v1.0.0 \
    -require=github.com/faustbrian/go-tenancy@v1.0.0
cp "${module_directory}/scripts/redis/consumer_test.go.tmpl" consumer_test.go
GOWORK=off go mod tidy
GOWORK=off go test -race -count=1 ./...
