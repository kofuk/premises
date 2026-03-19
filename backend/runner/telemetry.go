package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kofuk/premises/backend/common"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"golang.org/x/sync/errgroup"
)

type instrumentation struct {
	tp *trace.TracerProvider
	mp *metric.MeterProvider
	lp *log.LoggerProvider
}

func initTracerProvider(ctx context.Context, resource *resource.Resource, otlpEndpoint string) *trace.TracerProvider {
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpointURL(otlpEndpoint))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize tracer provider: %v\n", err)
		return nil
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource),
		trace.WithSampler(trace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)

	return tp
}

func initMeterProvider(ctx context.Context, resource *resource.Resource, otlpEndpoint string, exportIntervalMs int) *metric.MeterProvider {
	exporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithEndpointURL(otlpEndpoint))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize meter provider: %v\n", err)
		return nil
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter, metric.WithInterval(time.Duration(exportIntervalMs)*time.Millisecond))),
		metric.WithResource(resource),
	)
	otel.SetMeterProvider(mp)

	return mp
}

func initLoggerProvider(ctx context.Context, resource *resource.Resource, otlpEndpoint string) *log.LoggerProvider {
	exporter, err := otlploggrpc.New(ctx, otlploggrpc.WithEndpointURL(otlpEndpoint))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger provider: %v\n", err)
		return nil
	}

	lp := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(exporter)),
		log.WithResource(resource),
	)

	global.SetLoggerProvider(lp)

	return lp
}

func initInstrumentation(ctx context.Context, serviceName, otlpEndpoint string, metricsExportIntervalMs int) *instrumentation {
	if otlpEndpoint == "" {
		exporter, _ := stdoutlog.New()
		lp := log.NewLoggerProvider(
			log.WithProcessor(log.NewBatchProcessor(exporter)),
		)
		global.SetLoggerProvider(lp)

		return &instrumentation{
			lp: lp,
		}
	}

	res := resource.NewWithAttributes(semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(common.Version),
	)

	return &instrumentation{
		tp: initTracerProvider(ctx, res, otlpEndpoint),
		mp: initMeterProvider(ctx, res, otlpEndpoint, metricsExportIntervalMs),
		lp: initLoggerProvider(ctx, res, otlpEndpoint),
	}
}

func (i *instrumentation) shutdown(ctx context.Context) {
	eg, ctx := errgroup.WithContext(ctx)
	if i.tp != nil {
		eg.Go(func() error {
			return i.tp.Shutdown(ctx)
		})
	}
	if i.mp != nil {
		eg.Go(func() error {
			return i.mp.Shutdown(ctx)
		})
	}
	if i.lp != nil {
		eg.Go(func() error {
			return i.lp.Shutdown(ctx)
		})
	}
	eg.Wait()
}
