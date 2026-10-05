# Security policy

The v2 source is governed by the versioned
[threat model](docs/security/threat-model-v2.md), including explicit
application-owned risks and review conditions.

## Supported versions

Pin an exact tenancy module version and review every upgrade. For v1, only
versions explicitly listed in repository release notes are supported. The v2
source tree is not externally consumable until a v2 tag is published.

## Reporting

Report suspected vulnerabilities through GitHub's private security-advisory
workflow. Do not include tenant IDs, customer data, namespace HMAC keys,
credentials, database URLs, or unredacted request and event metadata. Include
the affected version, trust topology, enforcement seams, and a sanitized
reproduction.

## Boundary

This module transports and asserts explicit tenant routing identity. It does
not authenticate callers, authorize tenant membership, grant database roles,
or protect consumers that bypass its context, propagation, namespace, and
persistence seams. Applications own authorization, credential management,
broker and proxy policy, PostgreSQL roles, audit retention, and incident
response.
