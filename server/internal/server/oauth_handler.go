package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	stdhttp "net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"server/internal/biz"
	"server/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	httpx "github.com/go-kratos/kratos/v2/transport/http"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	oauthStateTTL     = 10 * time.Minute
	oauthCallbackPath = "/auth/oauth/callback"
)

type oauthRuntimeConfig struct {
	Enabled                bool
	Provider               string
	ClientID               string
	ClientSecret           string
	AuthURL                string
	TokenURL               string
	UserInfoURL            string
	Scopes                 []string
	AllowedFrontendOrigins []string
}

type oauthStatePayload struct {
	FrontendOrigin string `json:"frontend_origin"`
	Next           string `json:"next"`
	RedirectURI    string `json:"redirect_uri"`
	Nonce          string `json:"nonce"`
	ExpiresAt      int64  `json:"expires_at"`
}

type oauthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	IDToken     string `json:"id_token"`
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

type oauthUserInfoResponse struct {
	Subject           string `json:"sub"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	Error             string `json:"error"`
	Description       string `json:"error_description"`
}

func registerOAuthRoutes(
	srv *httpx.Server,
	logger log.Logger,
	tp *sdktrace.TracerProvider,
	adminAuthUC *biz.AdminAuthUsecase,
	dc *conf.Data,
) {
	handler := &oauthHTTPHandler{
		log:         log.NewHelper(log.With(logger, "module", "server.oauth")),
		adminAuthUC: adminAuthUC,
		dataCfg:     dc,
		httpClient:  &stdhttp.Client{Timeout: 10 * time.Second},
		now:         time.Now,
	}

	srv.Handle("/auth/oauth/config", newObservedHTTPHandler(logger, tp, "server.http.oauth_config", handler.config))
	srv.Handle("/auth/oauth/start", newObservedHTTPHandler(logger, tp, "server.http.oauth_start", handler.start))
	srv.Handle(oauthCallbackPath, newObservedHTTPHandler(logger, tp, "server.http.oauth_callback", handler.callback))
}

type oauthHTTPHandler struct {
	log         *log.Helper
	adminAuthUC *biz.AdminAuthUsecase
	dataCfg     *conf.Data
	httpClient  *stdhttp.Client
	now         func() time.Time
}

func currentOAuthRuntimeConfig() oauthRuntimeConfig {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("OAUTH_API_OAUTH_PROVIDER")))
	if provider == "" {
		provider = "google"
	}

	cfg := oauthRuntimeConfig{
		Provider:               provider,
		ClientID:               strings.TrimSpace(os.Getenv("OAUTH_API_OAUTH_CLIENT_ID")),
		ClientSecret:           strings.TrimSpace(os.Getenv("OAUTH_API_OAUTH_CLIENT_SECRET")),
		AuthURL:                strings.TrimSpace(os.Getenv("OAUTH_API_OAUTH_AUTH_URL")),
		TokenURL:               strings.TrimSpace(os.Getenv("OAUTH_API_OAUTH_TOKEN_URL")),
		UserInfoURL:            strings.TrimSpace(os.Getenv("OAUTH_API_OAUTH_USERINFO_URL")),
		Scopes:                 splitEnvList(os.Getenv("OAUTH_API_OAUTH_SCOPES")),
		AllowedFrontendOrigins: splitEnvList(os.Getenv("OAUTH_API_OAUTH_ALLOWED_FRONTEND_ORIGINS")),
	}

	if provider == "google" {
		if cfg.AuthURL == "" {
			cfg.AuthURL = "https://accounts.google.com/o/oauth2/v2/auth"
		}
		if cfg.TokenURL == "" {
			cfg.TokenURL = "https://oauth2.googleapis.com/token"
		}
		if cfg.UserInfoURL == "" {
			cfg.UserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"
		}
	}
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{"openid", "email", "profile"}
	}

	cfg.Enabled = cfg.ClientID != "" && cfg.ClientSecret != "" && cfg.AuthURL != "" && cfg.TokenURL != "" && cfg.UserInfoURL != ""
	return cfg
}

func splitEnvList(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t'
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if v := strings.TrimSpace(field); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func (h *oauthHTTPHandler) stateSecret() []byte {
	if h.dataCfg == nil || h.dataCfg.Auth == nil || strings.TrimSpace(h.dataCfg.Auth.JwtSecret) == "" {
		return nil
	}
	return []byte(h.dataCfg.Auth.JwtSecret)
}

func (h *oauthHTTPHandler) config(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request) {
	cfg := currentOAuthRuntimeConfig()
	writeJSON(w, stdhttp.StatusOK, map[string]any{
		"enabled":  cfg.Enabled,
		"provider": cfg.Provider,
	})
}

func (h *oauthHTTPHandler) start(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request) {
	cfg := currentOAuthRuntimeConfig()
	if !cfg.Enabled {
		writeJSON(w, stdhttp.StatusNotFound, map[string]any{"error": "oauth_not_configured"})
		return
	}
	secret := h.stateSecret()
	if len(secret) == 0 {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]any{"error": "oauth_state_secret_missing"})
		return
	}

	requestOrigin := requestExternalOrigin(r)
	frontendOrigin := requestedFrontendOrigin(r, requestOrigin)
	if !isAllowedFrontendOrigin(frontendOrigin, requestOrigin, cfg.AllowedFrontendOrigins) {
		h.log.WithContext(ctx).Warnw(
			"msg", "oauth frontend origin rejected",
			"frontend_origin", frontendOrigin,
			"request_origin", requestOrigin,
			"request_id", requestIDFromRequest(r),
			"trace_id", traceIDFromContext(ctx),
		)
		writeJSON(w, stdhttp.StatusBadRequest, map[string]any{"error": "invalid_frontend_origin"})
		return
	}

	redirectURI := strings.TrimRight(requestOrigin, "/") + oauthCallbackPath
	payload := oauthStatePayload{
		FrontendOrigin: frontendOrigin,
		Next:           safeNextPath(r.URL.Query().Get("next")),
		RedirectURI:    redirectURI,
		Nonce:          randomNonce(),
		ExpiresAt:      h.now().Add(oauthStateTTL).Unix(),
	}
	state, err := signOAuthState(payload, secret)
	if err != nil {
		h.log.WithContext(ctx).Errorf("sign oauth state failed: %v", err)
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]any{"error": "oauth_state_failed"})
		return
	}

	authURL, err := url.Parse(cfg.AuthURL)
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]any{"error": "oauth_auth_url_invalid"})
		return
	}
	q := authURL.Query()
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(cfg.Scopes, " "))
	q.Set("state", state)
	authURL.RawQuery = q.Encode()

	stdhttp.Redirect(w, r, authURL.String(), stdhttp.StatusFound)
}

func (h *oauthHTTPHandler) callback(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request) {
	cfg := currentOAuthRuntimeConfig()
	if !cfg.Enabled {
		writeJSON(w, stdhttp.StatusNotFound, map[string]any{"error": "oauth_not_configured"})
		return
	}
	if h.adminAuthUC == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]any{"error": "oauth_admin_auth_unavailable"})
		return
	}

	state, err := verifyOAuthState(r.URL.Query().Get("state"), h.stateSecret(), h.now())
	if err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]any{"error": "invalid_oauth_state"})
		return
	}
	if oauthErr := strings.TrimSpace(r.URL.Query().Get("error")); oauthErr != "" {
		h.redirectOAuthError(w, r, state, oauthErr)
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		h.redirectOAuthError(w, r, state, "missing_code")
		return
	}

	token, err := h.exchangeOAuthCode(ctx, cfg, code, state.RedirectURI)
	if err != nil {
		h.log.WithContext(ctx).Warnw("msg", "oauth token exchange failed", "error", err.Error(), "trace_id", traceIDFromContext(ctx))
		h.redirectOAuthError(w, r, state, "token_exchange_failed")
		return
	}
	userInfo, err := h.fetchOAuthUserInfo(ctx, cfg, token.AccessToken)
	if err != nil {
		h.log.WithContext(ctx).Warnw("msg", "oauth userinfo failed", "error", err.Error(), "trace_id", traceIDFromContext(ctx))
		h.redirectOAuthError(w, r, state, "userinfo_failed")
		return
	}

	appToken, expireAt, admin, err := h.adminAuthUC.LoginWithOAuth(ctx, biz.OAuthIdentity{
		Provider:          cfg.Provider,
		Subject:           userInfo.Subject,
		Email:             userInfo.Email,
		Name:              userInfo.Name,
		PreferredUsername: userInfo.PreferredUsername,
	})
	if err != nil {
		h.log.WithContext(ctx).Warnw("msg", "oauth admin login failed", "provider", cfg.Provider, "email", userInfo.Email, "error", err.Error(), "trace_id", traceIDFromContext(ctx))
		h.redirectOAuthError(w, r, state, "admin_not_allowed")
		return
	}

	target := buildOAuthSuccessURL(state, appToken, expireAt, admin)
	stdhttp.Redirect(w, r, target, stdhttp.StatusFound)
}

func (h *oauthHTTPHandler) exchangeOAuthCode(ctx context.Context, cfg oauthRuntimeConfig, code, redirectURI string) (*oauthTokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)

	req, err := stdhttp.NewRequestWithContext(ctx, stdhttp.MethodPost, cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var token oauthTokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || token.AccessToken == "" || token.Error != "" {
		return nil, fmt.Errorf("token endpoint status=%d error=%s", resp.StatusCode, token.Error)
	}
	return &token, nil
}

func (h *oauthHTTPHandler) fetchOAuthUserInfo(ctx context.Context, cfg oauthRuntimeConfig, accessToken string) (*oauthUserInfoResponse, error) {
	if accessToken == "" {
		return nil, errors.New("missing access token")
	}
	req, err := stdhttp.NewRequestWithContext(ctx, stdhttp.MethodGet, cfg.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var userInfo oauthUserInfoResponse
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || userInfo.Subject == "" || userInfo.Error != "" {
		return nil, fmt.Errorf("userinfo endpoint status=%d error=%s", resp.StatusCode, userInfo.Error)
	}
	return &userInfo, nil
}

func (h *oauthHTTPHandler) redirectOAuthError(w stdhttp.ResponseWriter, r *stdhttp.Request, state oauthStatePayload, code string) {
	target := joinFrontendPath(state.FrontendOrigin, "/oauth/callback")
	fragment := url.Values{}
	fragment.Set("error", code)
	fragment.Set("next", state.Next)
	target += "#" + fragment.Encode()
	stdhttp.Redirect(w, r, target, stdhttp.StatusFound)
}

func writeJSON(w stdhttp.ResponseWriter, status int, payload map[string]any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func signOAuthState(payload oauthStatePayload, secret []byte) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	bodyPart := base64.RawURLEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(bodyPart))
	sigPart := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return bodyPart + "." + sigPart, nil
}

func verifyOAuthState(raw string, secret []byte, now time.Time) (oauthStatePayload, error) {
	var payload oauthStatePayload
	if raw == "" || len(secret) == 0 {
		return payload, errors.New("missing state")
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return payload, errors.New("invalid state format")
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(parts[0]))
	expected := mac.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(got, expected) {
		return payload, errors.New("invalid state signature")
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return payload, err
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return payload, err
	}
	if now.Unix() > payload.ExpiresAt {
		return payload, errors.New("state expired")
	}
	if !isSafeAbsoluteOrigin(payload.FrontendOrigin) || payload.RedirectURI == "" {
		return payload, errors.New("state origin invalid")
	}
	payload.Next = safeNextPath(payload.Next)
	return payload, nil
}

func requestExternalOrigin(r *stdhttp.Request) string {
	proto := firstHeaderValue(r, "X-Forwarded-Proto")
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := firstHeaderValue(r, "X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return strings.ToLower(strings.TrimSpace(proto)) + "://" + strings.TrimSpace(host)
}

func firstHeaderValue(r *stdhttp.Request, name string) string {
	value := strings.TrimSpace(r.Header.Get(name))
	if i := strings.Index(value, ","); i >= 0 {
		value = strings.TrimSpace(value[:i])
	}
	return value
}

func requestedFrontendOrigin(r *stdhttp.Request, fallback string) string {
	if origin := strings.TrimSpace(r.URL.Query().Get("frontend_origin")); origin != "" {
		return strings.TrimRight(origin, "/")
	}
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		return strings.TrimRight(origin, "/")
	}
	if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" {
		if parsed, err := url.Parse(referer); err == nil && parsed.Scheme != "" && parsed.Host != "" {
			return parsed.Scheme + "://" + parsed.Host
		}
	}
	return strings.TrimRight(fallback, "/")
}

func isAllowedFrontendOrigin(origin, requestOrigin string, allowed []string) bool {
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	requestOrigin = strings.TrimRight(strings.TrimSpace(requestOrigin), "/")
	if origin == "" || !isSafeAbsoluteOrigin(origin) {
		return false
	}
	if strings.EqualFold(origin, requestOrigin) || isLoopbackOrigin(origin) {
		return true
	}
	for _, item := range allowed {
		if strings.EqualFold(origin, strings.TrimRight(strings.TrimSpace(item), "/")) {
			return true
		}
	}
	return false
}

func isSafeAbsoluteOrigin(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return parsed.Path == "" && parsed.RawQuery == "" && parsed.Fragment == "" && parsed.User == nil
}

func isLoopbackOrigin(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return false
	}
	host := parsed.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func safeNextPath(raw string) string {
	if raw == "" {
		return "/admin-dashboard"
	}
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.Contains(raw, "\\") {
		return "/admin-dashboard"
	}
	return raw
}

func randomNonce() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}

func buildOAuthSuccessURL(state oauthStatePayload, token string, expireAt time.Time, admin *biz.AdminUser) string {
	target := joinFrontendPath(state.FrontendOrigin, "/oauth/callback")
	fragment := url.Values{}
	fragment.Set("access_token", token)
	fragment.Set("expires_at", fmt.Sprintf("%d", expireAt.Unix()))
	fragment.Set("token_type", "Bearer")
	fragment.Set("user_id", fmt.Sprintf("%d", admin.ID))
	fragment.Set("username", admin.Username)
	fragment.Set("next", state.Next)
	return target + "#" + fragment.Encode()
}

func joinFrontendPath(origin, path string) string {
	return strings.TrimRight(origin, "/") + "/" + strings.TrimLeft(path, "/")
}
