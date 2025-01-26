// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Taken from
// https://github.com/open-telemetry/opentelemetry-go/blob/ae7ac48ebfa354a1dd7bd0239fed5fb34077a19c/trace/noop.go
// https://github.com/open-telemetry/opentelemetry-go/blob/ae7ac48ebfa354a1dd7bd0239fed5fb34077a19c/trace/nonrecording.go

package otel

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
)

// nonRecordingSpan is a minimal implementation of a Span that wraps a
// SpanContext. It performs no operations other than to return the wrapped
// SpanContext.
type nonRecordingSpan struct {
	noopSpan

	sc trace.SpanContext
}

// SpanContext returns the wrapped SpanContext.
func (s nonRecordingSpan) SpanContext() trace.SpanContext { return s.sc }

// NewNoopTracerProvider returns an implementation of TracerProvider that
// performs no operations. The Tracer and Spans created from the returned
// TracerProvider also perform no operations.
//
// Deprecated: Use [go.opentelemetry.io/otel/trace/noop.NewTracerProvider]
// instead.
func NewNoopTracerProvider() trace.TracerProvider {
	return noopTracerProvider{}
}

type noopTracerProvider struct{ embedded.TracerProvider }

var _ trace.TracerProvider = noopTracerProvider{}

// Tracer returns noop implementation of Tracer.
func (p noopTracerProvider) Tracer(string, ...trace.TracerOption) trace.Tracer {
	return noopTracer{}
}

// noopTracer is an implementation of Tracer that performs no operations.
type noopTracer struct{ embedded.Tracer }

var _ trace.Tracer = noopTracer{}

// Start carries forward a non-recording Span, if one is present in the context, otherwise it
// creates a no-op Span.
func (t noopTracer) Start(ctx context.Context, name string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
	span := trace.SpanFromContext(ctx)
	if _, ok := span.(nonRecordingSpan); !ok {
		// span is likely already a noopSpan, but let's be sure
		span = NoopSpan
	}
	return trace.ContextWithSpan(ctx, span), span
}

// noopSpan is an implementation of Span that performs no operations.
type noopSpan struct{ embedded.Span }

var NoopSpan trace.Span = noopSpan{}

// SpanContext returns an empty span context.
func (noopSpan) SpanContext() trace.SpanContext { return trace.SpanContext{} }

// IsRecording always returns false.
func (noopSpan) IsRecording() bool { return false }

// SetStatus does nothing.
func (noopSpan) SetStatus(codes.Code, string) {}

// SetError does nothing.
func (noopSpan) SetError(bool) {}

// SetAttributes does nothing.
func (noopSpan) SetAttributes(...attribute.KeyValue) {}

// End does nothing.
func (noopSpan) End(...trace.SpanEndOption) {}

// RecordError does nothing.
func (noopSpan) RecordError(error, ...trace.EventOption) {}

// AddEvent does nothing.
func (noopSpan) AddEvent(string, ...trace.EventOption) {}

// AddLink does nothing.
func (noopSpan) AddLink(trace.Link) {}

// SetName does nothing.
func (noopSpan) SetName(string) {}

// TracerProvider returns a no-op TracerProvider.
func (noopSpan) TracerProvider() trace.TracerProvider { return noopTracerProvider{} }
