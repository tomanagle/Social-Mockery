# Compatibility and scope

## Evidence and interpretation

The Instagram publishing route design is informed by Posty's `workspaces/libs/platforms/src/instagram/publish.ts` and Meta's official Instagram API examples in the Meta Postman workspace. These are design references, not a claim of exhaustive parity or live compatibility certification.

The resource simulator supports a narrow single-media publishing flow:

1. Create a container with an image URL, or `media_type=REELS` and a video URL.
2. Poll its processing state.
3. Publish using `creation_id`.
4. Retrieve the published media ID/permalink or list account media.

It also serves `GET /me` on `graph.instagram.com` for Instagram Login connection checks. Supported fields are `id`, `user_id`, `username`, `account_type` and `media_count`. `user_id` is the professional account ID used by publishing paths. `id` is the Instagram-scoped user ID, which is a synthetic deterministic value per app and account. `account_type` uses the documented `Business` and `Media_Creator` values; personal accounts are rejected. Documented fields with no fixture data (`name`, `profile_picture_url`, `biography`, `website`, `followers_count`, `follows_count`) are omitted rather than invented. Facebook Login `/me` on `graph.facebook.com` is not simulated.

Real clients may poll using wall time; in the simulator their test harness must explicitly advance the virtual clock. The simulator keeps that clock frozen between advances, including across a persisted database restart.

## Supported routing envelope

Local paths embed `https://graph.instagram.com` or `https://graph.facebook.com`, followed by `v24.0` or `v25.0`. This is an allowlisted route format, not an arbitrary upstream proxy. A recognized Graph host does not make every endpoint for that host available.

Meta documents that an unversioned call uses the version set in the app dashboard. The simulator resolves an unversioned path that starts with `me` or a numeric node ID through the token's app fixture `graph_api_version`. The token must be in the query string or `Authorization` header. Legacy fixture tokens have no app, so unversioned calls with them are rejected, as are apps without `graph_api_version`. A configured version without a runtime adapter, such as `v26.0`, returns the normal unsupported-version error.

The supplied BUSINESS account and fixture token include both Instagram Login-style business scopes and Facebook Login-style Instagram scopes so the supported publishing calls have predictable fixture authorization. This legacy fixture shortcut is separate from the synthetic OAuth flow. OAuth-created tokens preserve their login product, approved/granted scopes and selected assets. Real token introspection and Meta app review are not performed.

## Intentional differences

- Media URLs are input strings; the simulator does not download, transcode, or validate actual image/video contents.
- Processing is a deterministic virtual delay, not a model of Meta's infrastructure or production latency.
- IDs and output permalinks are synthetic. Reset restores repeatable IDs.
- Fixture token expiry uses the simulator's clock; zero expiry denotes a non-expiring fixture.
- Published state is local SQLite data, never a real social post.
- The service has a configurable record bound and is intended for isolated local tests.

## Not implemented

Real app review, analytics, Facebook/TikTok publishing, carousels, Stories, configurable scenarios, webhook delivery, snapshots and SDK clients are not implemented. Broader Meta endpoints, production rate-limit fidelity, and complete Posty test-suite compatibility are not promised. Add a targeted test for any client behavior beyond the documented flow before relying on it.

### Running an existing publisher without clock hooks

Set `processing_seconds: 0` in its seed to make containers immediately ready while keeping the clock deterministic. This lets an unmodified polling loop complete. To test processing delays, the test harness must advance `/admin/clock/advance` while the publisher polls. Posty's source currently defines its Graph base URL in code; integrating this simulator requires routing those requests to the prefixed simulator address in the test setup. This repository does not modify Posty or claim its entire existing suite runs unchanged.


## OAuth simulation

[OAuth consent flows](oauth-consent.md) run locally for Instagram Login, Instagram through Facebook Login, Facebook Login and TikTok Login. Facebook authorization/code exchange supports v24.0, v25.0 and v26.0; Instagram authorization/token exchange is unversioned; TikTok uses v2 and supports token refresh. These do not expand Instagram resource endpoints beyond v24.0/v25.0 or activate Facebook/TikTok resource APIs.

The screens are mock consent interfaces with seeded accounts, permission choices and eligible Page assets. They never collect real passwords or contact upstream providers. App-approved scopes, granted scopes, exact registered redirects, code expiry/replay and token grants are simulated; production login, identity verification, complete Business Login configurations and provider security infrastructure are not reproduced. See the OAuth guide for exact supported routes and limitations.

Native reference specifications cover broader [Instagram](../specs/meta/instagram/README.md), [Facebook](../specs/meta/facebook/README.md) and [TikTok](../specs/tiktok/README.md) surfaces. Generated interfaces and schema validation do not turn reference operations into working handlers.
