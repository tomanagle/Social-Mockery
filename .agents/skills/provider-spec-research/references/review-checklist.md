# Provider contract review

Use this checklist for the requested endpoint slice, not as a claim that the entire provider is supported.

## Evidence and scope

- Each covered operation and material constraint has a source or is explicitly synthetic/unknown.
- Official sources match the chosen API product, login flow, and upstream version. Ownership and source revisions are recorded where available.
- Requiredness and enum values were not inferred solely from example payloads. SDK validation was not mistaken for server enforcement.
- Error envelopes, status codes, retry semantics, pagination, and lifecycle timing are source-backed or labeled simulator choices.
- Provenance distinguishes documentation support from exact live observations. Coverage lists deferred endpoints and unsupported branches.

## OpenAPI and generation

- The spec validates with the repository's pinned validator/toolchain; all references resolve and operation IDs are unique.
- Documented content types, parameter locations, security requirements, request bodies, success responses, and supported error responses survive generation.
- Unknowns are not hidden behind unjustifiably precise schemas. Approximations and unmodeled branches are visible in compatibility notes.
- Generator version/configuration are pinned. Generated source names its origin; a clean regeneration produces no diff, including new untracked generated files.
- No hand-written lookalike interface or custom schema parser replaces the established generator. Generated code is not manually patched.
- Provider structs implement generated interfaces with a compile-time assertion; adapters do not create a second divergent model layer.

## Behavior checks

- Go compile/tests pass for the complete repository, with the required CGO/toolchain configuration.
- HTTP tests exercise actual router/version dispatch and adapters, not only domain methods.
- Tests cover the supported happy path and relevant documented failure cases, parameter/body encoding, omitted versus null fields, and response serialization.
- Synthetic deterministic behavior has tests but is not asserted to be provider-observed behavior.
- If the change affects a stateful lifecycle, tests cover its supported transitions and virtual-clock boundaries.
- Contract conformance checks are meaningful: sample response validity alone does not prove all upstream semantics or complete coverage.

## Safety and delivery

- Fixtures and source exports contain no secrets, real tokens, private identifiers, personal content, or authenticated capture headers.
- Any live-account testing had explicit permission for the specific account and actions; no live mutation was inferred from permission to research docs.
- No runtime crawler, AI dependency, or upstream forwarding was introduced.
- Final report includes commands and actual outcomes, blocked checks, deviations, and next scope. Never relabel skipped checks as passing or documentation-based coverage as live verification.
