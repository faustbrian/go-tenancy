# Hardening evidence

The executable suite covers identity normalization and hostile bytes; scope
construction and conflicts; cancellation and deadlines; HTTP and JSON-RPC
duplicates, spoofing, and input and encoded-output size limits; every
integration domain; pre-allocation namespace bounds; namespace separation;
bounded asynchronous close and shutdown; administrative resume and partial
failure; PostgreSQL pool reuse, rollback, readback, reset failure, and RLS
plans; randomized multi-tenant models; races; and fuzz targets.

Integration-domain evidence covers the owned `Integration` contract and its
state model. The clean-consumer gate additionally composes application-owned
adapters with the first-party cache backend, bounded search contract provider,
queue/CloudEvents conversion, workflow values, audit store, and OpenTelemetry
SDK. The OpenSearch matrix executes the same scoped search adapter against
digest-pinned OpenSearch 2.19.6 and 3.8.0 images with identical logical index
and document IDs for two tenants. The Redis Streams fixture uses opaque tenant
queue names and proves the same validated identity survives broker reclaim,
retry, and terminal dead-letter persistence while another tenant's stream
remains empty. Provider paths not declared by those fixtures remain outside the
claim.

The build-tagged PostgreSQL test adds a restricted application login, forced
and restrictive RLS, cross-tenant reads and mutations, alternating prepared
plans, rollback, cancellation, stale state, backend replacement, and concurrent
pool proof. It is valid only when run against a real PostgreSQL service with
`POSTGRES_URL`; the same gate reopens a first-party workflow store across a
tenant-bound retry and persists/query-filters first-party audit records from
context-derived tenant scope. The failover lane streams a pinned PostgreSQL 18
primary into a second host, holds a tenant transaction through a switching TCP
proxy, stops the primary, promotes the replica, and proves the failed
transaction is not reused while the same `database/sql` pool reconnects with
fresh tenant and unscoped state. A skipped run is not interoperability evidence.

`make check` and `make ci` run the checksum-pinned public verifier's bounded
local route, without building it from source. Ordinary deterministic tests
include fuzz seeds and saved corpus regressions; active changed-target smoke
is distinct from the four full 10000x exploration campaigns. One root race,
count one with a two-minute bound, selects concurrent graceful drain and proves
joined completion, intake refusal and release of the owned task context.
The manifest's PostgreSQL prerequisite and `integration` tag still apply to
these Make routes; unconfigured or untagged Go tests do not prove live RLS.

`make check-full` retains the original full gates and root benchmarks. Coverage
and full scheduled/release mutation gates operate per production package;
optional local mutation selects one package and is not a contribution prerequisite.
Exact statement
coverage does not by itself prove meaningful assertions, and mutation results
do not cover equivalent or invalid mutants without a reviewed record. Fuzzing
is bounded in CI and should be supplemented by longer scheduled runs. Race
testing proves only executed schedules. No evidence applies to consumer paths
that bypass explicit scope, namespace, transaction, or propagation seams.

Published-consumer compositions remain an explicit `public_consumers` manual
workflow with PostgreSQL, Redis and OpenSearch, pinned published v2.0.0 runtime
source parity, and all existing scripts and refusal checks. Current verifier
policy is not part of that runtime/API/dependency/specification comparison.
The separate development-only security diagnostic uses immutable qualified
tooling source with Go 1.27.2; it does not replace release CI or qualify Required.
PostgreSQL proxy failover is the explicit Docker-backed
`bash scripts/test-postgres-failover.sh` campaign. Release rehearsal remains
`golib release dry-run --module .`; see
[contributing](../CONTRIBUTING.md#verification) for focused commands.

Benchmark results must report Go version, OS, architecture, CPU, benchtime,
latency, and allocations. Compare only identical behavior and corpus inputs;
the repository stores benchmark output as gate evidence rather than claiming a
universal performance threshold.

The external administrative fixture uses a synchronized, fsync-before-rename
journal, bounded fan-out, fresh tenant contexts, stable operation identities,
and per-tenant actor, purpose, reference, attempt, and completion records. It
reopens the journal after partial failure and proves completed tenants are not
repeated while failed imports resume; a separate migration operation remains
tenant-attributed. This proves the documented application-owned pattern, not a
cluster-wide ledger implementation supplied by `tenancy`.
