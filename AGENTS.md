# Social Mockery contributor guide

- Use `.agents/skills/provider-spec-research/SKILL.md` before adding a provider or expanding a provider contract.
- Edit versioned specifications first. Run `make spec-validate`, `make generate`, and `make generate-check`. Never hand-edit `*.gen.go`.
- Keep provider behavior in handwritten Go methods that satisfy generated interfaces. Use the standard library for routing, HTTP, logging, flags and tests. Add dependencies only when their benefit justifies them.
- Preserve the GOTTH-style `cmd` / `internal/config` / `internal/handlers` / `internal/store` / `internal/store/db` / `internal/store/dbstore` package boundaries.
- The simulator must never forward provider requests or download user-supplied media URLs. Live contract research belongs in separately authorized tooling, never the runtime or default tests.
- Label behavior as live-verified, documentation-based or synthetic. Do not present unsupported endpoints or stub implementations as completed features.
- Keep telemetry optional, bounded and free of tokens, raw request bodies, URLs, account identifiers and personally identifying data.
- Run `make fmt-check`, `make generate-check`, `go vet ./...`, `go test -race ./...`, and `make build` before publishing changes. Docker and live-platform checks are separate and must be reported accurately.
