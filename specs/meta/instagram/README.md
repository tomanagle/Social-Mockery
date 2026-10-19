# Instagram contract coverage audit

Reviewed 2026-10-18 against the official [Instagram Platform index](https://developers.facebook.com/documentation/instagram-platform/llms.txt), [Instagram Messaging index](https://developers.facebook.com/documentation/business-messaging/instagram-messaging/llms.txt), endpoint references, and pinned official SDK/collection evidence.

The repository contains five **contract-only** products per version: Facebook Login, Instagram Login, OAuth, native upload, and application webhook callbacks. These are source-backed reference specifications, not a complete running Meta emulator. Their generated packages are isolated under `internal/contracts/instagram/`. The runnable subset is five operations in each of v24.0 and v25.0 (four publishing operations plus `/me`); v26.0 has no runtime adapter. The OAuth runtime additionally serves Instagram Login long-lived token exchange and refresh through the reference OAuth adapters.

## Version evidence

The official [Graph API version table](https://developers.facebook.com/docs/graph-api/changelog/versions/) lists v26.0 as the latest release, released 2026-07-29, with expiration TBD. v25.0 expires 2028-07-29 and v24.0 expires 2028-02-18. Marketing API retirement dates are different and are not used for these Graph contracts.

Current documentation plus pinned SDK evidence supports the contract inventory. Repeating current reference behavior in three release directories does **not** independently verify every historical permission, field, enum, metric, rollout, account restriction, or server response. No operation is live-verified.

## Facebook Login access and consent

The [current Instagram overview](https://developers.facebook.com/documentation/instagram-platform/overview) requires a professional Instagram account linked to a Facebook Page for Facebook Login. Its scopes remain `instagram_basic`, `instagram_content_publish`, `instagram_manage_comments`, `instagram_manage_insights` and `instagram_manage_messages`, with operation-specific Page scopes. `instagram_business_basic` and the `instagram_business_*` family belong to Instagram Login. These are distinct authorization products.

Facebook User and Page tokens are both used, with operation-specific requirements. The [getting-started read flow](https://developers.facebook.com/documentation/instagram-platform/instagram-api-with-facebook-login/get-started) uses a User token for Page discovery and Instagram media reads. Publishing endpoint references specify User tokens, while the publishing guide and provider collection also show Page-token publishing. Their Business Manager scope requirements differ too; the contracts preserve each source instead of flattening them into a universal requirement. Permission-reference dependencies are separately labeled evidence because they differ from individual endpoint and getting-started scope tables.

Page tasks constrain which permissions a person can grant. Current overview names use `PROFILE_PLUS_*`, while some endpoint references retain `MANAGE`, `CREATE_CONTENT` and other legacy labels. Standard Access covers owned/managed professional accounts and applicable app/business roles. Serving other accounts requires Advanced Access, App Review and Business Verification. These are authorization requirements, not extra HTTP endpoints or implemented simulator policies.

The OAuth dialog includes the optional [Facebook Login for Business configuration ID](https://developers.facebook.com/documentation/facebook-login/facebook-login-for-business/). Ordinary Facebook Login supports declining individual permissions; Facebook Login for Business states that all requested permissions must be granted or none are granted. Do not apply that business rule to ordinary Facebook Login.

The [Page subscription reference](https://developers.facebook.com/docs/graph-api/reference/page/subscribed_apps/) has method-specific access rules: GET/POST require Page-token scope/task eligibility, while DELETE documents an App-token branch for removing the app's own subscription. Its `subscribed_fields` parameter cannot configure Instagram topic webhooks; that setup uses the app dashboard.

## Measured inventory

| Version | Facebook Login | Instagram Login | OAuth | Upload | Webhooks | Total operations |
|---|---:|---:|---:|---:|---:|---:|
| v24.0 | 60 | 29 | 13 | 1 | 2 | 105 |
| v25.0 | 60 | 29 | 13 | 1 | 2 | 105 |
| v26.0 | 60 | 29 | 13 | 1 | 2 | 105 |

There are 315 method/path entries across the 15 documents. Count each HTTP method/path once per product; shared node reads and payload branches are not additional operations. Counts repeated across versions are not distinct provider capabilities. [coverage.json](coverage.json) records the exact operation IDs, source links, file hashes, schema counts, and a page-by-page index reconciliation. [sdk-inventory.json](sdk-inventory.json) is supporting SDK evidence, not a promise that every SDK-adjacent Ads/Business edge is implemented or included in the current public contract.

## Official category audit

| Category | Contract coverage and boundary |
|---|---|
| Login and account discovery | Separate Instagram/Facebook authorization and token flows, refresh/exchange, Facebook permission inspection/revocation and token debugging; account/Page discovery in login contracts. No Instagram debug/revoke endpoint invented by analogy. |
| Account/media/container reads | Copyright check status/information are container/media fields, not separate routes. Typed fields and field projection on shared node routes; children, media, stories/live media where documented. Business discovery and mentioned-media/comment selection remain fields, not invented endpoints. |
| Publishing | Image, Reel/video, carousel and Stories variants, container status, publish, quota and publishing errors; product-specific restrictions and wire encodings retained. |
| Native upload and audio | Dedicated `rupload.facebook.com` upload contract; raw bytes/offset/file-size or hosted URL variants. Audio root edge remains Facebook Login. Upload presence does not imply all flows/accounts support every mode. |
| Comments, tags, mentions and hashtags | Comment reads/replies/moderation/deletion, tags/mentions, Facebook hashtag search/recent/top media and recently searched hashtags. Shared paths preserve node-specific requirements. |
| Insights | Account and media metrics, period/breakdown differences and documented deprecations retained in schemas/extensions. Unknown historical combinations are not asserted. |
| Messaging and profiles | Messages, templates, quick replies, sender actions, private replies, self messaging, attachments, conversations, scoped-user reads, menus/ice breakers and supported welcome flows. Different token, identity and eligibility models remain separate. |
| Routing and moderation | Facebook Page conversation-control and moderation APIs. Conversation ownership never overrides messaging-policy windows; obsolete Handover Protocol branding is not a new product. |
| Creator marketplace | Creator discovery/insights projections and brand custom-audience information, permissions, eligibility and documented Graph error codes. Unspecified nested response envelopes remain identified as uncertain. |
| Shopping and collaborators | Catalogs/product search/variants, product tags, product appeals and media collaborators. Reference contradictions are preserved. |
| Threads-linked account references | Connected Threads read plus Instagram-backed Threads read/create only; no full Threads or Ads API imported. |
| Events and collaboration management | Facebook Login upcoming-event creation/listing and event node reads/updates, collaboration invitations/responses and collaborative-media reads are included. No event DELETE or undocumented list pagination inferred. |
| Webhooks | Verification GET, delivery POST, raw-body HMAC-SHA256 signature metadata, typed messaging/comment/mention variants and post-share transition. Callback path is application-chosen, not a Meta API endpoint. |
| oEmbed, sharing and embeds | oEmbed HTTP endpoint included. Mobile sharing intents, universal links, ig.me links and embed-button UI are integration workflows, not fabricated Graph routes. |
| App creation/review, migration and policy | Context and access requirements, not separate REST operations. |
| Marketing messages / opt-in | Indexed guides return missing-page content; production request contract remains unverified. Internal-host opt-in sample excluded from active send variants. |

## Changelog reconciliation

The official [Instagram changelog](https://developers.facebook.com/documentation/instagram-platform/changelog.md) documents media like/unlike operations from 2026-04-22 across Graph versions. These are represented in the Facebook Login contract. Webhook changes include `reply_to.story.link_sticker_url` and `message_edit` applicability across both login products.

Named insights updates include `link_clicks`, `total_likes`, `facebook_views`, `crossposted_views`, `reels_skip_rate`, and `reposts`. Metric inputs remain supported string selections rather than a falsely exhaustive global enum. Their presence in changelog evidence does not establish that every metric works on every account, media type, period, breakdown or historical release; operation-specific applicability and deprecations still govern.

## Known source conflicts and remaining boundaries

- Instagram authorization sources disagree on `www.instagram.com` versus `api.instagram.com`; retain source-specific host evidence.
- Instagram Login code exchange: the Business Login guide wraps the response in `data`, but Posty's production client parses a flat object, which the repository owner reported working on 2026-10-19. The runtime returns the flat object and accepts Posty's `www.instagram.com/oauth/authorize/` trailing-slash URL; the reference contracts keep the documented shapes. See docs/oauth-consent.md for the exact, limited live-verified claim.
- Instagram `/me` reference uses a direct user response while a getting-started example wraps `data`; preserve the conflict. Permission deletion reference uses `success` while a guide describes literal `true`.
- Media insights period tables and response prose differ. Permission tables also differ on conditional Business Manager scopes; avoid combining alternatives into a universal requirement. Publishing references use User tokens and alternative ads scopes while the guide uses Page tokens and lists both ads scopes. The Facebook conversations HTTP guide uses lowercase `platform=instagram`, while the SDK enum is uppercase `INSTAGRAM`; the server parameter remains permissive rather than treating SDK validation as exhaustive.
- Product-tag reference spells `ad_reads`, disagrees with samples/SDK on `string_sale_price_string` versus `stripped_sale_price_string`, and excludes Reels reads while supporting Reels tagging writes.
- Some messaging samples have wrong host, action or unrelated welcome-flow response; these do not authorize cross-product host substitution or false response shapes.
- Moderation prose describes boolean success while malformed examples quote it; Page lower-case actions and IGUser SDK upper-case enums must remain distinct.
- Webhook examples contain malformed illustrative JSON and conflicting comment nesting. The `story_insights` example repeats a postback payload; no story-insights schema was fabricated. The `ig_post` transition replaces legacy Instagram post `share` attachments from 2026-02-01.
- The indexed Instagram Login conversation-routing and upload-media aliases returned navigation HTML during this audit. Use the separately confirmed product-specific guides; an alias does not prove cross-login applicability.
- Marketing-message and opt-in pages are unavailable. A notification opt-in example targets `graph.intern.facebook.com`, which is not an approved public server and is not treated as production evidence.
- Where documentation specifies only Graph error codes or an incomplete response shape, no exhaustive HTTP status mapping, retry safety, required fields, or enum completeness is inferred. `additionalProperties: false` in evidence-only fixture schemas does not mean Meta promises never to add fields.
- Marketing API partnership-ad account/post operations, including the 2026-03-30 branded_content_ad_permissions creator_username filter and ad-code generation, are explicitly outside this Instagram publishing/messaging contract scope. Their changelog mention is not a missing Instagram runtime feature or permission to import the full Ads API.
- Instagram Login stories/live_media applicability is not inferred from Facebook Login edge references; retain product-specific uncertainty until a primary per-edge source resolves it.
- No authorized live account tests were run. Validation, deterministic generation and compilation test local consistency, not server fidelity. Neither schema presence nor generation registers runtime routes.

## Reproducibility and sources

Run `make spec-validate`, `make generate`, and `make generate-check`; broader runtime checks are documented in [the specification workflow](../../../docs/specifications.md). Generation uses pinned official `oapi-codegen`, not a custom interface generator.

- [Official Instagram Platform documentation](https://developers.facebook.com/documentation/instagram-platform/llms.txt)
- [Official Instagram Messaging documentation](https://developers.facebook.com/documentation/business-messaging/instagram-messaging/llms.txt)
- [Pinned Meta collection](https://github.com/fbsamples/messenger-platform-samples/blob/3c3853a84a5d87d2567e75e649655822f535fdf6/postman/instagram-platform-api.postman_collection.json)
- [Official Business SDK](https://github.com/facebook/facebook-python-business-sdk): v24 revision `f1722382526f18b5ab33de007641c4b89f5d86fc`, v25 `542e10e31c40f7925f9ac03c000922a2fd0a2365`, v26 `efd8423a2e595ea8d4c04eb824ce113f2f1d68cd`.

Per-operation source links and caveats are authoritative for each local contract claim. The older `v24.0/provenance.md` and `v25.0/provenance.md` describe the unchanged runtime subset and its earlier research, not this expanded contract audit.
