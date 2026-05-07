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
	stdhttp "net/http"
	"net/url"
	"strings"
	"time"

	"server/internal/biz"
	"server/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	httpx "github.com/go-kratos/kratos/v2/transport/http"
	"go.opentelemetry.io/otel/sdk/trace"
)

const (
	oauthCallbackPath = "/auth/oauth/callback"
	oauthStateCookie  = "oauth_state"
	oauthStateTTL     = 10 * time.Minute
	oauthScopeAdmin   = "admin"
	oauthScopeUser    = "user"
)

type oauthHandler struct {
	log         *log.Helper
	cfg         *conf.Data_Auth_OAuth
	secret      []byte
	adminAuthUC *biz.AdminAuthUsecase
	userAuthUC  *biz.AuthUsecase
	client      *stdhttp.Client
}

type oauthState struct {
	Nonce    string `json:"nonce"`
	Redirect string `json:"redirect"`
	Scope    string `json:"scope"`
	Exp      int64  `json:"exp"`
}

type oauthTokenResponse struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int64  `json:"expires_in"`
	RefreshToken     string `json:"refresh_token"`
	IDToken          string `json:"id_token"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type oauthUserInfo struct {
	Subject           string `json:"sub"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
}

func registerOAuthRoutes(srv *httpx.Server, logger log.Logger, tp *trace.TracerProvider, dataCfg *conf.Data, adminAuthUC *biz.AdminAuthUsecase, userAuthUC *biz.AuthUsecase) {
	h := newOAuthHandler(logger, dataCfg, adminAuthUC, userAuthUC)
	srv.Handle("/auth/oauth/config", newObservedHTTPHandler(logger, tp, "server.http.oauth_config", h.config))
	srv.Handle("/auth/oauth/start", newObservedHTTPHandler(logger, tp, "server.http.oauth_start", h.start))
	srv.Handle(oauthCallbackPath, newObservedHTTPHandler(logger, tp, "server.http.oauth_callback", h.callback))
}

func newOAuthHandler(logger log.Logger, dataCfg *conf.Data, adminAuthUC *biz.AdminAuthUsecase, userAuthUC *biz.AuthUsecase) *oauthHandler {
	var oauthCfg *conf.Data_Auth_OAuth
	var secret []byte
	if dataCfg != nil && dataCfg.Auth != nil {
		oauthCfg = dataCfg.Auth.Oauth
		secret = []byte(dataCfg.Auth.JwtSecret)
	}
	return &oauthHandler{
		log:         log.NewHelper(log.With(logger, "module", "server.oauth")),
		cfg:         oauthCfg,
		secret:      secret,
		adminAuthUC: adminAuthUC,
		userAuthUC:  userAuthUC,
		client:      &stdhttp.Client{Timeout: 10 * time.Second},
	}
}

func (h *oauthHandler) config(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request) {
	enabled := h.isEnabled()
	loginURL := ""
	providerName := "OAuth"
	if h.cfg != nil && strings.TrimSpace(h.cfg.ProviderName) != "" {
		providerName = strings.TrimSpace(h.cfg.ProviderName)
	}
	if enabled {
		loginURL = "/auth/oauth/start"
	}
	h.writeJSON(w, stdhttp.StatusOK, map[string]any{
		"enabled":       enabled,
		"provider_name": providerName,
		"login_url":     loginURL,
	})
}

func (h *oauthHandler) start(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if !h.isEnabled() {
		h.writeJSON(w, stdhttp.StatusNotFound, map[string]any{"error": "oauth_disabled"})
		return
	}

	nonce, err := randomURLToken(24)
	if err != nil {
		h.log.WithContext(ctx).Errorf("OAuth start nonce failed err=%v", err)
		h.writeJSON(w, stdhttp.StatusInternalServerError, map[string]any{"error": "oauth_start_failed"})
		return
	}
	scope := normalizeOAuthScope(r.URL.Query().Get("scope"))
	state, err := h.signState(oauthState{
		Nonce:    nonce,
		Redirect: safeRedirectPath(r.URL.Query().Get("redirect"), defaultRedirectForScope(scope)),
		Scope:    scope,
		Exp:      time.Now().Add(oauthStateTTL).Unix(),
	})
	if err != nil {
		h.log.WithContext(ctx).Errorf("OAuth start sign state failed err=%v", err)
		h.writeJSON(w, stdhttp.StatusInternalServerError, map[string]any{"error": "oauth_start_failed"})
		return
	}

	setOAuthStateCookie(w, r, nonce, int(oauthStateTTL.Seconds()))
	authURL, err := h.buildAuthURL(r, state)
	if err != nil {
		h.log.WithContext(ctx).Warnf("OAuth start build auth url failed err=%v", err)
		h.writeJSON(w, stdhttp.StatusInternalServerError, map[string]any{"error": "oauth_start_failed"})
		return
	}
	stdhttp.Redirect(w, r, authURL, stdhttp.StatusFound)
}

func (h *oauthHandler) callback(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if !h.isEnabled() {
		stdhttp.Redirect(w, r, "/admin-login?oauth_error=disabled", stdhttp.StatusFound)
		return
	}
	state, err := h.verifyState(r.URL.Query().Get("state"))
	if err != nil {
		h.log.WithContext(ctx).Warnf("OAuth callback invalid state err=%v", err)
		stdhttp.Redirect(w, r, "/admin-login?oauth_error=state", stdhttp.StatusFound)
		return
	}
	if cookie, err := r.Cookie(oauthStateCookie); err != nil || cookie.Value != state.Nonce {
		h.log.WithContext(ctx).Warn("OAuth callback state cookie mismatch")
		stdhttp.Redirect(w, r, oauthErrorRedirect(state.Scope, "state"), stdhttp.StatusFound)
		return
	}
	clearOAuthStateCookie(w, r)
	if providerErr := strings.TrimSpace(r.URL.Query().Get("error")); providerErr != "" {
		h.log.WithContext(ctx).Warnf("OAuth provider callback error scope=%s error=%s", state.Scope, providerErr)
		stdhttp.Redirect(w, r, oauthErrorRedirect(state.Scope, "provider"), stdhttp.StatusFound)
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		stdhttp.Redirect(w, r, oauthErrorRedirect(state.Scope, "missing_code"), stdhttp.StatusFound)
		return
	}

	tokenResp, err := h.exchangeCode(ctx, r, code)
	if err != nil {
		h.log.WithContext(ctx).Warnf("OAuth callback exchange code failed err=%v", err)
		stdhttp.Redirect(w, r, oauthErrorRedirect(state.Scope, "token"), stdhttp.StatusFound)
		return
	}
	userinfo, err := h.fetchUserInfo(ctx, tokenResp)
	if err != nil {
		h.log.WithContext(ctx).Warnf("OAuth callback fetch userinfo failed err=%v", err)
		stdhttp.Redirect(w, r, oauthErrorRedirect(state.Scope, "userinfo"), stdhttp.StatusFound)
		return
	}

	identity := biz.OAuthIdentity{
		Provider:          h.providerID(),
		Subject:           userinfo.Subject,
		Email:             userinfo.Email,
		Name:              userinfo.Name,
		PreferredUsername: userinfo.PreferredUsername,
	}

	accessToken, expiresAt, userID, username, loginScope, err := h.loginByScope(ctx, state.Scope, identity)
	if err != nil {
		h.log.WithContext(ctx).Warnf("OAuth callback login failed scope=%s provider=%s err=%v", state.Scope, h.providerID(), err)
		stdhttp.Redirect(w, r, oauthErrorRedirect(state.Scope, "login"), stdhttp.StatusFound)
		return
	}

	values := url.Values{}
	values.Set("access_token", accessToken)
	values.Set("expires_at", fmt.Sprintf("%d", expiresAt.Unix()))
	values.Set("token_type", "Bearer")
	values.Set("user_id", fmt.Sprintf("%d", userID))
	values.Set("username", username)
	values.Set("scope", loginScope)
	values.Set("redirect", state.Redirect)
	stdhttp.Redirect(w, r, "/oauth/callback#"+values.Encode(), stdhttp.StatusFound)
}

func (h *oauthHandler) isEnabled() bool {
	return h != nil && h.cfg != nil && len(h.secret) > 0 && h.cfg.Enabled &&
		strings.TrimSpace(h.cfg.ClientId) != "" &&
		strings.TrimSpace(h.cfg.AuthUrl) != "" &&
		strings.TrimSpace(h.cfg.TokenUrl) != "" &&
		strings.TrimSpace(h.cfg.UserInfoUrl) != ""
}

func (h *oauthHandler) loginByScope(ctx context.Context, scope string, identity biz.OAuthIdentity) (token string, expireAt time.Time, userID int, username string, loginScope string, err error) {
	scope = normalizeOAuthScope(scope)
	if scope == oauthScopeUser {
		if h.userAuthUC == nil {
			return "", time.Time{}, 0, "", scope, errors.New("user oauth login is unavailable")
		}
		token, expireAt, user, err := h.userAuthUC.LoginWithOAuth(ctx, identity)
		if err != nil {
			return "", time.Time{}, 0, "", scope, err
		}
		return token, expireAt, user.ID, user.Username, scope, nil
	}

	if h.adminAuthUC == nil {
		return "", time.Time{}, 0, "", oauthScopeAdmin, errors.New("admin oauth login is unavailable")
	}
	token, expireAt, admin, err := h.adminAuthUC.LoginWithOAuth(ctx, identity)
	if err != nil {
		return "", time.Time{}, 0, "", oauthScopeAdmin, err
	}
	return token, expireAt, admin.ID, admin.Username, oauthScopeAdmin, nil
}

func (h *oauthHandler) providerID() string {
	name := "oauth"
	if h.cfg != nil && strings.TrimSpace(h.cfg.ProviderName) != "" {
		name = strings.TrimSpace(h.cfg.ProviderName)
	}
	name = strings.ToLower(name)
	var b strings.Builder
	lastSep := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastSep = false
		case !lastSep:
			b.WriteByte('-')
			lastSep = true
		}
	}
	provider := strings.Trim(b.String(), "-")
	if provider == "" {
		return "oauth"
	}
	if len(provider) > 32 {
		return provider[:32]
	}
	return provider
}

func (h *oauthHandler) buildAuthURL(r *stdhttp.Request, state string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(h.cfg.AuthUrl))
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", strings.TrimSpace(h.cfg.ClientId))
	q.Set("redirect_uri", h.redirectURL(r))
	q.Set("scope", strings.Join(h.scopes(), " "))
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (h *oauthHandler) redirectURL(r *stdhttp.Request) string {
	if h.cfg != nil && strings.TrimSpace(h.cfg.RedirectUrl) != "" {
		return strings.TrimSpace(h.cfg.RedirectUrl)
	}
	scheme := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	return scheme + "://" + host + oauthCallbackPath
}

func (h *oauthHandler) scopes() []string {
	if h.cfg != nil && len(h.cfg.Scopes) > 0 {
		return h.cfg.Scopes
	}
	return []string{"openid", "profile", "email"}
}

func (h *oauthHandler) exchangeCode(ctx context.Context, r *stdhttp.Request, code string) (*oauthTokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", h.redirectURL(r))
	form.Set("client_id", strings.TrimSpace(h.cfg.ClientId))
	if secret := strings.TrimSpace(h.cfg.ClientSecret); secret != "" {
		form.Set("client_secret", secret)
	}

	req, err := stdhttp.NewRequestWithContext(ctx, stdhttp.MethodPost, strings.TrimSpace(h.cfg.TokenUrl), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var out oauthTokenResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || out.Error != "" {
		if out.Error != "" {
			return nil, fmt.Errorf("oauth token endpoint rejected request: %s", out.Error)
		}
		return nil, fmt.Errorf("oauth token endpoint status=%d", resp.StatusCode)
	}
	if out.AccessToken == "" {
		return nil, errors.New("oauth token response missing access_token")
	}
	return &out, nil
}

func (h *oauthHandler) fetchUserInfo(ctx context.Context, tokenResp *oauthTokenResponse) (*oauthUserInfo, error) {
	if tokenResp == nil {
		return nil, errors.New("missing token response")
	}
	req, err := stdhttp.NewRequestWithContext(ctx, stdhttp.MethodGet, strings.TrimSpace(h.cfg.UserInfoUrl), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("oauth userinfo endpoint status=%d", resp.StatusCode)
	}
	var out oauthUserInfo
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.Subject) == "" {
		return nil, errors.New("oauth userinfo missing subject")
	}
	return &out, nil
}

func (h *oauthHandler) signState(state oauthState) (string, error) {
	payload, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	p := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, h.secret)
	_, _ = mac.Write([]byte(p))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return p + "." + sig, nil
}

func (h *oauthHandler) verifyState(raw string) (oauthState, error) {
	var state oauthState
	parts := strings.Split(raw, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return state, errors.New("invalid state format")
	}
	mac := hmac.New(sha256.New, h.secret)
	_, _ = mac.Write([]byte(parts[0]))
	expected := mac.Sum(nil)
	actual, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return state, err
	}
	if !hmac.Equal(expected, actual) {
		return state, errors.New("state signature mismatch")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(payload, &state); err != nil {
		return state, err
	}
	if state.Nonce == "" || time.Now().Unix() > state.Exp {
		return state, errors.New("state expired")
	}
	state.Scope = normalizeOAuthScope(state.Scope)
	state.Redirect = safeRedirectPath(state.Redirect, defaultRedirectForScope(state.Scope))
	return state, nil
}

func (h *oauthHandler) writeJSON(w stdhttp.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func randomURLToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func safeRedirectPath(raw string, fallback string) string {
	if raw == "" {
		return fallback
	}
	u, err := url.Parse(raw)
	if err != nil || u.IsAbs() || strings.HasPrefix(raw, "//") || !strings.HasPrefix(raw, "/") {
		return fallback
	}
	return raw
}

func normalizeOAuthScope(raw string) string {
	if strings.EqualFold(strings.TrimSpace(raw), oauthScopeUser) {
		return oauthScopeUser
	}
	return oauthScopeAdmin
}

func defaultRedirectForScope(scope string) string {
	return "/admin-dashboard"
}

func oauthErrorRedirect(scope string, code string) string {
	return "/admin-login?oauth_error=" + url.QueryEscape(code)
}

func setOAuthStateCookie(w stdhttp.ResponseWriter, r *stdhttp.Request, value string, maxAge int) {
	stdhttp.SetCookie(w, &stdhttp.Cookie{
		Name:     oauthStateCookie,
		Value:    value,
		Path:     oauthCallbackPath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   requestIsHTTPS(r),
		SameSite: stdhttp.SameSiteLaxMode,
	})
}

func clearOAuthStateCookie(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	setOAuthStateCookie(w, r, "", -1)
}

func requestIsHTTPS(r *stdhttp.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
