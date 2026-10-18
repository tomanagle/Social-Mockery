# Facebook Pages contracts

Native Facebook Pages reference contracts for Graph v26.0, v25.0 and v24.0. Reviewed against official documentation on 2026-10-18. Specifications and generated interfaces do not implement Page resource handlers. OAuth simulation is documented in the repository README; no live Meta calls were made.

## Coverage

Each version has 37 operations: 28 Page operations, 6 OAuth operations, 1 native Reel upload and 2 application webhook callbacks. The 111 versioned entries repeat releases, not distinct capabilities. [coverage.json](coverage.json) lists every method/path, source and file hash.

- Identity, managed-Page discovery, Page tokens/tasks and linked Instagram account fields
- Page/post/comment/photo/video reads; Page/post/comment edits and post/comment deletion
- Text/link/photo publishing, resumable videos, Reel start/upload/status/finish
- Feed, published/scheduled posts, comments/replies, insights and individual metrics
- Page settings, recommendations, comment blocking, search and integrity status
- App subscriptions and signed Page feed webhook callbacks
- Browser authorization, code/long-lived token exchange, permission inspection/revocation and token debugging

Shared node routes represent several node types without inventing URLs. Selected fields and permissions control responses. Properties remain open where providers can add fields. Graph errors do not claim exhaustive HTTP status mappings or retry guarantees.

## Authentication and access

Facebook Login grants a Facebook User token. Request `pages_show_list`, inspect granted scopes/assets, then call `/{userID}/accounts` (`userID=me` is valid) for selected Page IDs, tokens and tasks. Requested permissions are not approvals. Page tokens are tied to the user/app/Page grant and cannot expand authorized assets.

Publishing requires `pages_manage_posts`, `pages_read_engagement`, `pages_show_list`, a Page token and `CREATE_CONTENT`. Comment moderation uses `pages_manage_engagement`; user-generated content reads additionally use `pages_read_user_content`. Insights use `read_insights`, `pages_read_engagement` and `ANALYZE`. Webhook installation uses `pages_manage_metadata`, `pages_show_list` and an eligible task; removing the app's own subscription can use its App token. Public Page data is a separate reviewed feature. Do not turn these requirements into a universal scope union.

App Review/Advanced Access, app/business roles, Business Verification where applicable, consent, Page selection and Page tasks are separate checks. Test-role access does not prove production access. Inspect expiry, revocation and data-access expiry rather than assuming Page tokens last forever.

### Instagram through Facebook Login

`instagram_basic` is the valid Facebook Login permission for basic linked professional Instagram account/media access. It differs from `instagram_business_basic`, which belongs to Instagram Login. Facebook Login requires an eligible professional Instagram account linked to a Page and an appropriate Facebook User or Page token for the operation. `instagram_basic` alone does not authorize publishing, comments, insights or messaging. See the [Instagram contracts](../instagram/README.md) for operation-specific scopes and source/token differences. Page `subscribed_fields` does not configure Instagram webhook fields; those are subscribed in the app dashboard.

## Versions and provenance

The [official Graph release table](https://developers.facebook.com/docs/graph-api/changelog/versions/) was retrieved directly: v26.0 released 2026-07-29 (expiry TBD), v25.0 expires 2028-07-29, v24.0 expires 2028-02-18. Marketing API release/retirement tables differ.

Current reference behavior is represented for three supported releases. This is not independent historical verification of fields, metrics, permissions or rollouts. [sdk-inventory.json](sdk-inventory.json) records pinned official Business SDK revisions, source hashes and inspected Page/Post/Comment/Photo/AdVideo/User methods. SDK methods establish client capabilities, not universal server requirements.

The [official Meta Postman team](https://www.postman.com/meta/) identifies its Facebook workspace. Its [managed-Page token request](https://www.postman.com/meta/facebook/request/bqfxwbp/get-access-tokens-of-pages-you-manage) corroborates discovery fields. This mutable example is not a pinned complete collection export. Primary sources remain the [Pages index](https://developers.facebook.com/documentation/pages-api/llms.txt), per-operation references, [Facebook Login manual flow](https://developers.facebook.com/docs/facebook-login/guides/advanced/manual-flow/), [Video publishing](https://developers.facebook.com/docs/video-api/guides/publishing/) and [Reels publishing](https://developers.facebook.com/docs/video-api/guides/reels-publishing/).

## Source conflicts and limits

- Some guides spell permissions `pages_manage_read_engagement` or `pages_read_user_engagement` and mention `publish_video`. Current endpoint requirements instead name `pages_read_engagement`, `pages_read_user_content` where applicable, and `pages_manage_posts`. Misspellings are not valid aliases here.
- Page Videos reference says GET is unavailable while the SDK has `get_videos`. No GET `/videos` is included. Reels listing uses its documented edge.
- Reel finish prose says `complete`, but its enum/example/SDK say `finish`; the latter is retained. Returned upload URLs/prose sometimes omit the version while upload examples include it. This contract captures the versioned example; clients should use returned URLs.
- Insights guides illustrate deprecated impression/fan metrics. All-version 2025/2026 deprecations apply even to older Graph paths. Metrics remain strings with applicability notes rather than an obsolete exhaustive enum.
- Integrity prose says `/pages_status`, but its request shows `/{pageID}/page_status`; the demonstrated route is retained.
- Page webhook examples show an outer array and quoted success values in places; generic delivery uses an object and endpoint references use boolean success. Callback paths and rejection policy belong to the receiver. Validate signatures against raw bytes, never reserialized JSON.
- Excluded: personal-profile publishing, Groups, Marketing/Ads, Business Manager administration, Messenger conversations, Live Video, Stories, commerce/orders, arbitrary batching, video collaborators and app-level Graph file-handle uploads. Video publishing accepts a handle obtained elsewhere without claiming its upload lifecycle is implemented.
- URL uploads describe upstream semantics; Social Mockery never fetches URLs or forwards requests. Fixtures use synthetic IDs/tokens. No live credentials, account calls or posting were used.

Run `make spec-validate`, `make generate` and `make generate-check` for deterministic validation/generation with the pinned OpenAPI toolchain.
