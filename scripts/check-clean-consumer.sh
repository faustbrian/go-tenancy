#!/usr/bin/env bash
set -euo pipefail

module_directory="$(cd "$(dirname "$0")/.." && pwd)"
consumer="$(mktemp -d "${TMPDIR:-/tmp}/tenancy-consumer.XXXXXX")"
cleanup() {
    chmod -R u+w "${consumer}" 2>/dev/null || true
    rm -rf "${consumer}"
}
trap cleanup EXIT HUP INT TERM

cd "${consumer}"
GOWORK=off go mod init example.com/tenancy-consumer
GOWORK=off go mod edit \
    -require=github.com/faustbrian/go-audit@v1.0.0 \
    -require=github.com/faustbrian/go-cache@v1.0.0 \
    -require=github.com/faustbrian/go-cloudevents/adapters/golib@v1.0.0 \
    -require=github.com/faustbrian/go-queue@v1.0.0 \
    -require=github.com/faustbrian/go-search@v1.0.0 \
    -require=github.com/faustbrian/go-telemetry@v1.0.0 \
    -require=github.com/faustbrian/go-tenancy@v1.0.0 \
    -require=github.com/faustbrian/go-workflow@v1.0.0 \
    -require=go.opentelemetry.io/otel/sdk/metric@v1.44.0
mkdir consumer
printf '%s\n' 'package consumer' \
    'import (' \
    '  "context"' \
    '  "github.com/faustbrian/go-tenancy"' \
    '  tenancyhttp "github.com/faustbrian/go-tenancy/http"' \
    '  tenancyjsonrpc "github.com/faustbrian/go-tenancy/jsonrpc"' \
    '  tenancypostgres "github.com/faustbrian/go-tenancy/postgres"' \
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
