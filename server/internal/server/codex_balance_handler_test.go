package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

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
