package rpc

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("github.com/kofuk/premises/runner/internal/rpc")
