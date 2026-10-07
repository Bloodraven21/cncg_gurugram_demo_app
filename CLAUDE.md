# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run locally (requires an OTLP collector on localhost:4317)
cp .env.example .env
go run ./cmd/server

# Run with full observability stack (app + OTel collector)
cd deploy && docker compose up

# Build binary
go build -o /bin/server ./cmd/server

# Build Docker image
docker build -t o11y-starter .

# Tidy dependencies
go mod tidy

# Test endpoints
curl localhost:8080/example
curl localhost:8080/healthz
```

There is no test suite in this repo yet — `go test ./...` will find nothing.

## Architecture

This is a Go HTTP service (Gin) with OpenTelemetry traces, metrics, and logs all wired at startup and exported via OTLP gRPC to an OTel collector.

**Startup flow** (`cmd/server/main.go`):
1. `config.Load()` reads env vars into a `Config` struct.
2. `observability.Setup(ctx, serviceName, otlpEndpoint)` initialises all three OTel providers (trace, metric, log) and returns a single `shutdown` func that flushes all three.
3. Gin engine is created with `otelgin` middleware for automatic per-request spans.
4. On `SIGINT`/`SIGTERM`: HTTP server drains (10 s), then `shutdown()` flushes OTel buffers.

**Observability package** (`internal/observability/`):
- `setup.go` — the single public entry point; composes tracer, meter, and logger providers and returns one shutdown func.
- `tracer.go` — OTLP gRPC trace exporter; registers the global `otel.TracerProvider` and a `TraceContext`+`Baggage` propagator.
- `meter.go` — OTLP gRPC metric exporter with a periodic reader; registers the global `otel.MeterProvider`.
- `logger.go` — OTLP gRPC log exporter; also installs a `multiHandler` on `slog.Default` that fans records to **both** stdout (JSON) and the OTel collector simultaneously.

**Handler pattern** (`internal/handler/example.go`):
- Initialise `metric.Int64Counter` and `metric.Float64Histogram` in the constructor via `otel.Meter(...)`.
- In `Handle`: pull `ctx` from `c.Request.Context()`, start a child span via `otel.Tracer(...).Start(ctx, "span.name")`, log with `slog.InfoContext(ctx, ...)` (trace ID is correlated automatically), record metrics, then call `span.End()` via defer.

**Deploy** (`deploy/`):
- `docker-compose.yml` — runs the app and `otel/opentelemetry-collector-contrib` together; the collector listens on `:4317` (gRPC) and `:4318` (HTTP), exposes Prometheus scrape at `:8889`, and health-check at `:13133`.
- `otel-collector.yaml` — OTLP receiver → batch processor → `debug` exporter (traces/logs) and `prometheus` exporter (metrics).
- `deploy/helm/o11y-starter/` — Helm chart for Kubernetes with separate `values-minikube.yaml` and `values-eks.yaml`.

## Adding a handler

1. Create `internal/handler/yourhandler.go` following `example.go`.
2. Register the route in `cmd/server/main.go`.
3. Always pass `ctx` from `c.Request.Context()` downstream — this carries the trace context.

## Renaming the module

```bash
find . -type f -name "*.go" | xargs sed -i 's|github.com/your-org/o11y-starter|github.com/you/your-service|g'
sed -i 's|github.com/your-org/o11y-starter|github.com/you/your-service|g' go.mod
```
