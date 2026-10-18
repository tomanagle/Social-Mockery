# Provider specifications and generated Go interfaces

Social Mockery evolves one evidence-backed API slice at a time. An OpenAPI file describes the supported upstream contract; generated Go types and server interfaces connect that contract to a handwritten simulator implementation. It does not imply that the whole provider API is supported.

## Repository layout

- `specs/meta/instagram/v25.0/openapi.yaml` and `specs/meta/instagram/v24.0/openapi.yaml`: versioned Instagram contract slices.
- `configs/openapi/instagram-v24.0.yaml` and `configs/openapi/instagram-v25.0.yaml`: generator configuration.
- `tools/go.mod`: pinned code-generation tooling, separate from runtime dependencies.
- `internal/providers/meta/generated/v24/api.gen.go` and `internal/providers/meta/generated/v25/api.gen.go`: generated Go models and standard-library HTTP server interface.
- `internal/providers/meta/`: handwritten provider implementation and HTTP adaptation.
- `.agents/skills/provider-spec-research/SKILL.md`: reusable agent workflow for researching and extending provider contracts.

The generator is `oapi-codegen` v2.8.0, pinned in `tools/go.mod` and configured for Go models and a standard-library HTTP server interface. A provider struct implements the generated `ServerInterface`; a compile-time assertion should make interface drift a build failure. The local rewritten URL includes an upstream host label. The HTTP wrapper validates that label and handles local routing before handing the path to the generated router. The server must never forward requests upstream.

Do not edit generated Go files. Change the contract or generator configuration and regenerate:

```sh
make spec-validate
make generate
make generate-check
make test
```

`generate-check` checks reproducibility; it is not by itself proof of upstream API compatibility. A successful generation also does not replace OpenAPI validation, HTTP tests, or evidence review. `make spec-validate` uses the pinned `kin-openapi` loader and validator to validate both versioned contracts. Report any check that could not run. Changes to generated output should be reviewed together with their source contract and configuration.

## Version policy and isolation

Target the latest officially verified upstream release plus up to two preceding releases, with explicit coverage per version. This is a support policy, not permission to copy a contract and label it compatible. Keep the upstream version in the directory, `info.version`, and fixed OpenAPI route paths. Generate each version into a separate Go package and implement its interface through a version-specific adapter (`internal/providers/meta/versions.go`). Dispatch on the version in the original request URL, never silently upgrade or fall back to another version.

Currently v24.0 and v25.0 have separate contracts, generator configurations, Go models/interfaces and HTTP routers. Their narrow simulator behavior is shared intentionally; neither has a live verification claim. They may evolve independently: validation checks each document rather than requiring equality between releases, and `generate-check` regenerates and compares each package without modifying the working tree. A version adapter is the place for differing parsing, response shapes or semantics; common state operations can remain shared.

As of the 2026-10-18 review, Meta's official version table and changelog returned HTTP 429 or were inaccessible. We therefore do **not** label v25.0 as the latest release, or claim that the latest-plus-two target is already met. A new release (including v26.0) must pass the research checklist before it is enabled. Unknown versions return an explicit unsupported-version error. See the per-version provenance files for coverage and remaining evidence gaps.

Adding a version requires its spec, generator config, generated package, handwritten adapter/dispatch entry, low-cardinality telemetry allowlist entry, HTTP tests, and provenance. The Makefile discovers versioned specs and Instagram generator configurations. Adding other providers also requires extending the generation mapping; it is deliberately not a generic provider framework yet.

## Researching an endpoint slice

Use the `provider-spec-research` skill when adding a provider/version, extending an operation, or auditing source provenance. Start with a bounded client workflow, such as image creation, container status polling, and publishing. Prefer official provider documentation, provider-owned Postman collections, and pinned official SDK sources. Client application code identifies the calls an integration needs; it does not establish the provider's full behavior.

For each source, record its URL and relevant section, retrieval date, API product and version, and revision or export checksum when available. Keep a provenance record next to the contract using the skill's [template](../.agents/skills/provider-spec-research/assets/provenance-template.md). Maintain an endpoint coverage matrix that shows supported and unsupported branches. A copied contract under a second version requires an explicit account of which behavior is evidenced for that version and which remains uncertain.

Do not infer schema requiredness or exhaustive enums from one Postman response. Do not invent error codes, permissions, retry rules, rate limits, or lifecycle timings. If sources disagree, preserve the uncertainty, narrow support, or use a permissive schema where appropriate. Record simulator-specific choices separately.

## Evidence labels

- **Documentation-based:** supported by cited official documentation, collection, or source; not verified by a matching live account observation.
- **Live-verified:** a particular behavior observed on a recorded date against the named API version with an explicitly authorized account. The observation does not verify unrelated schema branches or all account configurations.
- **Synthetic:** invented fixture data or deliberate simulator behavior, such as deterministic IDs and virtual-clock processing delays. It is not evidence of upstream semantics.

These labels apply to individual claims, not just a whole file. A synthetic token and ID can be used in a fixture testing a documentation-based response schema. That does not make the fixture a live capture.

## Safety and completion

Public documentation research does not authorize live API requests. Obtain explicit approval for the account and specific live test operations, including any publishing, deletion, media upload, permission change, or charge. Sanitize permitted captures before using them as fixtures. Never commit real credentials, raw tokens, private account identifiers, or personal content. Prefer synthetic fixtures.

Use the skill's [review checklist](../.agents/skills/provider-spec-research/references/review-checklist.md) before declaring a slice ready. Report the implemented operations, remaining unknowns, synthetic deviations, and the actual validation/generation/test results. The current simulator is a focused Instagram subset; broader provider coverage and generated client SDKs are future work. This workflow adds no runtime crawling or AI dependency.

## Broad Instagram research contracts

See [the complete inventory and explicit gaps](../specs/meta/instagram/README.md).
The versioned `facebook-login.openapi.yaml`, `instagram-login.openapi.yaml` and
`webhooks.openapi.yaml` documents describe a much broader, **contract-only** surface.
They are intentionally separate from `openapi.yaml`, which describes the implemented
simulator subset. `internal/contracts/instagram/` contains their generated types and
server interfaces and is never registered in the runtime router.

The broader surface is not yet a complete verified Instagram specification. OAuth,
upload transport, release-specific permissions and other listed gaps require accessible
primary documentation or separately authorized live captures. v26.0 is a pinned-SDK
research candidate, not an assertion of latest-version or runtime support.
