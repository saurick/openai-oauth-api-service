package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	klog "github.com/go-kratos/kratos/v2/log"
	httpx "github.com/go-kratos/kratos/v2/transport/http"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type captureLogEntry struct {
	level  string
	fields map[string]interface{}
}

type captureLogger struct {
	entries []captureLogEntry
}

func (l *captureLogger) Log(level klog.Level, keyvals ...interface{}) error {
	fields := make(map[string]interface{}, len(keyvals)/2)
	for i := 0; i+1 < len(keyvals); i += 2 {
		fields[fmt.Sprint(keyvals[i])] = keyvals[i+1]
	}
	l.entries = append(l.entries, captureLogEntry{
		level:  level.String(),
		fields: fields,
	})
	return nil
}

func (l *captureLogger) hasEntry(match func(entry captureLogEntry) bool) bool {
	for _, entry := range l.entries {
		if match(entry) {
			return true
		}
	}
	return false
}

type stubReadinessPinger struct {
	err error
}

func (p stubReadinessPinger) PingContext(context.Context) error {
	return p.err
}

func TestRegisterHealthRoutesHealthzOK(t *testing.T) {
	logger := &captureLogger{}
	srv := httpx.NewServer()
	registerHealthRoutes(srv, logger, sdktrace.NewTracerProvider(), nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Request-Id", "req-healthz")
	recorder := httptest.NewRecorder()
	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("healthz status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if body := strings.TrimSpace(recorder.Body.String()); body != "ok" {
		t.Fatalf("healthz body = %q, want %q", body, "ok")
	}

	if !logger.hasEntry(func(entry captureLogEntry) bool {
		return fmt.Sprint(entry.fields["operation"]) == "server.http.healthz" &&
			fmt.Sprint(entry.fields["status"]) == "200" &&
			fmt.Sprint(entry.fields["request_id"]) == "req-healthz"
	}) {
		t.Fatalf("expected completion log for healthz, got %+v", logger.entries)
	}
}

func TestRegisterHealthRoutesReadyzOK(t *testing.T) {
	logger := &captureLogger{}
	srv := httpx.NewServer()
	registerHealthRoutes(srv, logger, sdktrace.NewTracerProvider(), stubReadinessPinger{})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	recorder := httptest.NewRecorder()
	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("readyz status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if body := strings.TrimSpace(recorder.Body.String()); body != "ready" {
		t.Fatalf("readyz body = %q, want %q", body, "ready")
	}
}

func TestRegisterHealthRoutesReadyzFailureLogsStructuredWarning(t *testing.T) {
	logger := &captureLogger{}
	srv := httpx.NewServer()
	registerHealthRoutes(srv, logger, sdktrace.NewTracerProvider(), stubReadinessPinger{err: errors.New("dial tcp postgres: i/o timeout")})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	req.Header.Set("X-Request-Id", "req-readyz")
	recorder := httptest.NewRecorder()
	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if body := strings.TrimSpace(recorder.Body.String()); body != "postgres not ready" {
		t.Fatalf("readyz body = %q, want %q", body, "postgres not ready")
	}

	if !logger.hasEntry(func(entry captureLogEntry) bool {
		return entry.level == "WARN" &&
			fmt.Sprint(entry.fields["msg"]) == "dependency not ready" &&
			fmt.Sprint(entry.fields["operation"]) == "server.http.readyz" &&
			fmt.Sprint(entry.fields["component"]) == "postgres" &&
			fmt.Sprint(entry.fields["status"]) == "503" &&
			fmt.Sprint(entry.fields["request_id"]) == "req-readyz" &&
			fmt.Sprint(entry.fields["error"]) == "dial tcp postgres: i/o timeout" &&
			fmt.Sprint(entry.fields["trace_id"]) != ""
	}) {
		t.Fatalf("expected structured readiness warning log, got %+v", logger.entries)
	}
}
