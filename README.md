# o11y_enabled_starter

A minimal, production-ready Go HTTP service with OpenTelemetry traces, metrics, and logs wired up out of the box. Clone it, rename the module, add your business logic.

## What's inside

```
o11y_enabled_starter/
├── cmd/server/main.go              # Entry point — wires everything together
├── internal/
│   ├── config/config.go            # Env-based config with defaults
│   ├── observability/
│   │   ├── setup.go                # Single Setup() call returns one shutdown func
│   │   ├── tracer.go               # OTLP gRPC trace exporter + propagator
│   │   ├── meter.go                # OTLP gRPC metric exporter (periodic reader)
│   │   └── logger.go               # slog → stdout (JSON) AND OTel collector
│   └── handler/
│       └── example.go              # Example GET /example handler — replace with your logic
├── deploy/
│   ├── docker-compose.yml          # app + otel-collector
│   └── otel-collector.yaml         # collector config: OTLP in, debug + prometheus out
├── Dockerfile
└── .env.example
```

## Observability stack

| Signal  | How                                      | Where it lands               |
|---------|------------------------------------------|------------------------------|
| Traces  | `otelgin` middleware + manual spans      | OTel collector → debug log   |
| Metrics | `otel.Meter` counters + histograms       | OTel collector → Prometheus  |
| Logs    | `slog` with `multiHandler`               | stdout (JSON) + OTel collector |

Logs go to **both** stdout and the OTel collector simultaneously. Pod logs and the collector both have the full log stream.

## Endpoints

| Endpoint    | Description                          |
|-------------|--------------------------------------|
| `GET /example` | Example handler with trace, metric, and log |
| `GET /healthz` | Returns `{"status": "ok"}`        |

## Getting started

### 1. Rename the module

```bash
# Replace with your own module path
find . -type f -name "*.go" | xargs sed -i 's|github.com/your-org/o11y-starter|github.com/you/your-service|g'
sed -i 's|github.com/your-org/o11y-starter|github.com/you/your-service|g' go.mod
```

### 2. Run locally with Docker Compose

```bash
cd deploy
docker compose up
```

### 3. Run locally without Docker

```bash
# Requires an OTLP collector on localhost:4317
cp .env.example .env
go mod tidy
go run ./cmd/server
```

### 4. Hit the service

```bash
curl localhost:8080/example
curl localhost:8080/healthz
```

## Environment variables

| Variable                       | Default         | Description                        |
|--------------------------------|-----------------|------------------------------------|
| `HTTP_ADDR`                    | `:8080`         | Address the HTTP server listens on |
| `OTEL_EXPORTER_OTLP_ENDPOINT`  | `localhost:4317`| gRPC endpoint of the OTel collector|
| `OTEL_SERVICE_NAME`            | `o11y-starter`  | Service name in all OTel signals   |
| `LOG_LEVEL`                    | `info`          | Minimum log level for stdout       |

## Adding your own handler

1. Create `internal/handler/yourhandler.go` following the pattern in `example.go`:
   - Pull `ctx` from `c.Request.Context()` and pass it to every downstream call
   - Start child spans with `otel.Tracer(...).Start(ctx, "span.name")`
   - Record metrics via `otel.Meter(...)`
   - Log with `slog.InfoContext(ctx, ...)` — trace IDs are correlated automatically
2. Register the route in `cmd/server/main.go`

## Shutdown

The service handles `SIGINT`/`SIGTERM` gracefully:
1. HTTP server drains in-flight requests (10 s timeout)
2. OTel providers flush buffered traces, metrics, and logs before exit
