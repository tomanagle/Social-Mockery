# Provider specifications and generated Go interfaces

Social Mockery evolves one evidence-backed API slice at a time. An OpenAPI file describes the supported upstream contract; generated Go types and server interfaces connect that contract to a handwritten simulator implementation. It does not imply that the whole provider API is supported.

## Repository layout

- `specs/meta/instagram/v25.0/openapi.yaml` and `specs/meta/instagram/v24.0/openapi.yaml`: versioned Instagram contract slices.
- `configs/openapi/instagram.yaml`: generator configuration.
- `tools/go.mod`: pinned code-generation tooling, separate from runtime dependencies.
- `internal/providers/meta/generated/api.gen.go`: generated Go models and standard-library HTTP server interface.
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

The v24.0 and v25.0 files share the same route/schema subset; generation uses v25.0. The validator also rejects any contract difference other than `info.version`, so a version-specific change requires a separate generated interface/implementation before diverging. Validating both files does not establish independently observed upstream behavior for both versions. The generator invocation is:

```sh
go run -mod=readonly -modfile=tools/go.mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config configs/openapi/instagram.yaml specs/meta/instagram/v25.0/openapi.yaml
```

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
