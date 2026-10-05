#!/usr/bin/env bash
set -euo pipefail

: "${POSTGRES_URL:?POSTGRES_URL is required}"

module_directory="$(cd "$(dirname "$0")/.." && pwd)"
consumer="$(mktemp -d "${TMPDIR:-/tmp}/tenancy-postgres-consumer.XXXXXX")"
cleanup() {
    find "${consumer}" -type d -exec chmod u+w {} +
    find "${consumer}" -depth -delete
}
trap cleanup EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

export GOCACHE="${consumer}/.task-go/build"
export GOMODCACHE="${consumer}/.task-go/modules"
export GOTMPDIR="${consumer}/.task-go/tmp"
export GOWORK=off GOTOOLCHAIN=local GOSUMDB=sum.golang.org
export GOPROXY=https://proxy.golang.org,direct GONOSUMDB= GONOPROXY= GOPRIVATE=
mkdir -p "${GOCACHE}" "${GOMODCACHE}" "${GOTMPDIR}"

cd "${consumer}"
GOWORK=off go mod init example.com/tenancy-postgres-consumer
GOWORK=off go mod edit \
    -require=github.com/faustbrian/go-audit@v1.0.0 \
    -require=github.com/faustbrian/go-audit/postgres@v1.0.0 \
    -require=github.com/faustbrian/go-tenancy/v2@v2.0.0 \
    -require=github.com/faustbrian/go-workflow@v1.0.0 \
    -require=github.com/jackc/pgx/v5@v5.10.0
cp "${module_directory}/scripts/postgres/consumer_test.go.tmpl" consumer_test.go
GOWORK=off go mod tidy
GOWORK=off go test -race -count=1 ./...
