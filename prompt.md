# Social Mockery: project brief and implementation handoff

Use this document as the working prompt for continuing Social Mockery. It records the intended product, the verified implementation boundary, and the remaining work. Read the code before starting; do not rebuild completed features or treat generated interfaces as functioning endpoints.

Status baseline: inspected `main` at `59b29a1e70e1fc674a90f0079fbeaa10c529d1e5` on 2026-10-19. This is a snapshot, not a promise that later code matches it. Update the status and acceptance evidence as work lands.

Status updates: 2026-10-19, Instagram Login long-lived token exchange/refresh, `/me`, and unversioned Graph calls through the app's `graph_api_version` (section 4). 2026-10-28, Docker documentation mismatches reconciled (section 6B).

## 1. What we are building

Build a lightweight, local social-platform simulator for developing and testing integrations without real social accounts, uploads, posts, or provider API calls. The immediate use case is testing Posty, starting with Instagram publishing and realistic connection/consent flows; Facebook Pages and TikTok resource workflows follow. The longer-term goal includes deterministic influencer/account profiles, engagement, failures, rate limits, webhooks, and reproducible scenarios.

A developer should be able to start one Go executable, load synthetic fixtures, point an integration's provider URLs at it, connect a seeded account, publish, advance time, and assert provider-shaped results. The simulator should be useful both interactively and in fast, isolated CI tests.

Provider-shaped does not mean claiming complete or certified provider parity. Preserve actual product/version differences and explicitly label approximations. The simulation controls belong in a separate admin API; do not add mock-only query parameters or fields to native provider requests.

## 2. Non-negotiable implementation requirements

- One small Go server executable. Running it starts the server; configuration uses flags/environment. No `start`, `seed`, `reset`, or other CLI subcommands. Use admin HTTP endpoints for controls.
- Standard-library HTTP routing, flags, `log/slog`, and `testing`. Keep dependencies justified. SQLite through `database/sql`, not GORM; in-memory by default and optional file persistence.
- Preserve the GOTTH-inspired package boundaries: `cmd/main.go`, `internal/config`, `internal/handlers`, `internal/middleware`, `internal/store`, `internal/store/db`, `internal/store/dbstore`, `internal/providers`, `internal/profiles`, and `internal/simulation`. Existing `internal/app` and `internal/telemetry` handle composition/lifecycle and observability.
- Constructor dependency injection. Required startup dependencies have error-returning constructors plus `Must` variants. Explicit production panics belong only in `Must` constructors, called directly from `main`; runtime failures return errors. Panic messages must not reveal secrets or raw dependency errors. Optional telemetry failure is nonfatal.
- Prefix original provider URLs with the simulator origin, retaining the native host/path/query. Example: `https://graph.instagram.com/v25.0/ACCOUNT/media` becomes `http://127.0.0.1:3000/https://graph.instagram.com/v25.0/ACCOUNT/media`.
- This is an allowlisted routing envelope, never a proxy. Never forward provider traffic or fetch supplied media URLs. Unknown hosts, versions, and operations must not silently fall through to another provider/version.
- Use seeded synthetic data, a controlled virtual clock, repeatable IDs and deterministic simulation. Keep storage, queues, concurrency, and computational work bounded. Do not create millions of follower/event rows or one goroutine per follower/post.
- Preserve native authentication, parameter encodings, field projection, error shapes, pagination, token classes, and lifecycle differences where implemented and evidenced. Never invent precision merely to make a schema look complete.
- Keep requested scopes, app-approved capabilities, user-granted scopes, and selected/eligible assets distinct. Reconnecting must not silently grant every scope or reset existing posts.
- Use fake fixture credentials only. The UI must never ask for real provider passwords. Keep the default listener on loopback and support admin authentication.
- Dockerfiles and Compose configuration live in `dev/`; maintain Make targets, optional host Air workflow, and CI. Keep the main README short, usable, and checkbox-oriented; detailed engineering scope belongs here and in focused documentation.
- Use official optional OpenTelemetry SDKs/exporters with standard environment configuration, OTLP HTTP/gRPC, W3C propagation and contextual slog correlation. No placeholder spans implying unfinished features ran. No secrets, payloads, raw URLs, account IDs, or unbounded metric dimensions.

## 3. Read these before changing code

1. [Contributor instructions](AGENTS.md) and [provider research skill](.agents/skills/provider-spec-research/SKILL.md).
2. [README](README.md), [development](docs/development.md), [API examples](docs/api.md), and [compatibility boundaries](docs/compatibility.md).
3. [OAuth consent behavior](docs/oauth-consent.md), [specification workflow](docs/specifications.md), and [observability](docs/observability.md).
4. Runtime routing in `internal/handlers/http.go`, Meta adapters in `internal/providers/meta/`, OAuth adapters in `internal/providers/oauth/`, domain logic in `internal/simulation/`, and SQLite repositories in `internal/store/dbstore/`.
5. Provider contract inventories and provenance: [Instagram](specs/meta/instagram/README.md), [Facebook](specs/meta/facebook/README.md), [TikTok](specs/tiktok/README.md).
6. [Makefile](Makefile), [CI](.github/workflows/ci.yml), `tools/go.mod`, generator scripts/configuration, and relevant tests.

When documentation and implementation differ, investigate and resolve the specific mismatch; do not use this handoff as evidence of upstream behavior.

## 4. What is implemented

### Runnable server and state

- [x] Go HTTP server, health endpoint, flags, constructor-based composition, graceful lifecycle, and structured logging.
- [x] Real SQLite schema/repositories, isolated in-memory databases, optional file database, and restart persistence.
- [x] YAML/JSON seed loading, validation, built-in example fixtures, deterministic IDs, virtual time and container processing delay.
- [x] `GET /healthz`, `GET /admin/clock`, `POST /admin/clock/advance`, `POST /admin/reset`, and `POST /admin/oauth/fixtures`.
- [x] Optional admin bearer token, rejection of browser-Origin administrative requests, input limits, request-concurrency bound, and configurable combined container/post capacity (`--max-records`, default 10000). These are local safety bounds, not provider rate-limit simulation.
- [x] Reset restores startup fixtures or a supplied complete profile. Explicit startup `--seed` replaces persisted state; restarting without it preserves an existing database. Clock does not advance with wall time.
- [x] OAuth fixture upsert preserves posts/tokens and does not import the supplied profile's clock, processing configuration, or tokens. Omitted records are not deleted. It is not a general scenario/profile-management API.

### Instagram resource runtime

- [x] Separate generated runtime contracts/adapters for Graph v24.0 and v25.0, reached via recognized `graph.instagram.com` / `graph.facebook.com` routing labels.
- [x] Five method/path operations per version: create media, publish media, shared object read, list media, and `/me`. Container and published-media reads share the object-read route.
- [x] `GET /me` on `graph.instagram.com` with `id` (synthetic Instagram-scoped ID), `user_id`, `username`, documented `account_type` values and `media_count`; documented fields without fixture data are omitted.
- [x] Unversioned Graph calls starting with `me` or a numeric node resolve through the token's app `graph_api_version`, per Meta's documented app-dashboard default.
- [x] Single-image and Reel URL-input container creation; virtual processing; status reads; publish with `creation_id`; published ID/permalink reads; account-media listing.
- [x] Token/account/scope checks, expiry, ownership, processing and publish-state checks, field selection/pagination behavior within the documented slice, and deterministic reset.
- [x] OAuth-issued Instagram tokens connect to this resource slice, including linked-account authorization for Instagram through Facebook Login. Legacy fixture tokens remain a separate compatibility shortcut.

Not implemented here: Instagram v26 resource routing, carousel/Stories, real uploads/transcoding, full account discovery, insights, comments, messaging, webhook delivery, or every endpoint in the reference contracts.

### OAuth consent runtime

- [x] Local seeded-account consent flow for four distinct products: Instagram Login, Instagram through Facebook Login, Facebook Login, and TikTok Login.
- [x] Provider-native authorize/token route shapes; Instagram unversioned OAuth, Facebook v24.0/v25.0/v26.0 dialog/code exchange, and TikTok v2 authorize/code exchange/refresh.
- [x] Account choice, requested permission display, approve/deny and partial consent, eligible Page selection, app-approved scopes, persisted grants, exact registered redirect validation, returned state, expiring/single-use codes, and token issuance.
- [x] Product-specific scope families and selected-asset/task checks. Instagram through Facebook Login requires one selected Page linked to an eligible professional account; single-Page selection is a current simulator limitation.
- [x] Instagram Login authorize accepts Posty's `www.instagram.com/oauth/authorize/` form and the code exchange returns a flat object, both owner-reported live behaviour (2026-10-19) that conflicts with Meta's guide.
- [x] Instagram Login long-lived token exchange (`ig_exchange_token`) and refresh (`ig_refresh_token`) on unversioned `graph.instagram.com` paths, with the documented 60-day lifetime, 24-hour refresh age and `instagram_business_basic` requirement. Exact `expires_in`, idempotent exchange and in-place refresh are synthetic.
- [x] TikTok persistent atomic access/refresh rotation, with prior tokens invalidated and original refresh expiry retained. This exact rotation policy is synthetic.
- [x] PKCE where configured: TikTok desktop uses its documented hexadecimal S256 representation; optional Meta simulator PKCE uses base64url. TikTok web PKCE is explicitly unsupported by this simulator.
- [x] Browser-bound consent sessions, separate tab cookies, escaped HTML, no-store/no-referrer/frame restrictions, atomic session/code consumption, replay protection, and bounded persisted sessions/codes.
- [x] Upsert/reconnect preserves posts. Reducing app approval or grants affects subsequent OAuth-issued Instagram authorization; legacy fixture-token semantics stay separate.

Not implemented: real identity/password login, full Business Login `config_id` behavior, system-user flows, actual App Review/Business Verification, provider permission/revocation/debug endpoints, QR login, Facebook long-lived exchange/refresh, Facebook Page or TikTok resource APIs. Code and token lifetimes include documented and synthetic choices; see the OAuth guide rather than generalizing one product's behavior to another.

### Observability and engineering foundation

- [x] Official optional tracing, metrics and OTLP log export; HTTP/protobuf and gRPC; independent signals and no-op defaults; standard `OTEL_*` configuration.
- [x] W3C trace propagation, request/provider dispatch/authentication, profile/admin operations, SQLite repository operations, and publishing instrumentation; contextual slog trace/span correlation.
- [x] Bounded queues/metric cardinality, sanitized telemetry, asynchronous export, backend-failure isolation, shutdown flushing, and tests with in-memory/local exporters.
- [x] Make build/dev/test/race/vet/fmt/benchmark targets, host Air configuration, dev/production Docker assets, and GitHub Actions checks.
- [x] Unit and table-driven HTTP/API integration tests using real SQLite, persistence/migration/reset/authorization/security tests, concurrent/race coverage, benchmark smoke checks, and contract-generator validation tests. Extend these for every new feature; their existence does not establish full browser, SDK, or Posty end-to-end coverage.

## 5. Reference contracts are a separate deliverable

The repository already contains substantial versioned OpenAPI reference specifications, provenance/source inventories, coverage manifests, and generated Go models/server interfaces. They are not all mounted runtime handlers.

Measured directly from the current reference documents:

- Instagram: five product documents per Graph version v24.0, v25.0, v26.0; 105 operations per version (60 Facebook Login, 29 Instagram Login, 13 OAuth, 1 upload, 2 webhook callbacks). Total 315 versioned entries.
- Facebook Pages: four product documents per version v24.0, v25.0, v26.0; 37 operations per version (28 Pages, 6 OAuth, 1 upload, 2 webhook callbacks). Total 111 versioned entries.
- TikTok v2: five documents; 16 operations (5 OAuth, 3 Display, 6 Content Posting, 1 upload transport, 1 webhook receiver).
- The separate narrow Instagram runtime specifications contain five operations per v24.0/v25.0. OAuth runtime uses selected reference interfaces through handwritten adapters; it does not mount every reference method.

Reference coverage includes substantially broader publishing, account/discovery, comments, insights, messaging, native uploads and callbacks where the selected product exposes them. Consult each inventory for exact inclusions/exclusions. A receiver template is not an upstream API endpoint, and repeated version entries are not distinct capabilities.

Preserve the existing provenance model:

- **Documentation-based:** supported by cited official sources, not a live observation.
- **Live-verified:** only the particular authorized behavior/version/account configuration observed, with date and sanitized evidence.
- **Synthetic:** deliberate simulation choices or fixture data.

The current work has no live-provider verification claim. Current documentation represented across older release directories is not independent historical verification. Recheck official release/version evidence before adding support; do not freeze a historical “latest” claim indefinitely.

Use the pinned established OpenAPI generator (`oapi-codegen` in the tools module), never a home-grown interface generator or hand-edited `*.gen.go`. Change contracts/configuration, regenerate, validate, and implement interfaces with compile-time assertions and explicit version adapters. Keep runtime-subset specifications separate from reference-only coverage until actual handlers and tests exist.

## 6. Remaining work, in a useful execution order

These are open deliverables, not completed capabilities or an instruction to implement everything in one change. Choose a bounded vertical slice, inspect current code again, and finish its tests/documentation before expanding scope.

### A. Make the existing slice demonstrably useful for Posty

- [ ] Inspect the current Posty integration call sites and identify the precise routing/configuration hook for authorization, token exchange, media creation/polling/publish/read. Client call sites define required coverage, not provider semantics.
- [ ] Add a reproducible integration harness and synthetic fixtures demonstrating actual Posty calls against Social Mockery, using permitted repositories only. Record exact supported client revision and behavior.
- [ ] Support deterministic delayed processing via test-driven clock advancement, plus `processing_seconds: 0` for unmodified polling loops. Do not silently add wall-clock progression.
- [ ] Cover reconnect with existing posts, denied/partial scopes, wrong account or linked Page, expired token, processing not ready, and publish failure.

Acceptance: an automated local end-to-end test demonstrates connection and image/Reel publication, verifies stored output, and records that no provider network traffic occurred. Do not claim Posty integration complete based only on simulator curl/tests.

### B. Finish validating the current user/developer experience

- [ ] Run real browser QA for all four consent products: initial selection, denial, partial consent, Page eligibility, callback/state, multiple tabs, repeated submission, Back/Forward and expired sessions. Review visual/accessibility behavior and label UI approximations.
- [ ] Run Docker development and production builds, persistent-volume restart, and optional collector/Jaeger smoke checks in an environment with Docker.
- [x] Reconcile known documentation mismatches: the persistent Docker example in `docs/development.md` no longer passes a `start` argument (the binary accepts no subcommands), and the README no longer claims Docker live reload (development Compose uses `go run` and requires restart; Air is host-side). Fixed 2026-10-28: the corrected persistent `docker run` command was executed locally against the production image and kept the advanced clock across a container restart; `make dev` was not rerun.
- [ ] Keep short README checkboxes synchronized with actual runtime support; remove obsolete internal-phase wording from user docs when touched.

Acceptance: executable documented commands match the binary; browser/Docker/collector results are recorded separately from Go test results. A blocked visual or container check stays explicitly unverified.

### C. Typed fixtures, account profiles, and scenario controls

- [ ] Evolve basic seeds into validated, versioned, provider-specific typed fixture settings plus common simulation-profile/seed/scenario configuration. Preserve existing seed compatibility or document a migration.
- [ ] Represent account capabilities, app/product approvals, grants, Page/task/linked-account eligibility, creator options and provider-specific restrictions separately. Do not put every provider into one permissive universal scope/account model.
- [ ] Add named reusable profile/scenario configuration and admin controls without changing native API parameters. Define merge/upsert/reset semantics explicitly so configuration edits do not erase posts or silently grant permissions.
- [ ] Add bounded deterministic random-state handling, reproducibility across restart/reset, and safe validation of malformed or oversized profiles.

Acceptance: the same seed/profile/scenario/clock inputs reproduce results; a reconnect or fixture update preserves unrelated posts; invalid updates are atomic; provider-specific invalid combinations are rejected without touching state.

### D. Expand provider resource workflows

- [ ] Implement the next Instagram resource/version slices required by clients, including an explicit v26 adapter if needed. Add carousels, Stories, discovery, comments/moderation, insights or messaging only as separately scoped source-backed features.
- [ ] Implement Facebook Page discovery and User/Page token/task authorization, then text/link/photo publishing and read/list flows; add video/Reel upload/processing/publish lifecycle as a later bounded slice.
- [ ] Implement TikTok Display and creator-info, then Content Posting init/status and provider-specific direct versus inbox workflows. Respect chosen privacy, granted product scopes, audit/account restrictions, moderation and completion differences.
- [ ] If native upload transport is implemented, emulate returned upload URLs and chunk/state validation locally, with strict byte/storage bounds. Never use the returned host as permission to proxy upstream or fetch user media.
- [ ] Add supported token lifecycle/permission/revocation operations when client workflows require them. Revocation must affect eligible tokens/grants; it must not erase unrelated resources.

Acceptance per slice: correct native method/path/encoding; generated interface satisfied; explicit version dispatch; success and negative authorization/state tests; field/pagination/error behavior covered; unsupported operations stay visibly unsupported; provenance and compatibility matrix updated. Facebook/TikTok login alone does not satisfy their publishing deliverable.

### E. Deterministic influencer and engagement simulation

- [ ] Add configurable account/audience profile inputs and post-performance behavior. Define documented synthetic distributions for growth, reach/views, likes, comments and other supported metrics rather than claiming a provider prediction model.
- [ ] Compute aggregate counters/time buckets from seeded randomness and virtual time. Materialize only bounded sample objects where API reads need them; never enumerate each follower or engagement event.
- [ ] Persist only the minimal state required for reproducibility. Define ordering, overflow, retention and repeat-query semantics; repeated reads at the same time must not accidentally create engagement.
- [ ] Connect aggregate state to implemented provider analytics/read surfaces with provider-specific fields and availability rules.

Acceptance: identical inputs/time produce identical outputs across reset/restart; counters are internally consistent; very large synthetic audiences do not cause proportional memory, rows, goroutines or event queues; benchmarks report actual time/allocations under representative loads.

### F. Scenarios, failures, and realistic rate-limit behavior

- [ ] Add named success/failure scenarios and bounded admin-configured error injection: processing failures, expired/revoked tokens, missing grants/assets, intermittent failures and relevant client retry paths.
- [ ] Define deterministic matching, precedence, duration and consumption rules; isolate independent tests/accounts/apps where intended.
- [ ] Model rate limits by provider/product/operation and documented dimensions, using virtual time. Keep local overload/capacity protection distinct from simulated upstream quota errors.
- [ ] Preserve native provider error bodies/status/header semantics where evidenced, and label synthetic fallback policies. Do not invent universal retryability, rate quotas or idempotency.
- [ ] Instrument intentional simulated failures as expected outcomes rather than internal service errors.

Acceptance: exact failure sequences replay; retry/backoff tests need no sleeps; clock advance resets the intended windows; no cross-scenario leakage; expected injected errors do not inflate internal-failure telemetry.

### G. Webhook delivery

- [ ] Implement provider-specific event generation, payload encoding and signing against synthetic local callback fixtures. Preserve Meta versus TikTok verification/signature/event differences and raw-byte signing.
- [ ] Add bounded durable delivery state, retries/backoff, duplicate delivery, acknowledgements, failure inspection and deterministic clock controls.
- [ ] Keep callback destination policy explicit and safe. The no-provider-forwarding rule remains; locally configured webhook delivery is a separately implemented outbound feature, not a general URL fetcher.

Acceptance: local receiver verifies signature over exact bytes; retries and duplicates follow the selected scenario; persistence/reset semantics are tested; unreachable callbacks cannot block publishing or create unbounded work; tokens/private payloads never leak into logs.

### H. Snapshot/restore and reusable test clients

- [ ] Add versioned snapshot/export/restore through the admin surface, including state needed to reproduce time, IDs, random state, grants, scenarios and pending delivery work.
- [ ] Validate snapshots before atomic replacement; document secret/fixture content and compatibility/migration policy. Treat file persistence and reset as existing separate features, not completed snapshot support.
- [ ] Add small SDK/test-harness clients for URL rewriting and admin controls as justified by integration needs. Keep native provider SDK calls usable; avoid a new provider API dialect.

Acceptance: snapshot → mutate → restore reproduces relevant behavior, invalid restore leaves state unchanged, capacity limits remain effective, and SDK examples run in CI against the built server.

### I. Authorized contract verification and ongoing maintenance

- [ ] Design a separately run contract-recording/comparison tool and sanitization pipeline. Runtime/default tests must remain offline; recording is not a hidden proxy mode.
- [ ] Before any live account work, obtain explicit approval for the account, operations, mutations and potential charges. Documentation research is not permission to publish/upload/delete or probe rate limits.
- [ ] Capture only approved observations, replace credentials/private identifiers/content with safe fixtures, and review sanitization before committing. Never commit raw authenticated captures.
- [ ] Record source versions/dates, observed differences and unresolved branches; promote only specific evidenced behavior to live-verified.
- [ ] Maintain generation reproducibility, source inventories, version support, dependency hygiene and source-backed regression tests.

Acceptance: offline tests replay sanitized fixtures without credentials/network; provenance links each live claim to bounded evidence; an automated secret/privacy check plus review protects captures. If live access is absent, keep the implementation documentation-based/synthetic and say so.

## 7. Test and release bar for every change

Favor table-driven tests with clear `tc` cases. Use actual isolated SQLite for API-layer tests; mock only the boundary under test. Cover unit, integration and full workflow behavior, including malformed requests, native encodings, authorization products, partial grants, asset ownership/tasks, version isolation, atomicity, reset/restart and limits. Use the virtual clock instead of sleeps. Add browser E2E, client E2E and performance coverage where relevant rather than calling HTTP unit coverage equivalent.

From the repository root, with Go 1.26+ and a C compiler/CGO:

```sh
make fmt-check
make spec-validate
make generate-check
make vet
make test
make race
make build
go test -run '^$' -bench . -benchtime=1x -benchmem ./...
```

For a contract change also run `make generate` and review its source/generated diff; a fresh `make generate-check` must reproduce it. `make bench` runs fuller existing benchmarks but does not enforce performance budgets. Introduce measured budgets deliberately rather than inventing passed thresholds.

Before reporting done:

- Check the final diff for unrelated edits, generated drift, placeholders, sensitive data and misleading completion claims.
- Verify local documentation links and examples; keep supported runtime operations, reference-only operations, synthetic choices and unverified work distinct.
- Report passed, failed, blocked and not-run checks separately. Go tests do not prove Docker deployment, visual UX, external collector operation, live provider parity, or actual Posty integration.
- The repository owner requested direct commits to `main` without PRs for this work. Preserve unrelated changes, never force-push, and re-read the remote head before publishing. Follow any later owner instruction that changes that workflow.
- Verify the expected commit/file on remote `main`, then inspect CI for that exact commit through terminal success/failure. Diagnose failures; do not report an earlier commit's green CI as the new result.
- Update this handoff's baseline/status when implementation changes. Do not mark a task complete because a schema, interface, TODO, no-op handler or documentation section exists.

## 8. Known verification boundary at this baseline

The inspected code and tests establish a local Instagram publishing slice, four OAuth consent products, foundational storage/admin tooling, optional observability, and broad reference contracts. Browser visual QA, Docker/collector runtime checks, authorized live-provider parity, complete Posty end-to-end integration, and the remaining capabilities above are not established by this handoff. Keep those limits visible while making the next useful slice work end to end.
