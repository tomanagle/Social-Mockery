# Local API

Start the server with `profiles/example.yaml` as shown in the README. These examples need curl and Python 3 (only to read JSON response IDs). Use fixture credentials, never production tokens.

## Image: create, process, poll, publish

```sh
BASE='http://127.0.0.1:3000/https://graph.instagram.com/v25.0'
ADMIN='http://127.0.0.1:3000'
ACCOUNT='17841400000000001'
TOKEN='mock-token'

# Omit media_type for a single image.
CREATED=$(curl -fsS -X POST "$BASE/$ACCOUNT/media" \
  -H "Authorization: Bearer $TOKEN" \
  --data-urlencode 'image_url=https://example.com/fixture.jpg' \
  --data-urlencode 'caption=An image fixture')
CONTAINER=$(printf '%s' "$CREATED" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')

# Before advancing, the container is still processing.
curl -fsS "$BASE/$CONTAINER?fields=id,status,status_code" \
  -H "Authorization: Bearer $TOKEN"

curl -fsS -X POST "$ADMIN/admin/clock/advance" \
  -H 'Content-Type: application/json' -d '{"seconds":5}'
curl -fsS "$BASE/$CONTAINER?fields=id,status,status_code" \
  -H "Authorization: Bearer $TOKEN"

PUBLISHED=$(curl -fsS -X POST "$BASE/$ACCOUNT/media_publish" \
  -H "Authorization: Bearer $TOKEN" \
  --data-urlencode "creation_id=$CONTAINER")
MEDIA=$(printf '%s' "$PUBLISHED" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')
curl -fsS "$BASE/$MEDIA?fields=id,permalink" \
  -H "Authorization: Bearer $TOKEN"
curl -fsS "$BASE/$ACCOUNT/media" -H "Authorization: Bearer $TOKEN"
```

The permalink is simulator output, not evidence of a real Instagram publication. Image URLs are not fetched.

## Reel: create, process, poll, publish

Run in the same shell as the previous example:

```sh
CREATED=$(curl -fsS -X POST "$BASE/$ACCOUNT/media" \
  -H "Authorization: Bearer $TOKEN" \
  --data-urlencode 'media_type=REELS' \
  --data-urlencode 'video_url=https://example.com/fixture.mp4' \
  --data-urlencode 'caption=A Reel fixture')
CONTAINER=$(printf '%s' "$CREATED" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')

curl -fsS -X POST "$ADMIN/admin/clock/advance" \
  -H 'Content-Type: application/json' -d '{"seconds":5}'
curl -fsS "$BASE/$CONTAINER?fields=id,status,status_code" \
  -H "Authorization: Bearer $TOKEN"
curl -fsS -X POST "$BASE/$ACCOUNT/media_publish" \
  -H "Authorization: Bearer $TOKEN" \
  --data-urlencode "creation_id=$CONTAINER"
```

Advancing the clock affects all containers. Wall-clock waiting alone never finishes processing. Video URLs are not fetched or inspected.

## Routes

Provider routes are prefixed with `/https://graph.instagram.com/v25.0` (or either supported host/version combination):

- `POST /{account_id}/media`: create an image or Reel container
- `GET /{container_id}?fields=id,status,status_code`: inspect processing
- `POST /{account_id}/media_publish`: publish a finished container using `creation_id`
- `GET /{published_media_id}?fields=id,permalink`: inspect published media
- `GET /{account_id}/media`: list published media

Fixture access tokens can be supplied as an `Authorization: Bearer` header, an `access_token` query parameter, or an `access_token` form field. Headers avoid placing credentials in URLs and logs. Use the example token's scopes for the supported fixture flows.

## Admin and health

Admin routes are local server routes, without the provider prefix:

- `GET /healthz`: health probe
- `GET /admin/clock`: inspect the current virtual clock
- `POST /admin/clock/advance`: JSON body `{"seconds":5}`
- `POST /admin/reset`: reset simulator state; an empty body restores the startup seed, or supply a complete JSON/YAML profile

When `--admin-token local-secret` is set, include `Authorization: Bearer local-secret` on admin requests. This is distinct from the simulated Meta access token. The examples above assume the optional admin token is unset.

```sh
# Empty reset body: restore startup fixtures.
curl -fsS -X POST http://127.0.0.1:3000/admin/reset

# Replacement profile for this reset.
curl -fsS -X POST http://127.0.0.1:3000/admin/reset \
  -H 'Content-Type: application/yaml' \
  --data-binary @profiles/example.yaml
```

Reset deletes simulator-generated records and restarts IDs deterministically. Do not reset a shared instance while another test is using it. The admin surface intentionally contains no Phase 2 scenario, webhook, or snapshot APIs.
