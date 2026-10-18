.PHONY: build run test race vet fmt fmt-check bench clean docker

build:
	CGO_ENABLED=1 go build -o bin/social-mockery ./cmd

run:
	CGO_ENABLED=1 go run ./cmd start --addr 127.0.0.1:3000 --db :memory: --seed profiles/example.yaml

test:
	CGO_ENABLED=1 go test ./...

race:
	CGO_ENABLED=1 go test -race ./...

vet:
	CGO_ENABLED=1 go vet ./...

fmt:
	gofmt -w $$(find cmd internal tools/cmd -name '*.go' -type f)

fmt-check:
	@test -z "$$(gofmt -l $$(find cmd internal tools/cmd -name '*.go' -type f))" || { echo 'Run make fmt'; exit 1; }

bench:
	CGO_ENABLED=1 go test -run '^$$' -bench . -benchmem ./...

clean:
	rm -rf bin tmp

docker:
	docker build -t social-mockery .

.PHONY: generate generate-check spec-validate
spec-validate:
	cd tools && go run -mod=readonly ./cmd/specvalidate ../specs/meta/instagram/v24.0/openapi.yaml ../specs/meta/instagram/v25.0/openapi.yaml

generate: spec-validate
	go run -mod=readonly -modfile=tools/go.mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config configs/openapi/instagram.yaml specs/meta/instagram/v25.0/openapi.yaml

generate-check:
	@tmp=$$(mktemp); cp internal/providers/meta/generated/api.gen.go $$tmp; trap 'cp $$tmp internal/providers/meta/generated/api.gen.go; rm -f $$tmp' EXIT; $(MAKE) generate && cmp $$tmp internal/providers/meta/generated/api.gen.go
