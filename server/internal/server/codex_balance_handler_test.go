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
	t.Setenv("CODEX_BALANCE_TIMEOUT_SECONDS", "5")
	t.Setenv("CODEX_AUTH_FILE", writeFakeCodexAuthFile(t))
	t.Setenv("CODEX_RATE_LIMIT_RESET_CREDITS_URL", fakeResetCreditsURL(t, http.StatusOK, `{
      "available_count": 1,
      "total_earned_count": 1,
      "credits": [{
        "id": "RateLimitResetCredit_secret",
        "reset_type": "codex_rate_limits",
        "status": "available",
        "granted_at": "2026-06-12T02:10:16.436947Z",
        "expires_at": "2026-07-12T02:10:16.436947Z",
        "profile_user_id": "Codex Team",
        "title": "Full reset (Weekly + 5 hr)"
      }]
    }`))

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
	resetCredits := body["rate_limit_reset_credits"].(map[string]any)
	if resetCredits["status"] != "ok" || resetCredits["available_count"] != float64(1) {
		t.Fatalf("unexpected reset credits summary: %+v", resetCredits)
	}
	resetItems := resetCredits["credits"].([]any)
	firstResetCredit := resetItems[0].(map[string]any)
	if firstResetCredit["title"] != "Full reset (Weekly + 5 hr)" {
		t.Fatalf("reset credit title = %v", firstResetCredit["title"])
	}
	if _, ok := firstResetCredit["id"]; ok {
		t.Fatalf("reset credit should not expose id: %+v", firstResetCredit)
	}
	if _, ok := firstResetCredit["profile_user_id"]; ok {
		t.Fatalf("reset credit should not expose profile_user_id: %+v", firstResetCredit)
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
	t.Setenv("CODEX_BALANCE_TIMEOUT_SECONDS", "5")
	t.Setenv("CODEX_BALANCE_CACHE_SECONDS", "60")
	t.Setenv("CODEX_AUTH_FILE", writeFakeCodexAuthFile(t))
	t.Setenv("CODEX_RATE_LIMIT_RESET_CREDITS_URL", fakeResetCreditsURL(t, http.StatusOK, `{
      "available_count": 3,
      "total_earned_count": 3,
      "credits": []
    }`))

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

func TestCodexBalanceRouteRetriesTransientRateLimitReadFailure(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "fake-codex")
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"initialize"'*)
      echo '{"id":1,"result":{}}'
      ;;
    *'"method":"account/rateLimits/read"'*)
      if [ -f "$CODEX_TEST_RATE_LIMIT_RETRY_STATE" ]; then
        echo '{"id":2,"result":{"rateLimits":{"limitId":"codex","primary":{"usedPercent":12},"credits":{"balance":"7"}}}}'
      else
        : > "$CODEX_TEST_RATE_LIMIT_RETRY_STATE"
        echo '{"id":2,"error":{"code":-32000,"message":"failed to fetch codex rate limits: error sending request"}}'
      fi
      exit 0
      ;;
  esac
done
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_APP_SERVER_BIN", bin)
	t.Setenv("CODEX_BALANCE_TIMEOUT_SECONDS", "5")
	t.Setenv("CODEX_TEST_RATE_LIMIT_RETRY_STATE", filepath.Join(t.TempDir(), "retry-state"))
	t.Setenv("CODEX_AUTH_FILE", writeFakeCodexAuthFile(t))
	t.Setenv("CODEX_RATE_LIMIT_RESET_CREDITS_URL", fakeResetCreditsURL(t, http.StatusOK, `{
      "available_count": 0,
      "total_earned_count": 0,
      "credits": []
    }`))

	handler := &codexBalanceHTTPHandler{log: klog.NewHelper(&captureLogger{})}
	req := httptest.NewRequest(http.MethodGet, "/public/codex/balance", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	credits := body["credits"].(map[string]any)
	if credits["balance"] != "7" {
		t.Fatalf("credits.balance = %v, want 7", credits["balance"])
	}
}

func TestCodexBalanceRouteKeepsBalanceWhenResetCreditsFail(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "fake-codex")
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"initialize"'*)
      echo '{"id":1,"result":{}}'
      ;;
    *'"method":"account/rateLimits/read"'*)
      echo '{"id":2,"result":{"rateLimits":{"limitId":"codex","primary":{"usedPercent":12},"credits":{"balance":"5"}}}}'
      exit 0
      ;;
  esac
done
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_APP_SERVER_BIN", bin)
	t.Setenv("CODEX_BALANCE_TIMEOUT_SECONDS", "5")
	t.Setenv("CODEX_AUTH_FILE", writeFakeCodexAuthFile(t))
	t.Setenv("CODEX_RATE_LIMIT_RESET_CREDITS_URL", fakeResetCreditsURL(t, http.StatusBadGateway, `{"error":"temporary"}`))

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
	resetCredits := body["rate_limit_reset_credits"].(map[string]any)
	if resetCredits["status"] != "unavailable" {
		t.Fatalf("reset credits status = %v, want unavailable", resetCredits["status"])
	}
	credits := body["credits"].(map[string]any)
	if credits["balance"] != "5" {
		t.Fatalf("credits.balance = %v, want 5", credits["balance"])
	}
}

func writeFakeCodexAuthFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(path, []byte(`{"tokens":{"access_token":"fake-access-token","refresh_token":"fake-refresh-token","account_id":"acct_test"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func fakeResetCreditsURL(t *testing.T, status int, body string) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("reset credits method = %s, want GET", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer fake-access-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		if got := r.Header.Get("ChatGPT-Account-Id"); got != "acct_test" {
			t.Fatalf("ChatGPT-Account-Id header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server.URL
}
