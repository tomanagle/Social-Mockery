#!/bin/sh
# Generate reference models/interfaces; runtime adapters selectively use supported operations.
set -eu
mode=${1:-write}
case "$mode" in write|check) ;; *) echo 'usage: generate-contracts.sh [write|check]' >&2; exit 2 ;; esac
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
find specs -name '*.openapi.yaml' -type f | LC_ALL=C sort > "$tmp/specs"
while IFS= read -r spec; do
 version=$(basename "$(dirname "$spec")")
 version=${version%%.*}
 provider_root=$(dirname "$(dirname "$spec")")
 product=$(basename "$spec" .openapi.yaml)
 case "$provider_root:$product" in
  specs/meta/instagram:facebook-login|specs/meta/instagram:instagram-login|specs/meta/instagram:oauth|specs/meta/instagram:webhooks|specs/meta/instagram:upload) provider=instagram ;;
  specs/meta/facebook:pages|specs/meta/facebook:oauth|specs/meta/facebook:webhooks|specs/meta/facebook:upload) provider=facebook ;;
  specs/tiktok:oauth|specs/tiktok:display|specs/tiktok:posting|specs/tiktok:upload|specs/tiktok:webhooks) provider=tiktok ;;
  *) echo "Unknown contract provider or product: $spec" >&2; exit 1 ;;
 esac
 product=$(printf '%s' "$product" | tr -d '-')
 output="internal/contracts/$provider/$version/$product/api.gen.go"
 generated="$tmp/api.gen.go"
 cat > "$tmp/config.yaml" <<CONFIG
package: $product
generate:
  std-http-server: true
  models: true
output-options:
  skip-prune: true
output: $generated
CONFIG
 go run -mod=readonly -modfile=tools/go.mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config "$tmp/config.yaml" "$spec"
 if [ "$mode" = check ]; then
  cmp "$output" "$generated"
 else
  mkdir -p "$(dirname "$output")"
  cp "$generated" "$output"
 fi
done < "$tmp/specs"
