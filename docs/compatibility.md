# Compatibility and scope

## Evidence and interpretation

The Phase 1 route design is informed by Posty's `workspaces/libs/platforms/src/instagram/publish.ts` and Meta's official Instagram API examples in the Meta Postman workspace. These are design references, not a claim of exhaustive parity or live compatibility certification.

The intentionally narrow target is a single-media publishing flow:

1. Create a container with an image URL, or `media_type=REELS` and a video URL.
2. Poll its processing state.
3. Publish using `creation_id`.
4. Retrieve the published media ID/permalink or list account media.

Real clients may poll using wall time; in the simulator their test harness must explicitly advance the virtual clock. The simulator keeps that clock frozen between advances, including across a persisted database restart.

## Supported routing envelope

Local paths embed `https://graph.instagram.com` or `https://graph.facebook.com`, followed by `v24.0` or `v25.0`. This is an allowlisted route format, not an arbitrary upstream proxy. A recognized Graph host does not make every endpoint for that host available.

The supplied BUSINESS account and fixture token include both Instagram Login-style business scopes and Facebook Login-style Instagram scopes so the supported publishing calls have predictable fixture authorization. This does not model login, real access-token introspection, or Meta app review.

## Intentional differences

- Media URLs are input strings; the simulator does not download, transcode, or validate actual image/video contents.
- Processing is a deterministic virtual delay, not a model of Meta's infrastructure or production latency.
- IDs and output permalinks are synthetic. Reset restores repeatable IDs.
- Fixture token expiry uses the simulator's clock; zero expiry denotes a non-expiring fixture.
- Published state is local SQLite data, never a real social post.
- The service has a configurable record bound and is intended for isolated local tests.

## Excluded from Phase 1

OAuth, app review, analytics, Facebook publishing, carousels, Stories, configurable scenarios, webhooks, snapshots, and SDK clients are not implemented. Broader Meta endpoints, production rate-limit fidelity, and complete Posty test-suite compatibility are not promised. Add a targeted test for any client behavior beyond the documented flow before relying on it.

### Running an existing publisher without clock hooks

Set `processing_seconds: 0` in its seed to make containers immediately ready while keeping the clock deterministic. This lets an unmodified polling loop complete. To test processing delays, the test harness must advance `/admin/clock/advance` while the publisher polls. Posty's source currently defines its Graph base URL in code; integrating this simulator requires routing those requests to the prefixed simulator address in the test setup. This repository does not modify Posty or claim its entire existing suite runs unchanged.
