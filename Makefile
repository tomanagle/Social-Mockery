.PHONY: build run dev dev-down test race vet fmt fmt-check bench clean docker

build:
	CGO_ENABLED=1 go build -o bin/social-mockery ./cmd

run:
	CGO_ENABLED=1 go run ./cmd --addr 127.0.0.1:3000 --db :memory: --seed profiles/example.yaml

dev:
	docker compose -f dev/compose.yaml up --build

dev-down:
	docker compose -f dev/compose.yaml down

test:
	CGO_ENABLED=1 go test ./...
	cd tools && CGO_ENABLED=1 go test ./...

race:
	CGO_ENABLED=1 go test -race ./...
	cd tools && CGO_ENABLED=1 go test -race ./...

vet:
	CGO_ENABLED=1 go vet ./...
	cd tools && CGO_ENABLED=1 go vet ./...

fmt:
	gofmt -w $$(find cmd internal tools/cmd -name '*.go' -type f)

fmt-check:
	@test -z "$$(gofmt -l $$(find cmd internal tools/cmd -name '*.go' -type f))" || { echo 'Run make fmt'; exit 1; }

bench:
	CGO_ENABLED=1 go test -run '^$$' -bench . -benchmem ./...

clean:
	rm -rf bin tmp

docker:
	docker build -f dev/Dockerfile -t social-mockery .

.PHONY: generate generate-check spec-validate
SPEC_FILES := $(sort $(wildcard specs/*/*/*/openapi.yaml specs/*/*/*/*.openapi.yaml))
GENERATOR_CONFIGS := $(sort $(wildcard configs/openapi/*.yaml))
spec-validate:
	cd tools && go run -mod=readonly ./cmd/specvalidate $(addprefix ../,$(SPEC_FILES))

generate: spec-validate
	@for config in $(GENERATOR_CONFIGS); do \
		version=$$(basename "$$config" .yaml | sed 's/^instagram-//'); \
		go run -mod=readonly -modfile=tools/go.mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config "$$config" "specs/meta/instagram/$$version/openapi.yaml" || exit 1; \
	done
	@sh scripts/generate-contracts.sh write

generate-check: spec-validate
	@set -eu; tmp=$$(mktemp -d); trap 'rm -rf "$$tmp"' EXIT; \
	for config in $(GENERATOR_CONFIGS); do \
		version=$$(basename "$$config" .yaml | sed 's/^instagram-//'); \
		package=$${version%%.*}; \
		sed "s|^output:.*|output: $$tmp/$$package.go|" "$$config" > "$$tmp/config.yaml"; \
		go run -mod=readonly -modfile=tools/go.mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config "$$tmp/config.yaml" "specs/meta/instagram/$$version/openapi.yaml"; \
		cmp "internal/providers/meta/generated/$$package/api.gen.go" "$$tmp/$$package.go"; \
	done

	@sh scripts/generate-contracts.sh check
