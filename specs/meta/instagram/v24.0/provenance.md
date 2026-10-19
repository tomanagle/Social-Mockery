# Instagram v24.0 coverage and evidence

Reviewed 2026-10-18. This is the simulator's limited v24.0 contract, not a full provider specification or live certification.

| Operation | Implemented slice | Evidence status |
| --- | --- | --- |
| POST /v24.0/{objectID}/media | Image URL or REELS video URL container | Documentation-based shape; independent release-specific behavior not live verified |
| GET /v24.0/{objectID} | Fixture account, processing status, published media | Documentation-based fields; fixture data and timing synthetic |
| POST /v24.0/{objectID}/media_publish | Publish a ready container | Documentation-based flow; simulator transition/errors synthetic |
| GET /v24.0/{objectID}/media | Local published media list | Documentation-based envelope; pagination cursor synthetic |
| GET /v24.0/me | Instagram Login token owner: id, user_id, username, account_type, media_count | Documentation-based fields ([/me reference](https://developers.facebook.com/documentation/instagram-platform/reference/me), [Business Login guide](https://developers.facebook.com/documentation/instagram-platform/instagram-api-with-instagram-login/business-login)); app-scoped id derivation, omission of fields without fixture data, and errors synthetic. Added 2026-10-19 |

Initial reference: Meta-owned Instagram collection https://www.postman.com/meta/instagram/documentation/6yqw8pt/instagram-api . The collection does not establish exhaustive semantics or independent equivalence across API releases. During this review its public web rendering exposed only the documentation shell; no new release-specific claims were extracted.

The initial runtime-subset review encountered HTTP 429 on the version references. The subsequent 2026-10-18 comprehensive contract review verified the [official Graph version table](https://developers.facebook.com/docs/graph-api/changelog/versions/): v26.0 is latest, while v24.0/v25.0 remain supported. See the [updated coverage audit](../README.md). This file describes the five-operation runtime subset. No authenticated API requests were made and no live recordings exist; current documentation is not independent historical semantic verification.

The fixed version routes and generated Go package are independently maintained. Shared implementation for this slice is a deliberate simulator approximation, not a claim that all upstream behavior is equal. OAuth exchanges, app review, carousels, Stories, binary uploads, full analytics and other providers remain unsupported. Review docs/compatibility.md before integration.
