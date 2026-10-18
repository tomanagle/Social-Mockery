# Instagram v25.0 coverage and evidence

Reviewed 2026-10-18. This is the simulator's limited v25.0 contract, not a full provider specification or live certification.

| Operation | Implemented slice | Evidence status |
| --- | --- | --- |
| POST /v25.0/{objectID}/media | Image URL or REELS video URL container | Documentation-based shape; independent release-specific behavior not live verified |
| GET /v25.0/{objectID} | Fixture account, processing status, published media | Documentation-based fields; fixture data and timing synthetic |
| POST /v25.0/{objectID}/media_publish | Publish a ready container | Documentation-based flow; simulator transition/errors synthetic |
| GET /v25.0/{objectID}/media | Local published media list | Documentation-based envelope; pagination cursor synthetic |

Initial reference: Meta-owned Instagram collection https://www.postman.com/meta/instagram/documentation/6yqw8pt/instagram-api . The collection does not establish exhaustive semantics or independent equivalence across API releases. During this review its public web rendering exposed only the documentation shell; no new release-specific claims were extracted.

Version reference attempted: https://developers.facebook.com/docs/graph-api/changelog/versions/ and https://developers.facebook.com/docs/graph-api/changelog/ (HTTP 429 on review). Latest-version verification remains pending. No authenticated API requests were made. No live recordings exist. Do not label this release latest, or copy these schemas into another release without source review.

The fixed version routes and generated Go package are independently maintained. Shared implementation for this slice is a deliberate simulator approximation, not a claim that all upstream behavior is equal. OAuth exchanges, app review, carousels, Stories, binary uploads, full analytics and other providers remain unsupported. Review docs/compatibility.md before integration.
