package otel

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func InitializeTracer(ctx context.Context) error {
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		// Silently disable tracing
		return nil
	}

	res, err := resource.Detect(ctx)
	if err != nil {
		return err
	}

	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)

	return nil
}

func TraceContextFromContext(ctx context.Context) string {
	var (
		tc      propagation.TraceContext
		carrier propagation.MapCarrier
	)
	tc.Inject(ctx, carrier)
	return carrier.Get("traceparent")
}

func ContextFromTraceContext(ctx context.Context, traceContext string) context.Context {
	var (
		tc      propagation.TraceContext
		carrier propagation.MapCarrier
	)
	carrier.Set("traceparent", traceContext)
	return tc.Extract(ctx, carrier)
}
