#!/bin/sh
# Generate standalone reference contracts; none are registered with the simulator.
set -eu
mode=${1:-write}
case "$mode" in write|check) ;; *) echo 'usage: generate-contracts.sh [write|check]' >&2; exit 2 ;; esac
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
find specs/meta/instagram -name '*.openapi.yaml' -type f | LC_ALL=C sort > "$tmp/specs"
while IFS= read -r spec; do
 version=$(basename "$(dirname "$spec")")
 version=${version%%.*}
 product=$(basename "$spec" .openapi.yaml | tr -d '-')
 case "$product" in facebooklogin|instagramlogin|oauth|webhooks|upload) ;; *) echo "Unknown contract product: $spec" >&2; exit 1 ;; esac
 output="internal/contracts/instagram/$version/$product/api.gen.go"
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
