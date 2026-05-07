package server

import (
	"context"
	"fmt"
	stdhttp "net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	httpx "github.com/go-kratos/kratos/v2/transport/http"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type readinessPinger interface {
	PingContext(ctx context.Context) error
}

type statusCapturingResponseWriter struct {
	stdhttp.ResponseWriter
	status int
}

func (w *statusCapturingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusCapturingResponseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = stdhttp.StatusOK
	}
	return w.ResponseWriter.Write(p)
}

func (w *statusCapturingResponseWriter) StatusCode() int {
	if w.status == 0 {
		return stdhttp.StatusOK
	}
	return w.status
}

func requestIDFromRequest(r *stdhttp.Request) string {
	if r == nil {
		return ""
	}
	if requestID := r.Header.Get("X-Request-Id"); requestID != "" {
		return requestID
	}
	return r.Header.Get("X-Request-ID")
}

func traceIDFromContext(ctx context.Context) string {
	spanCtx := oteltrace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		return spanCtx.TraceID().String()
	}
	return ""
}

func spanIDFromContext(ctx context.Context) string {
	spanCtx := oteltrace.SpanContextFromContext(ctx)
	if spanCtx.HasSpanID() {
		return spanCtx.SpanID().String()
	}
	return ""
}

func writePlainText(w stdhttp.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// 统一给自定义 HTTP handler 补 trace、recover 和结构化收尾日志，避免健康检查与静态路由成为观测盲区。
func newObservedHTTPHandler(
	logger log.Logger,
	tp *sdktrace.TracerProvider,
	operation string,
	handler func(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request),
) stdhttp.Handler {
	helper := log.NewHelper(log.With(logger, "logger.name", "server.http.custom"))
	tracer := otel.Tracer("server.http.custom")
	if tp != nil {
		tracer = tp.Tracer("server.http.custom")
	}

	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		baseCtx := r.Context()
		baseCtx = otel.GetTextMapPropagator().Extract(baseCtx, propagation.HeaderCarrier(r.Header))
		httpx.SetOperation(baseCtx, operation)

		ctx, span := tracer.Start(baseCtx, operation, oteltrace.WithSpanKind(oteltrace.SpanKindServer))
		defer span.End()

		recorder := &statusCapturingResponseWriter{ResponseWriter: w}
		req := r.WithContext(ctx)
		requestID := requestIDFromRequest(req)
		start := time.Now()
		var panicErr error

		defer func() {
			status := recorder.StatusCode()
			duration := time.Since(start)

			span.SetAttributes(
				attribute.String("http.method", req.Method),
				attribute.String("http.path", req.URL.Path),
				attribute.Int("http.status_code", status),
			)
			if requestID != "" {
				span.SetAttributes(attribute.String("http.request_id", requestID))
			}

			if panicErr != nil {
				span.RecordError(panicErr)
				span.SetStatus(codes.Error, panicErr.Error())
				helper.WithContext(ctx).Errorw(
					"msg", "custom http handler panic",
					"operation", operation,
					"method", req.Method,
					"path", req.URL.Path,
					"status", status,
					"duration", duration.String(),
					"request_id", requestID,
					"trace_id", traceIDFromContext(ctx),
					"span_id", spanIDFromContext(ctx),
					"error", panicErr.Error(),
				)
				return
			}

			if status >= stdhttp.StatusBadRequest {
				span.SetStatus(codes.Error, stdhttp.StatusText(status))
			} else {
				span.SetStatus(codes.Ok, "OK")
			}

			helper.WithContext(ctx).Debugw(
				"msg", "custom http handler completed",
				"operation", operation,
				"method", req.Method,
				"path", req.URL.Path,
				"status", status,
				"duration", duration.String(),
				"request_id", requestID,
				"trace_id", traceIDFromContext(ctx),
				"span_id", spanIDFromContext(ctx),
			)
		}()

		defer func() {
			if recovered := recover(); recovered != nil {
				panicErr = fmt.Errorf("panic recovered: %v", recovered)
				if recorder.status == 0 {
					writePlainText(recorder, stdhttp.StatusInternalServerError, stdhttp.StatusText(stdhttp.StatusInternalServerError))
				}
			}
		}()

		handler(ctx, recorder, req)
	})
}

func registerHealthRoutes(srv *httpx.Server, logger log.Logger, tp *sdktrace.TracerProvider, postgres readinessPinger) {
	healthLogger := log.NewHelper(log.With(logger, "logger.name", "server.http.health"))

	srv.Handle("/ping", newObservedHTTPHandler(logger, tp, "server.http.ping", func(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request) {
		writePlainText(w, stdhttp.StatusOK, "pong")
	}))

	srv.Handle("/healthz", newObservedHTTPHandler(logger, tp, "server.http.healthz", func(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request) {
		writePlainText(w, stdhttp.StatusOK, "ok")
	}))

	srv.Handle("/readyz", newObservedHTTPHandler(logger, tp, "server.http.readyz", func(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if postgres != nil {
			if err := postgres.PingContext(ctx); err != nil {
				// 关键兜底：项目基线 readiness 只检查 Postgres 这一项通用硬依赖，避免把后续业务特有依赖预埋进来。
				healthLogger.WithContext(ctx).Warnw(
					"msg", "dependency not ready",
					"operation", "server.http.readyz",
					"component", "postgres",
					"status", stdhttp.StatusServiceUnavailable,
					"request_id", requestIDFromRequest(r),
					"trace_id", traceIDFromContext(ctx),
					"error", err.Error(),
				)
				writePlainText(w, stdhttp.StatusServiceUnavailable, "postgres not ready")
				return
			}
		}

		writePlainText(w, stdhttp.StatusOK, "ready")
	}))
}

func registerStaticHandler(srv *httpx.Server, logger log.Logger, tp *sdktrace.TracerProvider) {
	// 优先用环境变量 STATIC_DIR，没有的话默认 /app/public（容器内）。
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "/app/public"
	}

	if fi, err := os.Stat(staticDir); err == nil && fi.IsDir() {
		log.Infof("http static dir enabled: %s", staticDir)
		fileServer := stdhttp.FileServer(stdhttp.Dir(staticDir))

		srv.HandlePrefix("/", newObservedHTTPHandler(logger, tp, "server.http.static", func(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request) {
			path := r.URL.Path
			if path == "" || path == "/" {
				fileServer.ServeHTTP(w, r)
				return
			}

			fp := filepath.Join(staticDir, filepath.Clean(path))
			if fi, err := os.Stat(fp); err == nil && !fi.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}

			indexPath := filepath.Join(staticDir, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				stdhttp.ServeFile(w, r, indexPath)
				return
			}

			stdhttp.NotFound(w, r)
		}))
		return
	}

	log.Infof("http static dir not found or not dir: %s, skip static handler", staticDir)
}
