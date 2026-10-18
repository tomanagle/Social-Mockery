# TikTok v2 reference contracts

Reviewed 2026-10-18. These five native OpenAPI documents generate typed Go server interfaces under `internal/contracts/tiktok/v2/`. They are **documentation-based reference contracts**, not a claim that the entire TikTok surface runs in the simulator. Runtime OAuth consent, if enabled by an adapter, remains a separate bounded synthetic implementation; schema presence does not register a route. No TikTok account or live authenticated API was used.

## Version and product boundaries

TikTok uses `/v2/` for these HTTP APIs. This is not Meta's dated Graph release system, the mobile SDK's `2.x.y` release, or TikTok for Business's `/open_api/v1.3/`. Current [migration guidance](https://developers.tiktok.com/docs/en/tiktok-api-v2-introduction) and endpoint references establish the selected generation. The [OAuth v1 web migration](https://developers.tiktok.com/bulletin/migration-guidance-oauth-v1) ended v1 web support on 2024-02-29, and the [QR migration](https://developers.tiktok.com/bulletin/migration-guidance-qr-code-authorization-v2) likewise retired old QR endpoints. No previous version is copied under a new directory or represented as currently supported without product-specific evidence. The older OAuth bulletin says mobile should remain on v1 until further notice, while the current user-token reference explicitly accepts mobile codes and requires their verifier; this historical instruction is not silently generalized to current web or mobile support.

## Measured inventory

| Product | Operations | Scope |
| --- | ---: | --- |
| OAuth / Login Kit | 5 | Browser authorization, code/refresh token exchange, revoke, QR creation/status |
| Display | 3 | User/profile projection, public-video list, owned-video query |
| Content Posting | 6 | Direct video, inbox video, direct/inbox photos, creator info, status, download cancellation |
| Upload transport | 1 | PUT binary chunks to an opaque returned upload URL |
| Webhook receiver | 1 | Application-chosen callback receiving authorization and posting events |
| Total | 16 | Grants, post modes and event variants do not inflate operation counts |

[coverage.json](coverage.json) records exact operations, sources and spec hashes. [source-inventory.json](source-inventory.json) records public retrieval evidence and the bounded SDK/Postman investigation. The upload and callback entries are clearly labeled transport/receiver templates, not invented fixed upstream routes.

## Authentication and access

- Login uses `client_key`, comma-separated scope requests, a registered redirect URI, and anti-CSRF `state`. The callback's `scopes` and token response's `scope` represent granted access, which may be a subset of requested access.
- Web redirects must be static registered HTTPS URIs without query or fragment. Desktop loopback redirects have distinct host/port rules. Desktop requires PKCE; the official guide and pinned iOS implementation use hexadecimal SHA256 for the challenge. Web PKCE is not asserted universally required. Mobile app-switch/deep-link integration belongs to its SDK, not new REST endpoints.
- Token exchange, refresh and revoke are form encoded. Exchange and refresh share `/v2/oauth/token/`. Initial access-token lifetime is documented as 24 hours; refresh lifetime as 365 days. Refresh can rotate the refresh token. Revoke success has an empty response struct, not an invented `success: true` field. OAuth errors are flat `error`, `error_description`, `log_id` fields, unlike the nested API error object.
- QR authorization includes `client_ticket` integrity verification. Status `confirmed` is not itself a token. The reference table names `code`, but the example returns `redirect_uri` containing the code; both forms remain visible without claiming both are required.
- Profile fields are divided between `user.info.basic`, `user.info.profile` and `user.info.stats`. `video.list` exposes authorized public videos. Application product/scope approval and user consent are independent gates. `video.publish` and `video.upload` are distinct; an app/client token cannot replace a user grant.

## Posting and transport constraints

- Direct Post requires fresh creator-info settings and explicit user consent. Clients must respect the returned privacy options, interaction flags and account-specific duration; there is no silent privacy default. The [posting UX guidelines](https://developers.tiktok.com/docs/en/content-sharing-guidelines) require audit before lifting private-only limitations. Unaudited clients are limited to five posting users per 24-hour window and private accounts; creator and posting caps also apply, and typical daily figures are not universal fixed quotas.
- Video direct post uses `post_info`; inbox video uses only `source_info`. Photos use `/content/init/` with `DIRECT_POST` or `MEDIA_UPLOAD`. Photo `is_aigc` is top-level; video `is_aigc` is inside `post_info`. Inbox content still requires the creator to finish in TikTok. The photo upload reference requires TikTok app 31.8 or later and permits title/description.
- Init limits are six requests/minute per user access token; creator-info is 20 and status is 30. Creator-info can intentionally return HTTP 200 with posting-cap or banned-user errors. All clients must inspect `error.code`. These are API-specific limits, not an invented single global rate bucket.
- URL pulls require verified ownership, HTTPS, no redirects, and availability until the task finishes (documented one-hour download timeout). Photo sources contain at most 35 URLs and use URL pull only. The guide currently lists JPEG/WebP images, max 20 MB each and 1080p; it is not evidence for arbitrary image formats.
- File upload uses the entire returned signed URL, not a reconstructed host/path/query. Examples use different regional hosts and `/video/` versus `/upload/`. The contract's `/{uploadPath}` only captures the transport shape; generated `ServeMux` matching does not implement arbitrary returned upstream paths. Upload URLs expire after one hour. Chunks are sequential, generally 5–64 MB; the final chunk may include trailing bytes up to 128 MB, and small files use one chunk. The reference uses floor division for `total_chunk_count`, with trailing bytes included in the final chunk.
- HTTP 206 accepts an intermediate chunk; 201 starts processing after the last chunk. Completion of transfer does not prove publication. `SEND_TO_USER_INBOX` waits for creator action. Public IDs are withheld until moderation. Cancellation is best-effort for ongoing URL downloads, not deletion of posted content. No universal completion time or safe retry of initialization is asserted.

## Response fidelity and unresolved source details

- Display cursor is an int64 timestamp in **milliseconds**; video `create_time` is in **seconds**. Video IDs are strings. Status `publicaly_available_post_id` is an int64 list with the provider's misspelling retained. Field selection means omitted fields remain optional; no unverified nullability is added.
- General API errors use `invalid_params`; posting references use `invalid_param`. Both sources are retained without normalizing distinct wire codes. Posting tables spell `logid`, while examples and generic references use `log_id`; the modeled wire field follows the latter. Error codes remain open strings rather than falsely exhaustive enums.
- Branded-content fields are marked required in some posting tables but omitted in their same-page examples. They remain typed optional with this conflict recorded; an implementer must decide how to emulate a chosen policy scenario rather than treating either sample as universal evidence.
- Photo and video text limits are expressed in UTF-16 units by the provider. They are documented rather than incorrectly encoded as Unicode-codepoint `maxLength` limits. Conditional and cross-field constraints such as photo-cover index bounds, chosen privacy versus creator options, granted scopes versus post mode, and chunk byte arithmetic need handwritten validation when implemented.
- QR status examples include `utilised` beyond the table's four statuses. Status remains an open string. The query-video example mentions `fps`, which is absent from the Video Object field table; no `fps` field is invented.
- Webhook payload `content` is a JSON-serialized **string**, not a nested object. HMAC-SHA256 uses `client_secret` over the timestamp, a dot, and the unchanged raw request body. The guide leaves tolerated timestamp age to the consumer. HTTPS receiver acknowledges HTTP 200; best-effort at-least-once retries run up to 72 hours, so deduplication is required. No Meta-style GET challenge is invented.
- Generic webhook documentation lists older Video Kit `video.upload.failed` / `video.publish.completed` with `share_id`. Content Posting documents different `post.publish.*` events with `publish_id`; these are not silently merged. The documented `post.publish.no_longer_publicaly_available` typo is retained. Webhook `post_id` scalar type is not specified in its event table, so its decoded schema admits string or integer while the HTTP status schema retains documented int64.

## Explicit exclusions

Research, Commercial Content, Data Portability, TikTok for Business Ads/Organic/Business Messaging, Shop, Minis and monetization are separate products with different access and contracts. Their presence in a documentation sidebar or an official SDK/Postman collection does not include them here. Client-credentials token issuance is documented for Research/Commercial Content and intentionally excluded from this user-social scope. Mobile Share Kit is a native workflow, not fabricated server endpoints. Embed/oEmbed is not included in this bounded authenticated integration surface. No general comment management, likes mutation, video deletion, direct messaging, refresh-only route, scope-list route or webhook-registration route is invented for the selected products.

## Reproducibility

Run `make spec-validate`, `make generate`, `make generate-check`, and `make test`. The pinned `oapi-codegen` generates all interfaces; no generated Go file is edited by hand. `tools/cmd/specvalidate/tiktok_test.go` checks form encodings, grant/source/photo variants, projections, pagination, webhook string payloads, exact spelling and hash/count inventory. These tests establish local consistency only; they are not live-provider tests. Runtime consent coverage and synthetic deviations are documented separately by its implementing adapter.
