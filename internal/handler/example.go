package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Example struct {
	requests metric.Int64Counter
	latency  metric.Float64Histogram
}

func NewExample() *Example {
	meter := otel.Meter("o11y-starter")

	requests, _ := meter.Int64Counter("example_requests_total",
		metric.WithDescription("Total number of /example requests"),
	)
	latency, _ := meter.Float64Histogram("example_latency_seconds",
		metric.WithDescription("Duration of /example handler"),
		metric.WithUnit("s"),
	)

	return &Example{requests: requests, latency: latency}
}

func (e *Example) Handle(c *gin.Context) {
	ctx := c.Request.Context()
	start := time.Now()

	// Create a child span for any work done inside this handler.
	tracer := otel.Tracer("o11y-starter")
	ctx, span := tracer.Start(ctx, "example.handle",
		trace.WithAttributes(attribute.String("http.method", c.Request.Method)),
	)
	defer span.End()

	// Replace this block with your actual business logic.
	slog.InfoContext(ctx, "example request received", "path", c.Request.URL.Path)

	elapsed := time.Since(start).Seconds()
	e.requests.Add(ctx, 1)
	e.latency.Record(ctx, elapsed)

	c.JSON(http.StatusOK, gin.H{
		"message":   "pong",
		"timestamp": time.Now().UTC(),
	})
}
