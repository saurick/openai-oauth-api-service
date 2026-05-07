package logger

import (
	"context"
	"testing"

	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestTraceValuers(t *testing.T) {
	traceID := oteltrace.TraceID{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	sampledCtx := oteltrace.ContextWithSpanContext(context.Background(), oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     oteltrace.SpanID{2, 2, 2, 2, 2, 2, 2, 2},
		TraceFlags: oteltrace.FlagsSampled,
	}))
	unsampledCtx := oteltrace.ContextWithSpanContext(context.Background(), oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  oteltrace.SpanID{3, 3, 3, 3, 3, 3, 3, 3},
	}))

	if got := TraceSampled()(sampledCtx); got != true {
		t.Fatalf("TraceSampled(sampled) = %v, want true", got)
	}
	if got := TraceSampled()(unsampledCtx); got != false {
		t.Fatalf("TraceSampled(unsampled) = %v, want false", got)
	}

	if got := TraceLinkID()(sampledCtx); got != traceID.String() {
		t.Fatalf("TraceLinkID(sampled) = %v, want %s", got, traceID.String())
	}
	if got := TraceLinkID()(unsampledCtx); got != "" {
		t.Fatalf("TraceLinkID(unsampled) = %v, want empty", got)
	}

	if got := TraceSampled()(context.Background()); got != false {
		t.Fatalf("TraceSampled(background) = %v, want false", got)
	}
	if got := TraceLinkID()(context.Background()); got != "" {
		t.Fatalf("TraceLinkID(background) = %v, want empty", got)
	}
}
