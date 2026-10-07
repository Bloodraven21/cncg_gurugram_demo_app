package main

import (
	"context"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/your-org/o11y-starter/internal/config"
	"github.com/your-org/o11y-starter/internal/handler"
	"github.com/your-org/o11y-starter/internal/observability"
)

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdown, err := observability.Setup(ctx, cfg.ServiceName, cfg.OTLPEndpoint)
	if err != nil {
		slog.Error("failed to setup observability", "error", err)
		return
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(otelgin.Middleware(cfg.ServiceName))

	exampleHandler := handler.NewExample()
	engine.GET("/example", exampleHandler.Handle)
	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: engine,
	}

	go func() {
		slog.Info("server starting", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}
	if err := shutdown(shutdownCtx); err != nil {
		slog.Error("observability shutdown error", "error", err)
	}
}
