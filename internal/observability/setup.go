package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

// Setup initialises trace, metric, and log providers and returns a single
// shutdown function that flushes all three.
func Setup(ctx context.Context, serviceName, otlpEndpoint string) (func(context.Context) error, error) {
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes("",
			attribute.String("service.name", serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	tp, err := newTracerProvider(ctx, res, otlpEndpoint)
	if err != nil {
		return nil, err
	}

	mp, err := newMeterProvider(ctx, res, otlpEndpoint)
	if err != nil {
		return nil, err
	}

	lp, err := newLoggerProvider(ctx, res, otlpEndpoint)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context) error {
		var errs []error
		for _, fn := range []func(context.Context) error{
			tp.Shutdown,
			mp.Shutdown,
			lp.Shutdown,
		} {
			if err := fn(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return fmt.Errorf("observability shutdown: %v", errs)
		}
		return nil
	}, nil
}
