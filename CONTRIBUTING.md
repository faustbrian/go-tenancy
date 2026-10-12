# Contributing

## Before Editing

1. Read [`AGENTS.md`](AGENTS.md) and the affected module's docs.
2. Run the standalone Go baseline below.
3. Identify owned dependencies and reverse dependants in `modules.json`.
4. Preserve unrelated work and generated/corpus provenance.

## Changes

Keep commits focused and conventional. Update every affected changelog with
the behavior and migration impact. Public API changes require compatibility
evidence and documentation. Specification behavior requires a decision record,
fixture coverage, and interoperability evidence.

New direct dependencies and dependency updates must follow the
[dependency governance policy](AGENTS.md#dependencies-and-supply-chain). Package-local
update bots are forbidden; the root policy owns every module and action update.

Specification-backed changes must follow the
[specification governance contract](AGENTS.md#design), update
the affected stable decision entries, and complete the Specification Decisions
section of the pull request template. An unresolved interpretation or stale
source pin is release-blocking; peer behavior cannot silently select policy.

Full scheduled and release mutation gates must finish with zero surviving viable
mutants. Local mutation is optional, not a universal contribution prerequisite.

Do not add package-local workflows, permanent replacements, machine-specific
paths, bypass flags, broad mutation exclusions, or aggregate quality metrics
that hide a failing package.

## Verification

Use the declared Go 1.27.2 producer; the module minimum remains Go 1.27.0.
Test independently of sibling checkouts:

```bash
GOWORK=off go test ./...
```

Ordinary tests run deterministic fuzz seeds and saved corpus regressions, not
active exploration. This untagged command does not exercise live PostgreSQL.
The manifest requires PostgreSQL and selects `integration`, including for
`make check` and `make ci`. Real RLS/pool-isolation proof requires `POSTGRES_URL`
and a principal able to create temporary roles and tables; a skipped live test
is not isolation evidence. See [PostgreSQL guidance](docs/postgresql.md).

With the checksum-pinned public verifier selected by `.golib.yaml`, use:

```bash
make inventory
make check
make ci
```

Both Make routes run bounded local checks, including one race execution of
concurrent graceful drain: all callers join accepted work, intake is refused,
and the owned task context survives until completion then is released. Its
count is one and its timeout is two minutes. Base-free checks run seeds/corpora;
changed-source CI selects bounded active smoke for causally affected targets.
Ordinary CI consumes the public verifier without building it from source.

Short active exploration selects one target; optional local mutation selects
one package:

```bash
GOMAXPROCS=2 GOWORK=off go test -run '^$' -fuzz '^FuzzPropagationExtraction$' -fuzztime=2s -parallel=1 -timeout=2m .
golib mutation --module . --package ./http
```

`make check-full` retains all four native 10000x fuzz campaigns, root benchmarks
and full coverage, race, mutation, API, security and documentation qualification.
Routing, propagation, lifecycle and persistence changes still need their affected
isolation, refusal, cancellation and real-service assertions. Public-consumer
qualification and proxy failover remain explicit service-backed campaigns; see
[hardening guidance](docs/hardening.md). Release rehearsal remains
`golib release dry-run --module .`. Report skipped, unavailable and failed
boundaries; local checks are not hosted-CI or release proof.

## Adding A Module

Follow [repository structure policy](AGENTS.md#repository-structure). New modules
require an explicit purpose, ownership boundary, dependency review, package
catalog entry, full quality gates, documentation, changelog, license, security
policy, compatibility plan, and release dry-run.

## Source Documentation

Install the public `golib` v2.4.0 verifier declared in `.golib.yaml` and
validate its declared checksum before repository checks. Run
`make source-docs-check` for the tracked-source documentation contract.
`make check`, `make check-full` and `make ci` require the same strict check.
Required CI also checks all tracked source, including inactive build variants.
Manual-review findings stay visible; structural success does not certify prose
truth, specification conformance or generator reproducibility.
