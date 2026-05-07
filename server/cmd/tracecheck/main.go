// server/cmd/tracecheck/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	endpoint := getenv("OTLP_ENDPOINT", "127.0.0.1:4318")
	serviceName := getenv("TRACE_SERVICE_NAME", "openai-oauth-api-service-tracecheck")

	ctx := context.Background()

	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		panic(fmt.Errorf("create otlp exporter: %w", err))
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewSchemaless(
			semconv.ServiceNameKey.String(serviceName),
		)),
	)
	defer func() { _ = tp.Shutdown(ctx) }()
	otel.SetTracerProvider(tp)

	tracer := otel.Tracer("openai-oauth-api-service-tracecheck")

	ctx, span := tracer.Start(ctx, "test-span")
	span.SetAttributes(
		attribute.String("openai_oauth_api_service.env", "test"),
	)
	time.Sleep(500 * time.Millisecond)
	span.End()

	fmt.Println("trace sent, wait a few seconds then check your OTLP backend")

	// 等 flush
	time.Sleep(2 * time.Second)
}
