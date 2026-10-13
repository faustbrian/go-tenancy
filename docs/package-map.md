# Package map

[Documentation index](README.md) · [Repository entry](../README.md)

This map follows the current source modules, not a synchronized release.
Use the repository README for published installation versions; an unpublished
source major is not installable merely because its package appears here.
Internal packages and non-releasable fixtures are intentionally excluded.

## github.com/faustbrian/go-tenancy/v2

`tenancy` is a small explicit foundation for tenant isolation. It transports validated tenant identity and makes tenant-bound, system-wide, and deliberately unscoped work distinct in Go APIs. It does not authenticate callers, decide membership, or authorize access.

| Import path | Responsibility |
| --- | --- |
| [`github.com/faustbrian/go-tenancy/v2`](../administration.go) | Use for validated tenant identity, explicit scope, propagation, opaque provider namespaces, bounded background work, and administration seams. |
| [`github.com/faustbrian/go-tenancy/v2/http`](../http/http.go) | Use at authenticated HTTP boundaries for trust-gated tenant extraction, acceptance, injection, and middleware. |
| [`github.com/faustbrian/go-tenancy/v2/jsonrpc`](../jsonrpc/jsonrpc.go) | Use at authenticated JSON-RPC boundaries for bounded, duplicate-safe tenant metadata propagation. |
| [`github.com/faustbrian/go-tenancy/v2/postgres`](../postgres/manager.go) | Use for explicit tenant predicates, PostgreSQL row-level-security plans, and transaction-local tenant enforcement. |
| [`github.com/faustbrian/go-tenancy/v2/tenancytest`](../tenancytest/tenancytest.go) | Use in tests for deterministic tenant and system scopes and tenant assertions. |

Choose the next guide in the [documentation index](README.md)
or return to the [repository entry](../README.md) for installation and ownership.
