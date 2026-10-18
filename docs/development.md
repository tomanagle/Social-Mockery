# Development

## Stack and layout

Go 1.26+; standard-library `net/http.ServeMux`, `log/slog`, `testing`, and `database/sql`. Runtime dependencies provide SQLite, YAML parsing, generated OpenAPI parameter binding, and the official OpenTelemetry SDK/exporters. The OpenAPI generator and validator live in a separate development-only Go module under `tools/`.

The project follows a compact `cmd` / `internal` layout inspired by GOTTH, without importing its UI stack:

- `cmd/main.go`: command-line entry point
- `internal/config`: startup configuration
- `internal/handlers`: local HTTP handlers
- `internal/middleware`: HTTP middleware
- `internal/store`: storage contracts
- `internal/store/db`: SQLite setup and schema
- `internal/store/dbstore`: SQL-backed storage
- `internal/providers/meta`: the supported Meta API subset
- `internal/profiles`: fixture profile loading
- `internal/simulation`: deterministic clock and simulator logic
- `profiles/example.yaml`: reproducible starting fixtures

SQLite requires CGO and a working C compiler. The race detector also requires CGO on supported platforms. A `CGO_ENABLED=0` build is not a supported server build.

## Commands

```sh
make dev      # Docker Compose; foreground server on http://localhost:3000
make dev-down # Stop and remove the development container
make fmt
make fmt-check
make vet
make test
make race
make build
make bench
```

The benchmark command runs whatever benchmarks the package tests define and reports allocations; it does not enforce a performance threshold. CI checks formatting, vet, race-enabled tests, a build, and a one-iteration benchmark smoke check.

`make run` starts the example fixture with an in-memory database. An optional `.air.toml` is provided for developers who already use Air: run `air` in the repository root. Air is a development utility, not a Go module dependency. Reloading restarts the server and resets its in-memory database.

## Startup flags

Running `social-mockery` directly starts the server with default settings. There are no subcommands.

```text
social-mockery
  --addr 127.0.0.1:3000
  --db :memory:
  --seed profiles/example.yaml
  --admin-token optional-local-secret
  --max-records 10000
```

Without `--seed`, a new database uses the built-in example fixtures and an existing database retains its state. Explicit `--seed` always replaces existing database state, so omit it when restarting a database you want to preserve. Reset state through `POST /admin/reset`; an empty body restores the startup seed, while a JSON/YAML body supplies a replacement profile. The admin token is optional but advisable whenever the service is reachable beyond one trusted local process.

## Docker

All Dockerfiles, Docker build ignore files, and Compose files live in `dev/`.
Run `make dev` with Docker and Docker Compose v2 installed. It builds the development
image, bind-mounts the repository at `/src`, and starts the example fixture server
at http://localhost:3000. Go module and build caches use named volumes. No host Go
installation is needed for this command; the image includes Go and a C compiler.
The first run downloads the Go dependencies. Stop it with Ctrl-C; `make dev-down`
removes the container and network but retains the dependency caches.

The container uses `go run`, so restart `make dev` to pick up source changes. Its
in-memory database resets on each restart. For host-side hot reload, use the Air
configuration described above. Host `make test`, `make vet`, and `make fmt` require
Go; they can also run inside the running development container:

```sh
docker compose -f dev/compose.yaml exec app make test vet fmt-check
# This formats the bind-mounted files on the host:
docker compose -f dev/compose.yaml exec app make fmt
```

The development image is separate from the production image built by `make docker`.

The multi-stage build uses the official Go image with a C compiler, then a glibc-based Debian runtime. The process runs as UID/GID 10001. The binary is dynamically linked; a `scratch` runtime is not appropriate.

```sh
docker build -f dev/Dockerfile -t social-mockery .
docker run --rm --name social-mockery \
  -p 127.0.0.1:3000:3000 social-mockery
```

The server binds `:3000` inside the container so port forwarding works; the published host port remains loopback-only. The default command uses an in-memory database, discarded when the process exits. To persist state, both mount a volume and explicitly select a file database:

```sh
docker volume create social-mockery-data
docker run --rm --name social-mockery \
  -p 127.0.0.1:3000:3000 \
  -v social-mockery-data:/data social-mockery \
  start --addr :3000 --db /data/social-mockery.db
```

A new named volume inherits `/data` ownership from the image. Existing volumes or host bind mounts must be writable by UID 10001. For admin protection, add `--admin-token local-secret` to the command above, or pass `-e SOCIAL_MOCKERY_ADMIN_TOKEN=local-secret` before the image name. Do not add `--seed` on subsequent starts unless you intend to replace the saved state.

## Test strategy

Tests should use isolated databases and fixtures, advance virtual time explicitly, and assert state transitions instead of sleeping. Useful cases include creating an image or Reel, publishing too early, publishing after processing, repeat publishing, invalid tokens or account ownership, token expiry, restart persistence, reset determinism, invalid profiles, and record bounds.

The simulator neither downloads media nor forwards provider traffic. Keep tests offline and use placeholder URLs.
