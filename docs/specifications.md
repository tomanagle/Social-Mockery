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

`generate-check` checks reproducibility; it is not by itself proof of upstream API compatibility. A successful generation also does not replace OpenAPI validation, HTTP tests, or evidence review. `make spec-validate` uses the pinned `kin-openapi` loader and validator to validate the runtime subset and all contract-only product documents. Report any check that could not run. Changes to generated output should be reviewed together with their source contract and configuration.

## Version policy and isolation

Target the latest officially verified upstream release plus up to two preceding releases, with explicit coverage per version. This is a support policy, not permission to copy a contract and label it compatible. Keep the upstream version in the directory, `info.version`, and fixed OpenAPI route paths. Generate each version into a separate Go package and implement its interface through a version-specific adapter (`internal/providers/meta/versions.go`). Dispatch on the version in the original request URL, never silently upgrade or fall back to another version.

Currently v24.0 and v25.0 have separate contracts, generator configurations, Go models/interfaces and HTTP routers. Their narrow simulator behavior is shared intentionally; neither has a live verification claim. They may evolve independently: validation checks each document rather than requiring equality between releases, and `generate-check` regenerates and compares each package without modifying the working tree. A version adapter is the place for differing parsing, response shapes or semantics; common state operations can remain shared.

As of the 2026-10-18 review, the official [Graph API version table](https://developers.facebook.com/docs/graph-api/changelog/versions/) confirms v26.0 as the latest release (2026-07-29). Reference contracts cover v26.0 plus v25.0/v24.0, which remain supported in the Graph lifecycle table. This does not enable v26.0 Instagram resource endpoints or prove historical behavior. The separate OAuth simulator supports Facebook authorization/code exchange on v24.0–v26.0; resource support remains independently bounded.

Adding a version requires its spec, generator config, generated package, handwritten adapter/dispatch entry, low-cardinality telemetry allowlist entry, HTTP tests, and provenance. The Makefile discovers versioned specs across Meta and TikTok layouts. The reference generator has an explicit provider/product mapping and independent source/server/path validation; adding a provider requires extending those mappings and their regression tests.

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

Use the skill's [review checklist](../.agents/skills/provider-spec-research/references/review-checklist.md) before declaring a slice ready. Report the implemented operations, remaining unknowns, synthetic deviations, and the actual validation/generation/test results. The current simulator includes focused Instagram publishing and synthetic OAuth consent/token flows; broader resource behavior is not implied. This workflow adds no runtime crawling or AI dependency.

## Comprehensive Instagram reference contracts

See [the category audit, exact inventory and explicit boundaries](../specs/meta/instagram/README.md) and [machine-readable coverage](../specs/meta/instagram/coverage.json).

Each of v24.0, v25.0 and v26.0 has five contract-only documents:

- `facebook-login.openapi.yaml`: Graph Instagram and Page-backed messaging, public indexed commerce/creator/Threads-adjacent operations.
- `instagram-login.openapi.yaml`: Instagram Login reads, publishing, moderation and messaging.
- `oauth.openapi.yaml`: source-specific authorization, token and Facebook permission/debug flows.
- `upload.openapi.yaml`: the separate native `rupload.facebook.com` transport.
- `webhooks.openapi.yaml`: application callback verification and event delivery.

`openapi.yaml` is the five-operation runtime subset in v24.0/v25.0 (four publishing operations plus `/me`). Full contracts carry `x-contract-only: true`, operation source links, documentation-based provenance, and `x-simulator-implemented: false`. `internal/contracts/instagram/{v24,v25,v26}/` contains isolated generated models/interfaces; these full generated routers are not wholesale registered by runtime providers or handlers. The OAuth adapters use generated parameter models for their deliberately limited simulated routes. Creating a spec does not create a success stub.

`make spec-validate` discovers both file conventions, validates local references, unique operation IDs, release paths, evidence markers, and approved source/server hosts. OAuth/webhook exceptions are explicit and narrow. Upload paths must embed their directory release and use the dedicated upload host. `scripts/generate-contracts.sh` drives the pinned official `oapi-codegen` models/std-http-server generator; `make generate-check` regenerates into temporary files and compares output without modifying the working tree.

Current primary documentation is now accessible through official product indexes and Markdown references. OAuth, resumable upload and webhook signature semantics are source-backed. Remaining limits include unavailable public marketing-message/opt-in documentation, the nonpublic host in an opt-in sample, an invalid story-insights example, source contradictions, and unverified historical/account-dependent behavior. See the audit rather than treating any version-neutral guide as independent verification of three releases.

`additionalProperties: false` used in evidence-only fixture schemas expresses the reviewed local fixture shape; it is not a provider guarantee that responses will never add fields. Generated interfaces, validation and local tests establish internal consistency, not live compatibility. No live API tests or automatic upstream calls are part of this workflow.


## Facebook and TikTok reference contracts

- [Facebook Pages](../specs/meta/facebook/README.md): v24.0/v25.0/v26.0, each with Pages, OAuth, native Reel upload and webhook callback documents; 37 operations per release. Generated packages are under internal/contracts/facebook/{v24,v25,v26}/.
- [TikTok](../specs/tiktok/README.md): v2 OAuth, Display, Content Posting, upload and webhook callback documents; 16 operations. Generated packages are under internal/contracts/tiktok/v2/. TikTok uses its documented v2 generation; Meta release numbering and invented prior versions do not apply.

Provider markers constrain approved source/host/path combinations. Facebook upload paths retain their Graph release and native rupload host. TikTok upload URLs are opaque returned URLs: only its explicitly marked v2 upload contract may use the bounded dynamic-path exception. Neither exception authorizes runtime proxying or downloads.

Facebook and Instagram Login scope families remain separate. The Facebook route to Instagram uses instagram_basic plus operation-specific grants, Page linkage and access checks, not Instagram Login's instagram_business_basic. See each provider audit for precise token choices, app approvals, Page tasks and unresolved documentation conflicts. Synthetic OAuth behavior is described separately in [OAuth consent](oauth-consent.md).
