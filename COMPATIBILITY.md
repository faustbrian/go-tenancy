# Compatibility Policy

This repository releases one root Go module and follows semantic versioning.
Root release tags use `v<version>`.

Before `v1`, minor releases MAY contain reviewed breaking changes, but every
break MUST be documented with migration guidance. Patch releases MUST remain
backward compatible. At and after `v1`, incompatible exported API or documented
behavior changes require a new major version.

Compatibility includes exported Go APIs, error classification, serialization,
protocol behavior, persistence schemas, environment variables, command output,
resource ownership, ordering, retry/idempotency semantics, and documented
defaults. A compile-compatible change can still be behaviorally breaking.

The released v1 API is retained in `api/v1-baseline.txt`. The published v2
module is checked against `api/v2-baseline.txt`; the baselines are independent
because v2 intentionally changes JSON-RPC injection behavior. Both published
major lines retain their own compatibility contracts.

Specification-backed modules MUST NOT diverge from their declared standards.
Ambiguities require documented decisions and stable tests. Deprecated APIs
follow [`DEPRECATION.md`](DEPRECATION.md).
