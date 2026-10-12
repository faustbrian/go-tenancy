# Source documentation audit

## Scope and compact plan

Base: `dc21348fb53c5df401b2b7d33f78b20a016deef0` (`origin/main`).
Tier A documentation-only source pass owned by this standalone repository.
Direct consumers are tenant/scope constructors, context and propagation callers,
provider wrapper authors, background-group owners, administrative sources/audit
callbacks, SQL/RLS migration owners, and test-helper users. Existing reverse
module edges name the CloudEvents Golib and Service reference-HTTP adapters;
unchanged APIs do not require a consumer or Identity Platform campaign.

Audit every tracked Go file and public declaration/field/interface method;
document actual identity versus authorization, bounded routing/namespace inputs,
provider isolation limits, synchronous error and panic ownership, async lifetime,
SQL cleanup and archive/audit responsibilities. Preserve runtime behavior,
module/API/source pins, analyzer negative-fixture calls and historical evidence.
Use affected formatting, parsed/rendered docs, local links, changed examples,
changed-content secrets and one ordinary complete final-diff inspection.

## Reproducible inventory and generation

Use `git ls-files '*.go' 'go.mod' '**/go.mod'`; inspect package clauses, exported
AST declarations, public struct/interface members and all `//go:` directives.
Use `.golib.yaml`, `modules.json`, `packages.json`, the Makefile and `scripts/`
to reproduce module/package and native rendering scope.

One module: `github.com/faustbrian/go-tenancy/v2`; minimum Go 1.27.0, repository
producer Go 1.27.2. Five catalog packages: root `tenancy`, `http` (`tenancyhttp`),
`jsonrpc` (`tenancyjsonrpc`), `postgres` (`tenancypostgres`), and test support
`tenancytest`. Three additional tracked fixture packages (`adapter`, `consumer`,
`metrics`) live under `testdata/analyzer/`; their scripts relocate them into a
disposable, separately dependency-prepared module. They are not missing public
modules and are parsed/documented, not directly run as production packages.

The configured API baseline is `api/v2-baseline.txt`, **apidiff**, not a prose
snapshot. Native scripts contain no go-doc snapshot generator. Historical API
files remain immutable. No generator or generated Go source was found.
Canonical comments already exist for all eight package directories; extend the
five public/test-support headers where useful, without duplicating them. Fixture
function comments distinguish static matching from runtime success or metric I/O.

## Every tracked Go file

| File | Baseline lines | Investigation |
| --- | --- | --- |
| `administration.go` | 183 | public source contracts |
| `administration_test.go` | 338 | tests and helper/assertion audit; 8 entry points |
| `background.go` | 236 | public source contracts |
| `background_test.go` | 891 | tests and helper/assertion audit; 15 entry points |
| `context.go` | 82 | public source contracts |
| `core_edge_test.go` | 215 | tests and helper/assertion audit; 6 entry points |
| `enforcement.go` | 47 | public source contracts |
| `example_test.go` | 43 | tests and helper/assertion audit; 2 entry points |
| `fuzz_test.go` | 58 | tests and helper/assertion audit; 2 entry points |
| `http/fuzz_test.go` | 40 | tests and helper/assertion audit; 1 entry points |
| `http/http.go` | 138 | public source contracts |
| `http/http_edge_test.go` | 64 | tests and helper/assertion audit; 2 entry points |
| `http/http_test.go` | 134 | tests and helper/assertion audit; 5 entry points |
| `identity.go` | 139 | public source contracts |
| `identity_test.go` | 161 | tests and helper/assertion audit; 6 entry points |
| `integration.go` | 171 | public source contracts |
| `integration_edge_test.go` | 381 | tests and helper/assertion audit; 7 entry points |
| `integration_test.go` | 86 | tests and helper/assertion audit; 2 entry points |
| `jsonrpc/fuzz_test.go` | 33 | tests and helper/assertion audit; 1 entry points |
| `jsonrpc/jsonrpc.go` | 181 | public source contracts |
| `jsonrpc/jsonrpc_edge_test.go` | 79 | tests and helper/assertion audit; 3 entry points |
| `jsonrpc/jsonrpc_test.go` | 130 | tests and helper/assertion audit; 5 entry points |
| `jsonrpc/mutation_boundary_internal_test.go` | 41 | tests and helper/assertion audit; 1 entry points |
| `leak_test.go` | 62 | tests and helper/assertion audit; 1 entry points |
| `model_soak_test.go` | 133 | tests and helper/assertion audit; 2 entry points |
| `mutation_boundary_internal_test.go` | 250 | tests and helper/assertion audit; 5 entry points |
| `namespace.go` | 100 | public source contracts |
| `namespace_enforcement_test.go` | 166 | tests and helper/assertion audit; 5 entry points |
| `operation.go` | 22 | public source contracts |
| `postgres/fake_driver_test.go` | 266 | tests and helper/assertion audit; 0 entry points |
| `postgres/integration_test.go` | 725 | integration-tagged live RLS/pool and proxy failover fixture |
| `postgres/manager.go` | 207 | public source contracts |
| `postgres/manager_test.go` | 351 | tests and helper/assertion audit; 7 entry points |
| `postgres/mutation_resource_guard_unix_test.go` | 77 | Unix-tagged mutation-only TestMain resource/lock guard |
| `postgres/sql.go` | 159 | public source contracts |
| `postgres/sql_test.go` | 137 | tests and helper/assertion audit; 3 entry points |
| `propagation.go` | 156 | public source contracts |
| `propagation_test.go` | 152 | tests and helper/assertion audit; 4 entry points |
| `property_test.go` | 198 | tests and helper/assertion audit; 6 entry points |
| `scope.go` | 266 | public source contracts |
| `scope_context_test.go` | 156 | tests and helper/assertion audit; 4 entry points |
| `security_admission_diagnostic_test.go` | 142 | tests and helper/assertion audit; 4 entry points |
| `tenancytest/tenancytest.go` | 60 | public source contracts |
| `tenancytest/tenancytest_test.go` | 32 | tests and helper/assertion audit; 1 entry points |
| `testdata/analyzer/adapter/adapter.go` | 20 | relocated analyzer fixture; intentional static positive/negative calls, not production |
| `testdata/analyzer/consumer/consumer.go` | 25 | relocated analyzer fixture; intentional static positive/negative calls, not production |
| `testdata/analyzer/metrics/metrics.go` | 7 | relocated analyzer fixture; intentional static positive/negative calls, not production |

## Findings, evidence and rationale

Missing: exported configuration/result fields and interface methods in group,
administration, propagation, transport and SQL APIs. Existing methods generally
have name-led comments but several lifecycle, ownership and trust caveats need
semantic clarification. Private helper members and intentional analyzer fixture
calls do not need filler or runtime changes.

- Identity/scope/context: `identity.go`, `scope.go`, `context.go`,
  `enforcement.go`, `operation.go`, identity/core-edge/scope-context tests.
  Tenant IDs are exact ASCII routing values, not authenticated claims. Text/JSON
  decoding owns raw inputs before final ID validation; the 128-byte decoded-ID
  ceiling does not impose an encoded-JSON or transport allocation limit. Metadata
  and reasons have printable-text/count bounds, not credential detection.
  Scope equality includes metadata and reasons; constructors/capabilities do
  not authorize work. Direct functional callbacks are synchronous and do not
  contain panics or arbitrary errors. Context access does not itself check
  cancellation; callers retain parent lifetime.
- Propagation and transports: `propagation.go`, HTTP/JSON-RPC source plus all
  transport tests/fuzz files. Custom carriers own resource bounds and mutation
  safety. MapCarrier aliases remain mutable and its copying Values method does
  not cap counts. Presence/duplicate/conflict/count classification precedes the
  generic trust refusal. HTTP counts case-insensitive header entries, not comma
  splitting; trust callbacks own peer authentication and tenant authorization.
  JSON metadata duplicate detection applies to top-level members only; nested
  non-tenant RawMessage contents are not recursively duplicate-checked. Injection
  returns new bounded bytes, preserving semantics rather than original encoding.
  The adapter is not a complete JSON-RPC protocol implementation.
- Namespace/integration: `namespace.go`, `integration.go`, namespace,
  integration/property/soak and hosted admission diagnostic tests. HMAC-SHA-256
  uses package-owned `tenancy-namespace-v2` framing and `tn2_` lowercase hex.
  Key-length checks do not prove entropy; keys are copied, retained and not
  erased. Scope kind/tenant/domain/logical key are encoded, not metadata or audit
  reasons. Integration adds a length-delimited boundary and accepts tenant scope
  only. Deterministic outputs permit repeated-input correlation, not permissions, durable
  operations or provider enforcement. Provider and dual-read key rotation remain
  application-owned. No external cryptographic conformance claim is introduced.
- Background lifetime: `background.go`, all group/internal/race-boundary/leak
  tests. Group's mutex owns admission/terminal state, semaphore owns capacity,
  and task cancellation is linked to submission and group contexts. Parent must
  carry no explicit scope, including ScopeUnscoped. Submit waits, does not drop;
  task errors go only to HandleError (or are discarded if nil), not Drain.
  Cancellation cannot force a callback to return. Panics are not contained and
  handlers must not join their own unfinished group. Lifecycle wait timeouts do
  not reopen intake or undo graceful close.
- Administrative iteration: `administration.go`, administration and integration
  edge tests. Unscoped base means no installed Scope, not ScopeUnscoped. Source
  receives base context, audit receives system scope, work a fresh tenant child.
  Bounds are PageSize 1..1000, MaxTenants 1..1000000, cursor at most 256 bytes.
  Validation rejects duplicates within the invocation and cursor cycles but
  cannot prove stable snapshot provenance. Failed audit/work returns a retry
  token at that tenant; side effects may repeat and need idempotency. Returned
  errors/panics are caller-owned, not sanitized.
- SQL/RLS: `postgres/sql.go`, `postgres/manager.go`, fake-driver/unit and tagged
  live integration fixtures. Predicate/DDL only return strings/arguments and do
  not install policies or authorize queries. Restricted roles, FORCE RLS and
  matching Manager/plan settings remain deployment-owned. Final readback cannot
  prevent temporary callback setting mutation. Manager controls transaction
  commit/rollback and leased connection; cleanup detaches cancellation under
  configured context timeout, but driver cooperation/close duration remain
  outside that bound. A cleanup failure can accompany a successful commit;
  errors are not proof of rollback and no implicit retry is provided. Driver and
  operation errors are retained/joined, unlike safe owned classification text.
- Test support/analyzer fixtures: `tenancytest` and the three fixture packages,
  `analysis.yml`, `scripts/check-analyzers.sh`, [analyzer guide](analyzers.md).
  Test helpers use testing.TB and do not authorize System scope. Fixtures keep
  reviewed exact exceptions and intentionally invalid provider/telemetry/context
  calls. Script templates are service/consumer evidence, not tracked Go files.

Current narrow prose contradictions: blanket redacted-error claim in the active
[boundary inventory](inventory.md), and any general duplicate-key wording that
implies recursive JSON checks. Historical dated [security review](security-review.md),
old release entries, `.ai` goals, immutable fixture/provider pins and analyzer
policy are preserved. No decision register or pinned external specification is
declared in this repository; the namespace version and validated metadata
contracts above are package-owned. No runtime repair or Identity Platform work
is part of this pass.

## Tests, variants, examples and omitted campaigns

All 47 tracked Go files are covered. Four fuzz targets cover tenant IDs, generic
propagation, HTTP and JSON metadata. Three root benchmarks cover context,
namespace encoding and tenant assertion. Two root examples exist; only the
namespace example needs explicit scope-error handling and a deterministic-key
caveat. The basic Example already handles errors and remains unchanged.

`postgres/integration_test.go` is integration-tagged. The mutation TestMain is
Unix-platform-tagged and adds locking/heap/wall guards only when its mutation
coverage environment variable is set. Fuzz input/invariant and benchmark
workload/setup comments are clarified without
changing executable logic; the fake driver explicitly separates local lifecycle
model evidence from real PostgreSQL/RLS proof. Root admission diagnostics are
hosted-only
runtime conditions, not new build variants. No generated files, go:generate/
go:embed directives or unresolved TODO/FIXME/HACK/XXX markers were found in
tracked Go sources. The existing `Group.Close` deprecation remains idiomatic
(`Deprecated: use Drain.`); no NOTE/WARNING marker needs resolution. Core,
transport, SQL and fixture helper logic was included in the source/static
contract audit; broad live-service, race, active fuzz,
mutation, soak, failover, provider/consumer and release campaigns are omitted
because runtime/inputs/providers/SQL policies are unchanged.

## Verification and remaining boundaries

Completed under the pinned Go 1.27.2 producer with `GOWORK=off`:

- `gofmt -w` on the 23 changed Go files.
- Disposable standard-library parser/scanner/AST comparison: all 47 Go files,
  including both tagged test files and three analyzer fixtures; 239 name-led
  public declarations/fields/interface methods; one canonical package comment
  in each of eight directories; 263 parsed comment groups rendered as HTML.
  All non-example executable ASTs and tokens are exactly equal to base, with
  no semicolon normalization needed. Analyzer calls remain unchanged.
- Both API baselines, `go.mod`, `go.sum`, `analysis.yml`, build/lint directives
  and historical evidence are unchanged. No prose snapshot regeneration applies.
- `golib source-docs check`: one module boundary, 47 Go files, passed.
- `golib docs check --local`: passed, including after the final audit update.
- `go test -mod=readonly . -run '^ExampleNamespaceEncoder$' -count=1
  -timeout=2m`: passed, with unchanged `68 tn2_` output. The unchanged basic
  Example was not executed. Unchanged runtime assertions and fuzz bodies were
  not executed simply because comments were added.
- `go doc -all` for `.`, `./http`, `./jsonrpc`, `./postgres` and
  `./tenancytest`: all five catalog-package renders passed. Fixture comments
  were parsed/HTML-rendered without inventing dependency availability in the
  root module. The final decoded-ID-bound wording received an affected root
  render/parser/native-check refresh; unchanged example proof was reused.

The first disposable helper had an import/function name collision; its alias
was corrected before any passing parser/render claim. All failed and successful
sessions removed their exact owned `GOCACHE`, `GOMODCACHE`, `GOTMPDIR`, helpers
and render output, preserving pre-existing `.verification` evidence. Final
changed-content redacted secret scan, local links, ordinary complete-diff review
and Intent/ACK hygiene cover the committed handoff. Coordinator-owned CLI and
scanner binaries remain retained by their owner, not published-tool claims.

Real provider isolation, SQL privileges and RLS installation, trusted peer and
tenant authorization, key entropy/rotation, cooperative callbacks, stable admin
snapshots and durable audit/retry remain caller-owned. Local documentation proof
is not current hosted CI, production isolation, consumer or release proof.

## Objective Enforcement

The qualified public Tools v2.4.0 binary and immutable reusable CI source
provide the strict tracked-source check through `make source-docs-check`,
the existing local check/CI routes and Required aggregation. Release and
selected security/assurance routes retain their existing controls. This
structural status supplements the rendered semantic audit; it does not
certify prose meaning, generator execution or runtime conformance.
