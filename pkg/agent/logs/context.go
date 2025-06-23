package logs

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// FromContext returns a logr.Logger from the context or an instance of the global logger
func FromContext(ctx context.Context) logr.Logger {
	l, err := logr.FromContext(ctx)
	if err != nil {
		return NewLogger()
	}
	return l
}

func NewLogger() logr.Logger {
	// We need to AllowZapFields to ensure the protobuf message is logged correctly as a json object.
	// For that to work we need to do logr.Info("message", zap.Object("key", protoMessage))
	// Which means we are passing zap.Field to the logr interface.
	return zapr.NewLoggerWithOptions(zap.L(), zapr.AllowZapFields(true))
}

// FromContextWithTrace returns a logr.Logger from the context, enriched with traceId and spanId if available
func FromContextWithTrace(ctx context.Context) logr.Logger {
	log := FromContext(ctx)
	span := trace.SpanFromContext(ctx)
	if span != nil {
		traceId := span.SpanContext().TraceID()
		spanId := span.SpanContext().SpanID()
		log = log.WithValues("traceId", traceId, "spanId", spanId)
	}
	return log
}
