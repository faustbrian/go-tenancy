# Deprecation Policy

Deprecations MUST identify the replacement, reason, migration steps, and
earliest removal version. Public Go identifiers use a valid `Deprecated:` doc
paragraph and corresponding changelog entry.

At `v1` and later, a supported replacement SHOULD exist for at least one minor
release before removal. Security or correctness defects MAY require faster
removal when continued support would be unsafe; the release notes must explain
the exception.

Silent behavior changes, undocumented aliases, and indefinite deprecated code
are prohibited. Deprecations are checked during compatibility and release
review.

## Active deprecations

### `Group.Close(ctx)`

- **Replacement:** `Group.Drain(ctx)`.
- **Reason:** `Drain` makes the graceful drain-then-release ordering explicit
  and distinguishes it from the cancel-then-wait behavior of `Shutdown`.
- **Migration:** Replace `group.Close(ctx)` with `group.Drain(ctx)`. The
  operation remains source compatible and preserves tenant isolation,
  concurrent and repeated-call behavior, timeout handling, and ordering.
- **Removal horizon:** Removal is not permitted before `v2.0.0`. It also
  requires every identified `Group.Close` consumer to migrate, and must wait
  until the later of 180 days after the first stable release containing this
  deprecation or two subsequent stable minor releases. The reverse-dependency
  inventory names `go-cloudevents/adapters/golib` and
  `go-service/integration/reference-http`; inventory membership does not by
  itself assert that either package calls `Group.Close`.
