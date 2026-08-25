#!/usr/bin/env bash
set -euo pipefail

: "${OPENSEARCH_URL:?OPENSEARCH_URL is required}"
: "${OPENSEARCH_EXPECTED_VERSION:?OPENSEARCH_EXPECTED_VERSION is required}"

module_directory="$(cd "$(dirname "$0")/.." && pwd)"
consumer="$(mktemp -d "${TMPDIR:-/tmp}/tenancy-opensearch.XXXXXX")"
cleanup() {
    find "${consumer}" -depth -delete
}
trap cleanup EXIT HUP INT TERM

cd "${consumer}"
GOWORK=off go mod init example.com/tenancy-opensearch
GOWORK=off go mod edit \
    -require=github.com/faustbrian/go-search@v1.0.0 \
    -require=github.com/faustbrian/go-search/adapters/opensearch@v1.0.0 \
    -require=github.com/faustbrian/go-tenancy@v1.0.0
cp "${module_directory}/scripts/opensearch/consumer_test.go.tmpl" consumer_test.go
GOWORK=off go mod tidy
GOWORK=off go test -race -count=1 ./...
