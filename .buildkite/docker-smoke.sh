#!/usr/bin/env bash
# Builds the production image, checks health, and confirms that state on a named volume
# survives a container restart without --seed.
set -euo pipefail

suffix="${BUILDKITE_JOB_ID:-local}"
name="social-mockery-ci-${suffix}"
image="social-mockery:ci-${suffix}"
volume="social-mockery-ci-${suffix}"
base=""

cleanup() {
  local status=$?
  if [[ $status -ne 0 ]]; then
    docker logs "$name" 2>&1 | tail -n 50 || true
  fi
  docker rm -f "$name" >/dev/null 2>&1 || true
  docker volume rm "$volume" >/dev/null 2>&1 || true
  docker image rm "$image" >/dev/null 2>&1 || true
  exit "$status"
}
trap cleanup EXIT

start() {
  docker run --detach --name "$name" --publish 127.0.0.1::3000 --volume "$volume:/data" \
    "$image" --addr :3000 --db /data/social-mockery.db "$@" >/dev/null
  local port
  port="$(docker port "$name" 3000/tcp | head -n 1 | sed 's/.*://')"
  base="http://127.0.0.1:${port}"
  curl -fsS --retry 30 --retry-all-errors --retry-delay 1 "$base/healthz" >/dev/null
}

docker build --file dev/Dockerfile --tag "$image" .

start --seed /app/profiles/example.yaml
curl -fsS -X POST "$base/admin/clock/advance" -H 'Content-Type: application/json' -d '{"seconds":42}' >/dev/null
before="$(curl -fsS "$base/admin/clock")"
docker stop "$name" >/dev/null
docker rm "$name" >/dev/null

start
after="$(curl -fsS "$base/admin/clock")"
if [[ "$before" != "$after" ]]; then
  echo "Clock state changed across restart: before=${before} after=${after}" >&2
  exit 1
fi
echo "Restart without --seed preserved state: ${after}"
