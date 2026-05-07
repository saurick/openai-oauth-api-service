package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	pkglogger "server/pkg/logger"

	klog "github.com/go-kratos/kratos/v2/log"
	httpx "github.com/go-kratos/kratos/v2/transport/http"
)

func TestRequestIDFilterPreservesIncomingHeader(t *testing.T) {
	logger := &captureLogger{}
	helper := klog.NewHelper(klog.With(logger, "request_id", pkglogger.RequestID()))
	srv := httpx.NewServer(httpx.Filter(RequestIDFilter()))
	srv.Handle("/request-id", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		helper.WithContext(r.Context()).Infow("msg", "request id observed")
		_, _ = w.Write([]byte(pkglogger.RequestIDFromContext(r.Context())))
	}))

	req := httptest.NewRequest(http.MethodGet, "/request-id", nil)
	req.Header.Set(requestIDHeader, "req-upstream-123")
	recorder := httptest.NewRecorder()
	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get(requestIDHeader); got != "req-upstream-123" {
		t.Fatalf("response request id = %q, want %q", got, "req-upstream-123")
	}
	if got := recorder.Body.String(); got != "req-upstream-123" {
		t.Fatalf("context request id = %q, want %q", got, "req-upstream-123")
	}
	if !logger.hasEntry(func(entry captureLogEntry) bool {
		return fmt.Sprint(entry.fields["msg"]) == "request id observed" &&
			fmt.Sprint(entry.fields["request_id"]) == "req-upstream-123"
	}) {
		t.Fatalf("expected log with propagated request_id, got %+v", logger.entries)
	}
}

func TestRequestIDFilterGeneratesHeaderWhenMissing(t *testing.T) {
	logger := &captureLogger{}
	helper := klog.NewHelper(klog.With(logger, "request_id", pkglogger.RequestID()))
	srv := httpx.NewServer(httpx.Filter(RequestIDFilter()))
	srv.Handle("/request-id", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		helper.WithContext(r.Context()).Infow("msg", "request id generated")
		_, _ = w.Write([]byte(pkglogger.RequestIDFromContext(r.Context())))
	}))

	req := httptest.NewRequest(http.MethodGet, "/request-id", nil)
	recorder := httptest.NewRecorder()
	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	requestID := recorder.Header().Get(requestIDHeader)
	if requestID == "" {
		t.Fatal("expected generated request id in response header")
	}
	if got := recorder.Body.String(); got != requestID {
		t.Fatalf("context request id = %q, want %q", got, requestID)
	}
	if !logger.hasEntry(func(entry captureLogEntry) bool {
		return fmt.Sprint(entry.fields["msg"]) == "request id generated" &&
			fmt.Sprint(entry.fields["request_id"]) == requestID
	}) {
		t.Fatalf("expected log with generated request_id, got %+v", logger.entries)
	}
}
