# Tenancy threat model v2

Status: active for the v2 source; not a release verification verdict.

Owner: go-tenancy maintainers.

Review condition: changes to identity, scope, propagation, namespace encoding,
background lifetime, administrative iteration, SQL enforcement, or consumers.

## Assets and trust boundaries

Protected assets are tenant isolation, routing identity, namespace separation,
request and task lifetime, administrative attribution, and caller metadata.
Namespace HMAC keys, tenant identifiers, database credentials, and transported
metadata may be sensitive. Ordinary diagnostics must not expose these values.

Tenant identity is routing data, not authentication or authorization. The
application authenticates each transport hop and explicitly supplies trust.
HTTP and JSON-RPC metadata, logical namespace keys, administrative pages,
contexts, and provider envelopes cross caller-controlled boundaries. Provider
clients, SQL callbacks, database roles, task functions, and audit sinks remain
application-owned collaborators.

## Controls and verification boundaries

| Boundary | Control | Evidence to exercise |
| --- | --- | --- |
| Scope and propagation | Validated identity, explicit tenant/system scope, conflicting-scope refusal, explicit transport trust and overwrite refusal | Scope/context and transport tests listed in the isolation matrix |
| Namespace construction | Length-delimited inputs and standard-library HMAC; oversized integration logical keys are refused before input construction | Integration key refusal and allocation characterization; namespace separation tests |
| JSON-RPC metadata | Default 4,096-byte allowance, configurable up to 65,536 bytes; bounded input decoding and refusal when encoded injection output exceeds the configured allowance | JSON-RPC input/output boundary tests and hosted public-entry diagnostic |
| Background work | Group-owned admission and shutdown; cancellation and deadlines propagate to accepted task contexts | Group lifecycle and cancellation tests |
| Administrative traversal | Finite page/tenant policies, sequential audited attempts, explicit resume and cursor-cycle refusal | Administrative boundary, retry and resume tests |
| Persistence | Explicit tenant predicates or scoped transaction manager; restricted roles and forced row-level security remain deployment requirements | SQL/manager tests and applicable hosted PostgreSQL isolation fixtures |
| Diagnostics | Owned identity values redact ordinary formatting and structured logs; namespace output is opaque | Formatting/log tests; manual source and history review |

The detailed test inventory is [the isolation matrix](../security-review.md).
Test names and historical reports are not evidence that the current source,
toolchain, dependencies, release, or consumer integration passed. Current gate
results and unresolved findings remain in the ecosystem execution ledger.

## Accepted collaborator risks

| Risk | Severity and owner | Rationale | Mitigation | Review condition |
| --- | --- | --- | --- | --- |
| SQL callbacks can temporarily alter tenant settings and restore them before final readback | Medium; adopting application's database and authorization owners | A caller-owned `sql.Tx` exposes application SQL; final readback cannot certify every intervening statement | Restricted roles, enforced policies, reviewed SQL interfaces and consumer isolation tests | SQL callback interface, role privileges, policies or readback behavior changes |
| Task and provider callbacks can ignore cancellation or bypass owned seams | Medium; adopting application's integration owners | Go cannot forcibly terminate arbitrary trusted application code or prevent a separately held provider client from bypassing an adapter | Cooperative cancellation, bounded operations and provider timeouts; use owned admission/enforcement seams | Callback lifetime, new provider boundary or context ownership changes |
| Administrative cursors do not cryptographically identify a source snapshot | Medium; adopting application's administration owner | Cursor meaning and durable progress belong to the application source | Stable snapshot cursors, bounded traversal, durable completion/retry and audited attribution | Source snapshot semantics, asynchronous fan-out or retry ownership changes |
| Static analysis covers declared constructors and sinks, not arbitrary dynamic wrappers | Medium; go-tenancy maintainers and adopting application's integration owners | Runtime authorization and isolation cannot be inferred from undeclared dynamic code | Keep the policy inventory current; retain runtime negative tests at each owned boundary | New provider, generated wrapper, reflection path or telemetry sink |

## Release and adoption

The active module uses `/v2` at the repository root; versions are Git tags on
main, not separate source trees. Existing public-v1 consumers remain on v1
until a supported v2 release is published. Follow the
[migration guide](../migration.md) when adopting v2, including allowance for
encoded tenant metadata. Release readiness requires actual current-source CI,
published artifacts, and clean public consumer proof; this document grants no
gate exception and does not claim those boundaries are complete.
