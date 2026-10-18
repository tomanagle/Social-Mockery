# Social Mockery

A local, deterministic social API simulator. Phase 1 implements a focused subset of Instagram Graph publishing so applications can exercise image and Reel creation, processing, polling, and publishing without contacting Meta.

This is not a complete Meta emulator. Compatibility is based on Posty's `workspaces/libs/platforms/src/instagram/publish.ts` request flow and Meta's official Postman API examples, not a live production verification. Existing applications must point their API requests at this server; their entire test suites are not expected to work unchanged.

## Quick start

Requirements: Go 1.26 or newer, CGO enabled, and a C compiler (SQLite uses `github.com/mattn/go-sqlite3`). On Debian/Ubuntu, install `build-essential`; on macOS, install the Xcode command-line tools.

```sh
make build
./bin/social-mockery start --addr 127.0.0.1:3000 --db :memory: --seed profiles/example.yaml
```

In another terminal:

```sh
curl -fsS http://127.0.0.1:3000/healthz
```

For persisted state, use `--db ./social-mockery.db`. On subsequent starts, omit `--seed` to preserve the virtual clock and records. Explicit `--seed` always replaces existing database state with that profile. See [the API guide](docs/api.md) for copy-and-paste image and Reel flows, and [development](docs/development.md) for layout, tests, and Docker.

## URL rewriting

Keep the original API host and version in the local request path:

```text
https://graph.instagram.com/v25.0/17841400000000001/media
                           ↓
http://127.0.0.1:3000/https://graph.instagram.com/v25.0/17841400000000001/media
```

Supported upstream host labels: `graph.instagram.com` and `graph.facebook.com`. Supported API versions: `v24.0` and `v25.0`. These labels select the local provider route; requests are never forwarded upstream. Supporting the `graph.facebook.com` host does not implement Facebook publishing.

## Deterministic fixtures

The example profile starts at `2026-01-01T00:00:00Z`, with account `17841400000000001` and token `mock-token`. Containers process for five virtual seconds. Real time does not advance processing: use the clock endpoint explicitly.

```sh
curl -fsS -X POST http://127.0.0.1:3000/admin/clock/advance \
  -H 'Content-Type: application/json' -d '{"seconds":5}'
./bin/social-mockery reset --url http://127.0.0.1:3000
```

Reset clears simulator records and restarts deterministic IDs and the virtual clock using the startup profile. Use `reset --seed profiles/example.yaml` to supply a replacement profile for that reset. `expires_at: 0` means a fixture token never expires.

## Local safety

The default listener is loopback-only. There are no real Meta credentials to configure and no media downloads or network proxying. Media URLs are fixture inputs. Set `--admin-token YOUR_LOCAL_SECRET` to protect admin endpoints, and supply that value to the reset command with `--admin-token` or to admin requests as a Bearer token. Keep the server private: this is a development tool, not an Internet-facing service. `--max-records` defaults to `10000` to bound simulator state.

## Phase 1 boundaries

Implemented scope: single-image and single-Reel publishing, container status polling, published-media lookup and listing, fixture tokens, reset, and a controllable persisted clock.

Not implemented: OAuth, app review, analytics, Facebook publishing, carousels, Stories, scenario configuration, webhooks, snapshots, or an SDK. See [compatibility notes](docs/compatibility.md) before integrating.

## License

Licensed under the [MIT License](LICENSE).

## Spec-first development

Provider contracts live in versioned [OpenAPI specifications](specs/meta/instagram/v25.0/openapi.yaml). Run `make generate` to produce Go models and the standard-library server interface; implement behavior in handwritten provider methods. `make generate-check` validates both versions and detects drift. The [provider-spec research skill](.agents/skills/provider-spec-research/SKILL.md) and [workflow guide](docs/specifications.md) cover adding providers without claiming unverified API fidelity.

## Observability

Optional OpenTelemetry tracing, metrics and correlated structured logs are
available over OTLP HTTP/protobuf or gRPC. All exporters are disabled by default.
See [observability setup and guarantees](docs/observability.md) for the local
Collector/Jaeger Compose stack, environment variables, privacy and tests.
