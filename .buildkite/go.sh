#!/usr/bin/env bash
# Runs a command in the pinned Go container as the agent's own user, so the persistent
# checkout never gains root-owned files. Modules are downloaded first with network access;
# the command itself runs with --network none, so tests cannot reach providers.
set -euo pipefail

: "${GO_IMAGE:?GO_IMAGE must be set by the pipeline}"
[[ $# -gt 0 ]] || { echo "usage: $0 command [args...]" >&2; exit 2; }

cache="${SOCIAL_MOCKERY_CI_CACHE:-${HOME}/.cache/social-mockery-ci}"
mkdir -p "$cache/gocache" "$cache/gomodcache"

run() {
  local network="$1"
  shift
  docker run --rm --init --network "$network" \
    --user "$(id -u):$(id -g)" \
    --env HOME=/tmp \
    --env GOCACHE=/cache/gocache \
    --env GOMODCACHE=/cache/gomodcache \
    --env GOTOOLCHAIN=local \
    --env CGO_ENABLED=1 \
    --volume "$PWD:/src" --workdir /src \
    --volume "$cache:/cache" \
    "$GO_IMAGE" "$@"
}

run bridge sh -c 'go mod download && cd tools && go mod download'
run none "$@"
