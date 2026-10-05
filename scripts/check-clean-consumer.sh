#!/usr/bin/env bash
set -euo pipefail

module_directory="$(cd "$(dirname "$0")/.." && pwd)"
consumer="$(mktemp -d "${TMPDIR:-/tmp}/tenancy-consumer.XXXXXX")"
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
GOWORK=off go mod init example.com/tenancy-consumer
GOWORK=off go mod edit \
    -require=github.com/faustbrian/go-audit@v1.0.0 \
    -require=github.com/faustbrian/go-cache@v1.0.0 \
    -require=github.com/faustbrian/go-cloudevents/adapters/golib@v1.0.0 \
    -require=github.com/faustbrian/go-queue@v1.0.0 \
    -require=github.com/faustbrian/go-search@v1.0.0 \
    -require=github.com/faustbrian/go-telemetry@v1.0.0 \
    -require=github.com/faustbrian/go-tenancy/v2@v2.0.0 \
    -require=github.com/faustbrian/go-workflow@v1.0.0 \
    -require=go.opentelemetry.io/otel/sdk/metric@v1.44.0
mkdir consumer
printf '%s\n' 'package consumer' \
    'import (' \
    '  "context"' \
    '  "github.com/faustbrian/go-tenancy/v2"' \
    '  tenancyhttp "github.com/faustbrian/go-tenancy/v2/http"' \
    '  tenancyjsonrpc "github.com/faustbrian/go-tenancy/v2/jsonrpc"' \
    '  tenancypostgres "github.com/faustbrian/go-tenancy/v2/postgres"' \
    ')' \
    'var _ = context.Background' \
    'var _ = tenancy.ParseTenantID' \
    'var _ = tenancyhttp.New' \
    'var _ = tenancyjsonrpc.New' \
    'var _ = tenancypostgres.NewManager' > consumer/consumer.go
cp "${module_directory}/scripts/clean-consumer/consumer_test.go.tmpl" consumer/consumer_test.go
cp "${module_directory}/scripts/clean-consumer/providers_test.go.tmpl" consumer/providers_test.go
cp "${module_directory}/scripts/clean-consumer/administration_test.go.tmpl" consumer/administration_test.go
GOWORK=off go mod tidy
GOWORK=off go test -race ./...
