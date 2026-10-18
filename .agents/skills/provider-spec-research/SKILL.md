---
name: provider-spec-research
description: Research a social provider's official API documentation, Postman collections, and SDK sources to build or extend versioned OpenAPI contracts and generated Go server interfaces for Social Mockery. Use for adding provider endpoints or checking contract provenance and compatibility.
---

# Provider spec research

Turn a bounded set of provider operations into an evidence-backed OpenAPI contract, reproducibly generated Go types/interfaces, and a simulator implementation. This is an authoring workflow: do not add runtime crawling, AI, or upstream API dependencies to the simulator.

## Scope and evidence

1. Inspect repository instructions, existing provider contracts, generator configuration, adapters, and compatibility notes. Agree on provider, API product/login flow, upstream version, operations, and acceptance tests. Extend a small useful slice; do not imply that one working endpoint completes a provider.
2. Read official documentation first, then provider-owned Postman collections and official SDK source. Verify ownership through an official site or repository. Pin collection exports and SDK references to a revision when possible. Application call sites can establish required client coverage, but cannot establish provider semantics.
3. Track evidence per operation and material behavior using [the provenance template](assets/provenance-template.md). Record exact source URL/section, provider version, retrieval date, source revision if available, covered fields/status codes, and unresolved questions. Distinguish:
   - **documentation-based**: a claim supported by cited provider documentation or source, without a matching live observation.
   - **live-verified**: the specific request/response behavior was observed against the named version and authorized account on the recorded date. This does not verify every schema branch or make behavior universal.
   - **synthetic**: a simulator choice or invented fixture value, clearly identified as such. Never present this as evidence of upstream behavior.
4. Where sources conflict, record the conflict and prefer evidence specific to the chosen product/version. Do not silently merge Facebook Login and Instagram Login APIs, SDK versions, or different providers. Treat collection examples as examples, not exhaustive schemas or proof of required fields. Treat SDK defaults and client validation as client behavior unless authoritative server documentation confirms them.

## Contract and Go implementation

- Keep contracts versioned by provider and upstream API version. Use the repository's layout and generator configuration. Separate the upstream contract from Social Mockery's rewritten URL routing and admin API.
- Select an OpenAPI version supported by the pinned generator. Specify unique stable operation IDs, path/query/body encodings, documented authentication, request/response schemas and supported response statuses. Model nullable, omitted, and required fields separately when the evidence supports the distinction.
- Do not fabricate required properties, enum exhaustiveness, limits, error codes, retry rules, permissions, or error semantics. Leave uncertain fields permissive or narrow the supported scope; state the limitation in provenance. A synthetic fallback error must be labeled as simulator behavior rather than as a documented upstream response.
- Preserve polymorphism and differing request encodings only where verified by sources and supported by the toolchain. Validate local references; avoid mutable remote references in a reproducible build. Explain any intentional approximation instead of forcing invented precision merely to satisfy code generation.
- Use a pinned established OpenAPI generator, preferably the repository's existing one, for Go interfaces and types. Do not write a bespoke schema parser or hand-maintain a parallel interface. Commit generator configuration and a repeatable generation/check command alongside generated output according to repository conventions.
- Implement generated interfaces with provider structs and methods. Add a compile-time interface assertion. Put simulator state transitions in the simulator/domain layer; keep version or HTTP translation in adapters. Generated output is disposable: change the spec/configuration and regenerate rather than editing it directly.
- Target the latest release verified in the official provider version table plus up to two preceding releases. Record the verification date and source. Never infer "latest" from a client default, search snippet, or copied contract; if official evidence is unavailable, mark the target pending. Keep fixed versioned paths and `info.version` aligned with the directory. Generate a separate package and interface per version, add an explicit adapter/dispatch entry, and test that other/unknown versions cannot reach it. Versions may differ; never require cross-version contract equality.
- When a version changes, compare the old and new contracts, identify source-backed changes, and test version dispatch. Do not duplicate an older contract under a new version and claim independent verification.

## Repository commands

The current Instagram slice uses `specs/meta/instagram/{v24.0,v25.0}/openapi.yaml`, separate `configs/openapi/instagram-v*.yaml` configurations, and independent `internal/providers/meta/generated/v*/api.gen.go` packages. Tool versions are pinned in `tools/go.mod`. Run `make spec-validate`, `make generate`, `make generate-check`, and `make test`. Inspect these targets before adding a provider so new contracts and generated files are included in validation and reproducibility checks. See [the specification guide](../../../docs/specifications.md) for the current generator invocation.

## Verification and delivery

Read [the review checklist](references/review-checklist.md) before declaring the slice ready. Run the repository's OpenAPI validation and generation check, inspect the generated diff, compile/test all Go packages, and exercise the implemented operations through HTTP. Generation must be deterministic and a fresh regeneration must produce no diff. If a tool cannot run, report that check as blocked rather than passed.

Live account tests require explicit permission for the account, API operations, and any mutations or charges. Public documentation research never grants permission to publish, delete, upload media, request new scopes, or probe live rate limits. Use sanitized synthetic fixtures by default; never commit real credentials, tokens, private account IDs, personal content, or raw authenticated captures. Record a live result without committing secrets or identifying account data.

Deliver the versioned contract, provenance with an endpoint coverage matrix, pinned generator configuration and generated Go output, implementing adapter, tests, and exact check results. State supported operations, unsupported branches, synthetic behavior, remaining unknowns, and whether anything was live-tested. Report partial coverage honestly; do not broaden the task to every provider or publish changes without authorization.
