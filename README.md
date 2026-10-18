# Social Mockery

Test social integrations locally without real accounts, uploads or API calls. Social Mockery simulates OAuth permission screens and a focused Instagram publishing workflow, with seeded accounts, repeatable IDs and a clock you control.

## Supported providers

Runnable simulation:

- [x] Instagram: image/Reel publishing on Graph v24.0 and v25.0; Instagram Login and Facebook Login consent/code exchange (Facebook Login v24.0–v26.0)
- [x] Facebook: Login consent and code exchange on Graph v24.0, v25.0 and v26.0
- [x] TikTok: v2 Login consent, code exchange and token refresh
- [ ] Facebook Page publishing and TikTok publishing at runtime

Reference specifications and generated Go interfaces: [Instagram v24–v26](specs/meta/instagram/README.md), [Facebook v24–v26](specs/meta/facebook/README.md), [TikTok v2](specs/tiktok/README.md). These cover more endpoints than the running simulator.

## Run it

Requires Go 1.26+, CGO and a C compiler (SQLite).

```sh
make build
./bin/social-mockery --addr 127.0.0.1:3000 --db :memory: --seed profiles/example.yaml
curl -fsS http://127.0.0.1:3000/healthz
```

Or use Docker with live reload: `make dev` (stop with `make dev-down`). For persistent state use `--db ./social-mockery.db`; omit `--seed` on later starts to keep existing data.

## Use it

Point your integration at the local server, keeping the original host and API path:

```text
https://graph.instagram.com/v25.0/17841400000000001/media
→ http://127.0.0.1:3000/https://graph.instagram.com/v25.0/17841400000000001/media
```

The example account is `17841400000000001`, with token `mock-token`. Create a publishing container:

```sh
curl -fsS -X POST \
  http://127.0.0.1:3000/https://graph.instagram.com/v25.0/17841400000000001/media \
  -H 'Authorization: Bearer mock-token' \
  --data-urlencode 'image_url=https://example.invalid/photo.jpg'
```

Advance processing by five virtual seconds, then publish the returned container ID via `/{account_id}/media_publish`. [Copy-and-paste publishing examples](docs/api.md).

```sh
curl -fsS -X POST http://127.0.0.1:3000/admin/clock/advance \
  -H 'Content-Type: application/json' -d '{"seconds":5}'
curl -fsS -X POST http://127.0.0.1:3000/admin/reset
```

For browser login/consent tests, start with `--seed profiles/oauth-demo.yaml`. Choose a mock account, approve or deny permissions, select Page assets and exchange the returned code. Configure your callback in the seed; the demo callback is an illustrative address. [OAuth examples and setup](docs/oauth-consent.md).

## Keep it local

Use fixture credentials only. The server never forwards provider requests or downloads media URLs. Keep the loopback listener; protect admin routes with `--admin-token YOUR_LOCAL_SECRET` if needed. Reset or explicit seeding replaces simulator state. Behavior is documentation-based or synthetic, not live-provider verified.

[Development and tests](docs/development.md) · [API compatibility](docs/compatibility.md) · [Specifications](docs/specifications.md) · [Optional telemetry](docs/observability.md) · [MIT license](LICENSE)
