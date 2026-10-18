# Optional OpenTelemetry

Social Mockery works without a collector. Traces, metrics and OTLP logs are off
unless their individual standard `OTEL_*_EXPORTER` is `otlp`. JSON `log/slog`
logging stays available on stderr. No global tracer/meter provider is replaced;
components receive one telemetry instance through constructors.

## Local collector and Jaeger

```sh
docker compose -f dev/compose.telemetry.yaml up -d
OTEL_SERVICE_NAME=social-mockery \
OTEL_TRACES_EXPORTER=otlp \
OTEL_METRICS_EXPORTER=otlp \
OTEL_LOGS_EXPORTER=otlp \
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf \
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 \
OTEL_TRACES_SAMPLER=always_on \
OTEL_METRIC_EXPORT_INTERVAL=5000 \
go run ./cmd --seed profiles/example.yaml
```

Run a publishing request from the README. Open http://localhost:16686 and select
`social-mockery` in Jaeger. Metrics are available in Prometheus exposition format
at http://localhost:8889/metrics (a Prometheus server is optional). Correlated
logs remain on the simulator's stderr; with OTLP logs enabled, inspect them with
`docker compose -f dev/compose.telemetry.yaml logs collector`. Jaeger stores traces,
not logs or metrics. Data is ephemeral and this compose setup is for local use.
All published ports bind to localhost. Do not expose the development collector
or Jaeger publicly.

For gRPC use `OTEL_EXPORTER_OTLP_PROTOCOL=grpc` and
`OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317`. HTTP is the default protocol.
Signal-specific protocol and endpoint settings override generic settings.
The optional compose files have not been represented as a live-deployment test;
run the commands above to verify Docker availability in your environment.

## Configuration

| Variable | Behaviour |
| --- | --- |
| `OTEL_TRACES_EXPORTER` | `otlp` enables tracing; unset/`none` disables |
| `OTEL_METRICS_EXPORTER` | `otlp` enables metrics independently |
| `OTEL_LOGS_EXPORTER` | `otlp` adds OTLP export of sanitized slog records |
| `OTEL_SDK_DISABLED=true` | Disables all three exporters |
| `OTEL_SERVICE_NAME` | Service name, default `social-mockery` |
| `OTEL_RESOURCE_ATTRIBUTES` | SDK resource attributes, e.g. `service.version=0.1.0,deployment.environment.name=test` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | SDK exporter endpoint |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` (default) or `grpc` |
| `OTEL_EXPORTER_OTLP_{TRACES,METRICS,LOGS}_PROTOCOL` | Signal-specific transport override |
| `OTEL_EXPORTER_OTLP_{TRACES,METRICS,LOGS}_ENDPOINT` | Signal-specific URL; HTTP path must include `/v1/traces`, `/v1/metrics` or `/v1/logs` |
| `OTEL_EXPORTER_OTLP_HEADERS` | SDK exporter authentication; never logged by Social Mockery |
| `OTEL_EXPORTER_OTLP_CERTIFICATE` | SDK trusted CA bundle |
| `OTEL_TRACES_SAMPLER`, `OTEL_TRACES_SAMPLER_ARG` | Standard SDK sampling; e.g. `parentbased_traceidratio` and `0.1` |
| `OTEL_METRIC_EXPORT_INTERVAL` | SDK metrics export interval in milliseconds |

The official SDK also handles its signal-specific TLS, headers and endpoint
variables. Only OTLP exporters are supported. Other exporter names do not enable
export. Configuration is read once at startup; restart to change it. Keep secrets
out of resource attributes: those are intentionally exported operator metadata.

`OTEL_RESOURCE_ATTRIBUTES=service.version=...` sets a deployed build's version;
the binary's fallback version is `dev`.

## What is instrumented now

- HTTP server span with W3C `traceparent`/`tracestate` extraction. Send Posty's
  trace header with its prefixed simulator request to keep the same trace.
- Provider/route identification, generated Instagram operation dispatch and
  authentication/permission checks.
- Seed profile loading, administrative reset/clock operations, SQLite repository
  operations, container creation, publishing and reads.
- Low-cardinality HTTP request count/duration/status; simulation and database
  duration; publishing outcomes; expected provider rejection counts; configured
  virtual media-processing duration; active account/post gauges.

The media-processing histogram measures configured simulation seconds at
container creation, not wall-clock CPU time or an asynchronous video transcoder.
State gauges reflect stored aggregate counts at startup and after reset/publish.
SQLite spans cover repository operations/transactions, not individual SQL
statements. Request spans use real elapsed time; advancing the virtual clock does
not distort performance measurements.

Engagement, webhooks, live contract verification, named simulation profiles and
additional providers are future features. There are no placeholder spans or
zero-valued "success" counters pretending those operations ran. Instrument them
where their real implementations are added.

## Errors and privacy

Expected provider rejections are marked `social_mockery.outcome=simulated_error`
and do not become internal error spans. Internal failures set error status and
record a generic sanitized error. A future intentional injected 5xx must call
`telemetry.ExpectedFailure(ctx)` so its HTTP parent is also classified correctly.

No raw URL, query, header, payload, caption, account/post ID, token, SQL statement
or raw error string is recorded. Route names use `{id}` templates; unknown routes
and methods collapse to fixed labels. W3C baggage is not extracted or exported.
Metric attributes contain only fixed operation/provider/outcome/category,
supported version, normalized method/route and HTTP status.

Correlated slog records include `trace_id` and `span_id` only when their context
has a recording span. Use `InfoContext`/`DebugContext`; use fixed log messages and
only allowlisted diagnostic attributes. The log handler drops unrecognized
attributes, including credentials and payloads. Never interpolate private data
into log messages. Sampling off means normal logs remain but no recording-span
IDs are added. Operation/database diagnostics use debug level; the default
stderr handler is info level.

## Failure isolation and overhead

Tracing/logging use non-blocking 1,024-record queues and 256-record batches; full
queues drop telemetry rather than block API requests. Export work has a
three-second cap. Metrics have a 128-series limit per instrument (overflow is
aggregated), bounded labels, and periodic asynchronous export. Spans/logs also
have attribute limits. Disabled mode uses official no-op providers and creates
no exporter connections or background export workers. Normal logging and small
instrumentation costs remain.

Shutdown drains requests, then shuts down all enabled providers concurrently
within a five-second deadline. Provider shutdown flushes buffered records before
closing exporters. Backend failure cannot fail an API simulation, though records
may be dropped. An exporter configuration error emits a sanitized warning and
falls back to no-op observability; it does not prevent the simulator starting.

## Validation and benchmarks

```sh
go test ./...
go test -race ./...
go vet ./...
go test ./internal/telemetry -bench . -benchmem
```

Tests use official in-memory trace exporters/manual metric readers, a recording
log exporter and local-only failure exporters. They cover trace hierarchy,
propagation, errors, redaction, metrics, no-op mode, shutdown and backend outage.
No real Meta account or external collector is needed.

## Implementation references

The implementation was inspected against Buildkite Agent's
[span helpers](https://github.com/buildkite/agent/blob/main/tracetools/span.go),
[tracing setup](https://github.com/buildkite/agent/blob/main/internal/job/tracing.go),
[metrics](https://github.com/buildkite/agent/blob/main/metrics/metrics.go) and
[OTLP job logging](https://github.com/buildkite/agent/blob/main/internal/job/otlp_job_logger.go).
Adapted ideas include explicit no-op behaviour, signal-specific OTLP protocol
precedence, context-owned child spans, and redaction before correlated logging.
Unlike job-output logging, simulator OTLP logs are batched without backpressure.
No Buildkite-specific job fields or compatibility propagators are copied.

Compose pins correspond to official upstream releases:
[Collector 0.161.0](https://github.com/open-telemetry/opentelemetry-collector-releases/releases/tag/v0.161.0)
and [Jaeger 2.22.0](https://github.com/jaegertracing/jaeger/releases/tag/v2.22.0).
Container images have not been pulled or runtime-tested in the development
execution environment used to author this change.
