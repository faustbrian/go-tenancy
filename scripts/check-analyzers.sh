#!/usr/bin/env bash
set -euo pipefail

security=false
if [[ $# -eq 1 && "$1" == --security ]]; then
    security=true
elif [[ $# -ne 0 ]]; then
    printf 'usage: check-analyzers.sh [--security]\n' >&2
    exit 2
fi

module_directory="$(cd "$(dirname "$0")/.." && pwd)"
temporary_directory="$(mktemp -d "${TMPDIR:-/tmp}/tenancy-analyzers.XXXXXX")"
cleanup() {
    find "${temporary_directory}" -depth -delete
}
trap cleanup EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

analyzer="${temporary_directory}/golib-analysis"
fixture="${temporary_directory}/fixture"
policy="${fixture}/analysis.yml"
consumer_report="${temporary_directory}/consumer.json"
adapter_report="${temporary_directory}/adapter.json"

GOBIN="${temporary_directory}" GOWORK=off go install \
    github.com/faustbrian/go-analysis/cmd/golib-analysis@v1.0.0

mkdir -p "${fixture}/analyzerfixture"
find "${module_directory}" -maxdepth 1 -type f -name '*.go' \
    ! -name '*_test.go' -exec cp '{}' "${fixture}" ';'
for package in adapter consumer metrics; do
    mkdir -p "${fixture}/analyzerfixture/${package}"
    sed 's|github.com/faustbrian/go-tenancy/v2/testdata/analyzer|github.com/faustbrian/go-tenancy/v2/analyzerfixture|g' \
        "${module_directory}/testdata/analyzer/${package}/${package}.go" \
        >"${fixture}/analyzerfixture/${package}/${package}.go"
done
sed 's|github.com/faustbrian/go-tenancy/v2/testdata/analyzer|github.com/faustbrian/go-tenancy/v2/analyzerfixture|g' \
    "${module_directory}/analysis.yml" >"${policy}"
(
    cd "${fixture}"
    GOWORK=off go mod init github.com/faustbrian/go-tenancy/v2
    GOWORK=off go mod edit \
        -require=github.com/faustbrian/go-audit@v1.0.0 \
        -require=github.com/faustbrian/go-cache@v1.0.0 \
        -require=github.com/faustbrian/go-queue@v1.0.0 \
        -require=github.com/faustbrian/go-telemetry@v1.0.0 \
        -require=github.com/faustbrian/go-workflow@v1.0.0
    GOWORK=off go mod tidy
)

"${analyzer}" validate-config "${policy}" >/dev/null

set +e
(
    cd "${fixture}"
    GOWORK=off "${analyzer}" check \
        -config "${policy}" -root "${fixture}" -format json \
        ./analyzerfixture/consumer
) >"${consumer_report}"
consumer_status=$?
set -e
if [[ ${consumer_status} -ne 1 ]]; then
    printf 'tenancy analyzer consumer exit status: got %d, want 1\n' "${consumer_status}" >&2
    exit 1
fi

assert_diagnostic_count() {
    local rule="$1"
    local expected="$2"
    local actual
    actual="$(grep -oE "\\\"rule\\\":\\\"${rule}\\\"" "${consumer_report}" | wc -l | tr -d ' ')"
    if [[ "${actual}" != "${expected}" ]]; then
        printf '%s diagnostics: got %s, want %s\n' "${rule}" "${actual}" "${expected}" >&2
        exit 1
    fi
}

assert_diagnostic_count 'api/forbidden-call' 5
assert_diagnostic_count 'context/no-background' 1
assert_diagnostic_count 'observability/high-cardinality-label' 1

(
    cd "${fixture}"
    GOWORK=off "${analyzer}" check \
        -config "${policy}" -root "${fixture}" -format json \
        ./analyzerfixture/adapter
) >"${adapter_report}"
if ! grep -q '"diagnostics":\[\]' "${adapter_report}"; then
    printf 'reviewed tenancy adapter emitted analyzer diagnostics\n' >&2
    exit 1
fi

if [[ "${security}" == true ]]; then
    # Install before imposing scanner-output limits: compilation uses the
    # job-owned Go caches, not the scanner's private report budget.
    GOBIN="${temporary_directory}" GOWORK=off go install \
        github.com/securego/gosec/v2/cmd/gosec@v2.29.0
    # Populate export/build caches before the scanner's file-size limit reaches
    # its package-loader subprocesses; compilation is not report generation.
    (
        cd "${fixture}"
        GOWORK=off go list -deps -export ./...
    ) >"${temporary_directory}/exports.log" 2>&1
    scanner_status=0
    (
        cd "${fixture}" || exit "$?"
        ulimit -c 0 || exit "$?"
        ulimit -f 8192 || exit "$?"
        GOWORK=off "${temporary_directory}/gosec" \
            -nosec-require-rules -nosec-require-justification \
            -fmt=json -out="${temporary_directory}/gosec.json" ./...
    ) >"${temporary_directory}/gosec.log" 2>&1 || scanner_status=$?
    if [[ ${scanner_status} -ne 0 ]]; then
        printf 'prepared tenancy fixture security scan failed (status %d); private report withheld\n' \
            "${scanner_status}" >&2
        exit "${scanner_status}"
    fi
    printf 'prepared tenancy fixture security scan passed\n'
fi
