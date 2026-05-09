package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"server/internal/conf"

	httpx "github.com/go-kratos/kratos/v2/transport/http"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestOAuthStateRoundTripKeepsDynamicFrontendOrigin(t *testing.T) {
	payload := oauthStatePayload{
		FrontendOrigin: "http://localhost:5176",
		Next:           "/admin-keys",
		RedirectURI:    "http://127.0.0.1:8400/auth/oauth/callback",
		Nonce:          "nonce",
		ExpiresAt:      time.Now().Add(time.Minute).Unix(),
	}
	raw, err := signOAuthState(payload, []byte("secret"))
	if err != nil {
		t.Fatalf("signOAuthState err = %v", err)
	}
	got, err := verifyOAuthState(raw, []byte("secret"), time.Now())
	if err != nil {
		t.Fatalf("verifyOAuthState err = %v", err)
	}
	if got.FrontendOrigin != payload.FrontendOrigin || got.RedirectURI != payload.RedirectURI || got.Next != payload.Next {
		t.Fatalf("state = %+v, want %+v", got, payload)
	}
}

func TestOAuthStartUsesBackendCallbackAndRefererFrontendOrigin(t *testing.T) {
	t.Setenv("OAUTH_API_OAUTH_CLIENT_ID", "client-id")
	t.Setenv("OAUTH_API_OAUTH_CLIENT_SECRET", "client-secret")
	t.Setenv("OAUTH_API_OAUTH_AUTH_URL", "https://accounts.example.test/auth")
	t.Setenv("OAUTH_API_OAUTH_TOKEN_URL", "https://accounts.example.test/token")
	t.Setenv("OAUTH_API_OAUTH_USERINFO_URL", "https://accounts.example.test/userinfo")

	srv := httpx.NewServer()
	registerOAuthRoutes(
		srv,
		&captureLogger{},
		sdktrace.NewTracerProvider(),
		nil,
		&conf.Data{Auth: &conf.Data_Auth{JwtSecret: "state-secret"}},
	)

	req := httptest.NewRequest(http.MethodGet, "/auth/oauth/start?next=/admin-keys", nil)
	req.Host = "127.0.0.1:8400"
	req.Header.Set("Referer", "http://localhost:5176/admin-login")
	recorder := httptest.NewRecorder()
	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d body=%s", recorder.Code, http.StatusFound, recorder.Body.String())
	}
	location := recorder.Header().Get("Location")
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse location err = %v", err)
	}
	if parsed.Scheme != "https" || parsed.Host != "accounts.example.test" || parsed.Path != "/auth" {
		t.Fatalf("location = %s", location)
	}
	if got := parsed.Query().Get("redirect_uri"); got != "http://127.0.0.1:8400/auth/oauth/callback" {
		t.Fatalf("redirect_uri = %q", got)
	}

	state, err := verifyOAuthState(parsed.Query().Get("state"), []byte("state-secret"), time.Now())
	if err != nil {
		t.Fatalf("verify state err = %v", err)
	}
	if state.FrontendOrigin != "http://localhost:5176" {
		t.Fatalf("frontend origin = %q", state.FrontendOrigin)
	}
	if state.Next != "/admin-keys" {
		t.Fatalf("next = %q", state.Next)
	}
}

func TestOAuthFrontendOriginRejectsExternalWhenNotAllowlisted(t *testing.T) {
	if isAllowedFrontendOrigin("https://evil.example.test", "https://api.example.test", nil) {
		t.Fatalf("external origin should be rejected")
	}
	if !isAllowedFrontendOrigin("http://localhost:5177", "https://api.example.test", nil) {
		t.Fatalf("localhost dynamic origin should be allowed")
	}
	if !isAllowedFrontendOrigin("https://app.example.test", "https://api.example.test", []string{"https://app.example.test"}) {
		t.Fatalf("allowlisted origin should be allowed")
	}
	if isAllowedFrontendOrigin("javascript:alert(1)", "https://api.example.test", nil) {
		t.Fatalf("unsafe origin should be rejected")
	}
	if strings.Contains(safeNextPath("https://evil.example.test"), "evil") {
		t.Fatalf("absolute next path should be sanitized")
	}
}
