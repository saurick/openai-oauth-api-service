package main

import (
	"context"
	"math"
	"strings"
	"testing"

	"server/internal/conf"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestNormalizeTraceRatio(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input float64
		want  float64
	}{
		{name: "nan falls back to zero", input: math.NaN(), want: 0},
		{name: "negative clamps to zero", input: -0.5, want: 0},
		{name: "zero keeps zero", input: 0, want: 0},
		{name: "fraction keeps ratio", input: 0.25, want: 0.25},
		{name: "one keeps one", input: 1, want: 1},
		{name: "above one clamps to one", input: 3, want: 1},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeTraceRatio(tc.input); got != tc.want {
				t.Fatalf("normalizeTraceRatio(%v) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestOverrideDevServerPortsAppliesFixedOAuthBundle(t *testing.T) {
	t.Parallel()
	cfg := &conf.Server{
		Http: &conf.Server_HTTP{Addr: "0.0.0.0:8400"},
		Grpc: &conf.Server_GRPC{Addr: "[::]:9400"},
	}
	values := map[string]string{"DEV_HTTP_PORT": "8410", "DEV_GRPC_PORT": "9410"}
	if err := overrideDevServerPorts("./configs/dev/config.yaml", cfg, func(key string) string {
		return values[key]
	}); err != nil {
		t.Fatalf("overrideDevServerPorts returned error: %v", err)
	}
	if cfg.Http.Addr != "0.0.0.0:8410" || cfg.Grpc.Addr != "[::]:9410" {
		t.Fatalf("unexpected overridden addresses: %#v", cfg)
	}
}

func TestOverrideDevServerPortsRecognizesDevConfigDirectory(t *testing.T) {
	t.Parallel()
	cfg := &conf.Server{
		Http: &conf.Server_HTTP{Addr: "0.0.0.0:8400"},
		Grpc: &conf.Server_GRPC{Addr: "[::]:9400"},
	}
	values := map[string]string{"DEV_HTTP_PORT": "8410", "DEV_GRPC_PORT": "9410"}
	if err := overrideDevServerPorts("./server/configs/dev", cfg, func(key string) string {
		return values[key]
	}); err != nil {
		t.Fatalf("overrideDevServerPorts returned error: %v", err)
	}
	if cfg.Http.Addr != "0.0.0.0:8410" || cfg.Grpc.Addr != "[::]:9410" {
		t.Fatalf("development directory override was not applied: %#v", cfg)
	}
}

func TestOverrideDevServerPortsDoesNotChangeProduction(t *testing.T) {
	t.Parallel()
	cfg := &conf.Server{
		Http: &conf.Server_HTTP{Addr: "0.0.0.0:8400"},
		Grpc: &conf.Server_GRPC{Addr: "0.0.0.0:9400"},
	}
	if err := overrideDevServerPorts("./configs/prod/config.yaml", cfg, func(string) string { return "8500" }); err != nil {
		t.Fatalf("production config returned error: %v", err)
	}
	if cfg.Http.Addr != "0.0.0.0:8400" || cfg.Grpc.Addr != "0.0.0.0:9400" {
		t.Fatalf("production addresses changed: %#v", cfg)
	}
}

func TestOverrideDevServerPortsRejectsDuplicatePorts(t *testing.T) {
	t.Parallel()
	cfg := &conf.Server{
		Http: &conf.Server_HTTP{Addr: "0.0.0.0:8400"},
		Grpc: &conf.Server_GRPC{Addr: "0.0.0.0:9400"},
	}
	err := overrideDevServerPorts("./configs/dev/config.yaml", cfg, func(string) string { return "8500" })
	if err == nil || !strings.Contains(err.Error(), "duplicates") {
		t.Fatalf("expected duplicate development port error, got %v", err)
	}
}

func TestBuildTraceSamplerHonorsParentDecision(t *testing.T) {
	t.Parallel()

	traceID := oteltrace.TraceID{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	sampledParent := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     oteltrace.SpanID{2, 2, 2, 2, 2, 2, 2, 2},
		TraceFlags: oteltrace.FlagsSampled,
		Remote:     true,
	})
	unsampledParent := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  oteltrace.SpanID{3, 3, 3, 3, 3, 3, 3, 3},
		Remote:  true,
	})

	cases := []struct {
		name   string
		ratio  float64
		parent oteltrace.SpanContext
		want   sdktrace.SamplingDecision
	}{
		{
			name:   "zero ratio drops root spans",
			ratio:  0,
			parent: oteltrace.SpanContext{},
			want:   sdktrace.Drop,
		},
		{
			name:   "full ratio samples root spans",
			ratio:  1,
			parent: oteltrace.SpanContext{},
			want:   sdktrace.RecordAndSample,
		},
		{
			name:   "sampled parent still wins when local ratio is zero",
			ratio:  0,
			parent: sampledParent,
			want:   sdktrace.RecordAndSample,
		},
		{
			name:   "unsampled parent stays unsampled",
			ratio:  1,
			parent: unsampledParent,
			want:   sdktrace.Drop,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			if tc.parent.IsValid() {
				ctx = oteltrace.ContextWithSpanContext(ctx, tc.parent)
			}
			got := buildTraceSampler(tc.ratio).ShouldSample(sdktrace.SamplingParameters{
				ParentContext: ctx,
				TraceID:       traceID,
				Name:          "test-operation",
				Kind:          oteltrace.SpanKindServer,
			}).Decision
			if got != tc.want {
				t.Fatalf("buildTraceSampler(%v) decision = %v, want %v", tc.ratio, got, tc.want)
			}
		})
	}
}
