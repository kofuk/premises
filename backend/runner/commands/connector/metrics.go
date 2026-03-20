package connector

import (
	"github.com/kofuk/premises/backend/common/util"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	openCount  metric.Int64Counter
	closeCount metric.Int64Counter
	io         metric.Int64Counter
}

func NewMetrics() *Metrics {
	meter := otel.Meter("connector")

	return &Metrics{
		openCount: util.Must(meter.Int64Counter(
			"premises.runner.connector.open.count",
			metric.WithDescription("Total number of connections opened"),
			metric.WithUnit("{request}"),
		)),
		closeCount: util.Must(meter.Int64Counter(
			"premises.runner.connector.close.count",
			metric.WithDescription("Total number of connections closed"),
			metric.WithUnit("{request}"),
		)),
		io: util.Must(meter.Int64Counter(
			"premises.runner.connector.io",
			metric.WithDescription("Total number of bytes transferred"),
			metric.WithUnit("By"),
		)),
	}
}
