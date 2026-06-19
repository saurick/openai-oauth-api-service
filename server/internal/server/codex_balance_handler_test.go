package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	klog "github.com/go-kratos/kratos/v2/log"
	httpx "github.com/go-kratos/kratos/v2/transport/http"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestCodexBalanceRouteReturnsRateLimitsWithoutAuth(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "fake-codex")
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"initialize"'*)
      echo '{"id":1,"result":{"userAgent":"fake","codexHome":"/tmp/codex","platformFamily":"unix","platformOs":"linux"}}'
      ;;
    *'"method":"account/rateLimits/read"'*)
      echo '{"id":2,"result":{"rateLimits":{"limitId":"codex","limitName":null,"primary":{"usedPercent":16,"windowDurationMins":300,"resetsAt":1779190000},"secondary":{"usedPercent":8,"windowDurationMins":10080,"resetsAt":1779590000},"credits":{"hasCredits":false,"unlimited":false,"balance":"0"},"planType":"prolite","rateLimitReachedType":null},"rateLimitsByLimitId":{"codex":{"limitId":"codex","limitName":null,"primary":{"usedPercent":16,"windowDurationMins":300,"resetsAt":1779190000},"secondary":{"usedPercent":8,"windowDurationMins":10080,"resetsAt":1779590000},"credits":{"hasCredits":false,"unlimited":false,"balance":"0"},"planType":"prolite","rateLimitReachedType":null}}}}'
      exit 0
      ;;
  esac
done
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_APP_SERVER_BIN", bin)
	t.Setenv("CODEX_BALANCE_TIMEOUT_SECONDS", "2")

	srv := httpx.NewServer()
	registerCodexBalanceRoutes(srv, &captureLogger{}, sdktrace.NewTracerProvider())

	req := httptest.NewRequest(http.MethodGet, "/public/codex/balance", nil)
	recorder := httptest.NewRecorder()
	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status field = %v, want ok", body["status"])
	}
	credits := body["credits"].(map[string]any)
	if credits["balance"] != "0" {
		t.Fatalf("credits.balance = %v, want 0", credits["balance"])
	}
	rateLimits := body["rate_limits"].(map[string]any)
	primary := rateLimits["primary"].(map[string]any)
	if primary["remaining_percent"] != float64(84) {
		t.Fatalf("remaining_percent = %v, want 84", primary["remaining_percent"])
	}
	if _, ok := body["rate_limits_by_limit_id"].(map[string]any)["codex"]; !ok {
		t.Fatalf("missing codex entry in rate_limits_by_limit_id: %+v", body["rate_limits_by_limit_id"])
	}
}

func TestCodexBalanceRouteRejectsNonGET(t *testing.T) {
	srv := httpx.NewServer()
	registerCodexBalanceRoutes(srv, &captureLogger{}, sdktrace.NewTracerProvider())

	req := httptest.NewRequest(http.MethodPost, "/public/codex/balance", nil)
	recorder := httptest.NewRecorder()
	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	if allow := recorder.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("Allow = %q, want GET", allow)
	}
}

func TestCodexBalanceRouteServesStaleCacheOnTemporaryFailure(t *testing.T) {
	successBin := filepath.Join(t.TempDir(), "fake-codex-success")
	successScript := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"initialize"'*)
      echo '{"id":1,"result":{}}'
      ;;
    *'"method":"account/rateLimits/read"'*)
      echo '{"id":2,"result":{"rateLimits":{"limitId":"codex","primary":{"usedPercent":12},"credits":{"balance":"9"}}}}'
      exit 0
      ;;
  esac
done
`
	if err := os.WriteFile(successBin, []byte(successScript), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_APP_SERVER_BIN", successBin)
	t.Setenv("CODEX_BALANCE_TIMEOUT_SECONDS", "2")
	t.Setenv("CODEX_BALANCE_CACHE_SECONDS", "60")

	handler := &codexBalanceHTTPHandler{log: klog.NewHelper(&captureLogger{})}
	req := httptest.NewRequest(http.MethodGet, "/public/codex/balance", nil)
	first := httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), first, req)
	if first.Code != http.StatusOK {
		t.Fatalf("initial status = %d, want %d; body=%s", first.Code, http.StatusOK, first.Body.String())
	}

	handler.mu.Lock()
	handler.cacheExpiresAt = time.Now().Add(-time.Second)
	handler.mu.Unlock()

	failBin := filepath.Join(t.TempDir(), "fake-codex-fail")
	failScript := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"initialize"'*)
      echo '{"id":1,"result":{}}'
      ;;
    *'"method":"account/rateLimits/read"'*)
      echo '{"id":2,"error":{"code":-32000,"message":"temporary upstream error"}}'
      exit 0
      ;;
  esac
done
`
	if err := os.WriteFile(failBin, []byte(failScript), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_APP_SERVER_BIN", failBin)

	second := httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), second, req)
	if second.Code != http.StatusOK {
		t.Fatalf("stale status = %d, want %d; body=%s", second.Code, http.StatusOK, second.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(second.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" || body["stale"] != true {
		t.Fatalf("stale body status/stale = %v/%v, body=%s", body["status"], body["stale"], second.Body.String())
	}
	credits := body["credits"].(map[string]any)
	if credits["balance"] != "9" {
		t.Fatalf("stale credits.balance = %v, want 9", credits["balance"])
	}
	if body["stale_reason"] != "codex_balance_query_failed" {
		t.Fatalf("stale_reason = %v", body["stale_reason"])
	}
}
