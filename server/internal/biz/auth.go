// server/internal/biz/auth.go
package biz

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists      = errors.New("user already exists")
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidPassword = errors.New("invalid password")
	ErrUserDisabled    = errors.New("user disabled")
)

type AuthRepo interface {
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByID(ctx context.Context, id int) (*User, error)
	GetUserByOAuthIdentity(ctx context.Context, provider, subject string) (*User, error)
	CreateUser(ctx context.Context, u *User) (*User, error)
	BindUserOAuthIdentity(ctx context.Context, id int, identity OAuthIdentity) error
	UpdateUserLastLogin(ctx context.Context, id int, t time.Time) error
}

type User struct {
	ID            int
	Username      string
	PasswordHash  string
	Disabled      bool
	OAuthProvider *string
	OAuthSubject  *string
	OAuthEmail    *string
	OAuthName     *string
	// Role 只用于登录态返回与 token 生成；当前项目默认不再把业务角色字段持久化到 users 表。
	Role        int8
	LastLoginAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type OAuthIdentity struct {
	Provider          string
	Subject           string
	Email             string
	Name              string
	PreferredUsername string
}

type TokenGenerator func(userID int, username string, role int8) (token string, expireAt time.Time, err error)
type AdminTokenGenerator func(userID int, username string, role int8) (token string, expireAt time.Time, err error)

type AuthUsecase struct {
	// 日志
	log    *log.Helper
	logger log.Logger

	// tracing
	tp     *tracesdk.TracerProvider
	tracer trace.Tracer

	// repo & token
	repo   AuthRepo
	genTok TokenGenerator
}

func normalizeOAuthIdentity(in OAuthIdentity) OAuthIdentity {
	in.Provider = strings.TrimSpace(in.Provider)
	in.Subject = strings.TrimSpace(in.Subject)
	in.Email = strings.TrimSpace(in.Email)
	in.Name = strings.TrimSpace(in.Name)
	in.PreferredUsername = strings.TrimSpace(in.PreferredUsername)
	return in
}

func oauthSubjectHash(provider, subject string) string {
	sum := sha256.Sum256([]byte(provider + ":" + subject))
	return hex.EncodeToString(sum[:])[:10]
}

func sanitizeUsernameSeed(seed string) string {
	seed = strings.ToLower(strings.TrimSpace(seed))
	if at := strings.Index(seed, "@"); at > 0 {
		seed = seed[:at]
	}

	var b strings.Builder
	lastSep := false
	for _, r := range seed {
		valid := unicode.IsLetter(r) || unicode.IsDigit(r)
		sep := r == '-' || r == '_' || r == '.' || unicode.IsSpace(r)
		switch {
		case valid:
			b.WriteRune(r)
			lastSep = false
		case sep && !lastSep:
			b.WriteByte('-')
			lastSep = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func truncateString(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func oauthUsernameCandidate(identity OAuthIdentity, attempt int) string {
	seed := identity.PreferredUsername
	if seed == "" {
		seed = identity.Email
	}
	if seed == "" {
		seed = identity.Name
	}
	base := sanitizeUsernameSeed(seed)
	if base == "" {
		base = "oauth"
	}
	hash := oauthSubjectHash(identity.Provider, identity.Subject)
	if attempt <= 0 {
		return fmt.Sprintf("%s-%s", truncateString(base, 21), hash)
	}
	return fmt.Sprintf("%s-%02d-%s", truncateString(base, 18), attempt, hash)
}

func NewAuthUsecase(repo AuthRepo, genTok TokenGenerator, logger log.Logger, tp *tracesdk.TracerProvider) *AuthUsecase {
	helper := log.NewHelper(log.With(logger, "module", "biz.auth"))

	// tracer 优先用注入的 tp；tp 为空就 fallback 全局 provider
	var tr trace.Tracer
	if tp != nil {
		tr = tp.Tracer("biz.auth")
	} else {
		tr = otel.Tracer("biz.auth")
	}

	return &AuthUsecase{
		repo:   repo,
		genTok: genTok,
		log:    helper,
		logger: logger,
		tp:     tp,
		tracer: tr,
	}
}

func (uc *AuthUsecase) Tracer(opts ...trace.TracerOption) trace.Tracer {
	if uc.tracer != nil {
		// opts 不会影响已创建 tracer（一般也没必要）
		return uc.tracer
	}
	return otel.Tracer("biz.auth", opts...)
}

// ======================
// 注册
// ======================

func (uc *AuthUsecase) Register(ctx context.Context, username, password string) (token string, expireAt time.Time, u *User, err error) {
	ctx, span := uc.Tracer().Start(ctx, "auth.register",
		trace.WithAttributes(
			attribute.String("auth.username", username),
		),
	)
	defer span.End()

	l := uc.log.WithContext(ctx)

	if username == "" || password == "" {
		err = errors.New("missing username or password")
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid argument")
		l.Warnf("Register invalid args username=%q", username)
		return "", time.Time{}, nil, err
	}

	l.Infof("Register start username=%s", username)

	// 2) 用户名是否已存在
	exist, e := uc.repo.GetUserByUsername(ctx, username)
	if e == nil && exist != nil {
		err = ErrUserExists
		span.SetStatus(codes.Error, err.Error())
		l.Infof("Register user already exists username=%s", username)
		return "", time.Time{}, nil, err
	}
	// 如果 repo 返回 error（比如 not found / db error），这里不强判，交给后续 CreateUser 去兜底（唯一索引）

	// 3) 哈希密码（不打印 password）
	hash, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if e != nil {
		err = e
		span.RecordError(err)
		span.SetStatus(codes.Error, "hash password failed")
		l.Errorf("Register hash password failed username=%s err=%v", username, err)
		return "", time.Time{}, nil, err
	}

	newUser := &User{
		Username:     username,
		PasswordHash: string(hash),
	}

	// 4) 创建用户
	created, e := uc.repo.CreateUser(ctx, newUser)
	if e != nil {
		err = e
		span.RecordError(err)
		span.SetStatus(codes.Error, "create user failed")
		l.Errorf("Register create user failed username=%s err=%v", username, err)
		return "", time.Time{}, nil, err
	}

	span.SetAttributes(attribute.Int("auth.user_id", created.ID))

	// 5) 创建 token
	// 注册出来的用户默认 Role=0（普通用户）
	created.Role = 0
	token, expireAt, e = uc.genTok(created.ID, created.Username, created.Role)
	if e != nil {
		err = e
		span.RecordError(err)
		span.SetStatus(codes.Error, "generate token failed")
		l.Errorf("Register generate token failed user_id=%d username=%s err=%v", created.ID, created.Username, err)
		return "", time.Time{}, nil, err
	}

	span.SetAttributes(attribute.Int64("auth.token_expires_at", expireAt.Unix()))

	// 6) 更新 last_login_at（失败不影响主流程）
	if e := uc.repo.UpdateUserLastLogin(ctx, created.ID, time.Now()); e != nil {
		span.RecordError(e)
		l.Warnf("Register update last_login_at failed user_id=%d err=%v", created.ID, e)
	}

	span.SetStatus(codes.Ok, "OK")
	l.Infof("Register success user_id=%d username=%s", created.ID, created.Username)

	return token, expireAt, created, nil
}

// ======================
// 登录
// ======================

func (uc *AuthUsecase) Login(ctx context.Context, username, password string) (token string, expireAt time.Time, u *User, err error) {
	ctx, span := uc.Tracer().Start(ctx, "auth.login",
		trace.WithAttributes(
			attribute.String("auth.username", username),
		),
	)
	defer span.End()

	l := uc.log.WithContext(ctx)

	if username == "" || password == "" {
		err = errors.New("missing username or password")
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid argument")
		l.Warnf("Login invalid args username=%q", username)
		return "", time.Time{}, nil, err
	}

	l.Infof("Login start username=%s", username)

	usr, e := uc.repo.GetUserByUsername(ctx, username)
	if e != nil || usr == nil {
		err = ErrUserNotFound
		span.RecordError(e)
		span.SetStatus(codes.Error, err.Error())
		l.Infof("Login user not found username=%s err=%v", username, e)
		return "", time.Time{}, nil, err
	}

	span.SetAttributes(attribute.Int("auth.user_id", usr.ID))

	if usr.Disabled {
		err = ErrUserDisabled
		span.SetStatus(codes.Error, err.Error())
		l.Infof("Login user disabled user_id=%d username=%s", usr.ID, username)
		return "", time.Time{}, nil, err
	}

	// 不要记录 password
	if bcrypt.CompareHashAndPassword([]byte(usr.PasswordHash), []byte(password)) != nil {
		err = ErrInvalidPassword
		span.SetStatus(codes.Error, err.Error())
		l.Infof("Login invalid password user_id=%d username=%s", usr.ID, username)
		return "", time.Time{}, nil, err
	}

	uc.log.WithContext(ctx).Infof("Login user=%s id=%d role=%d", usr.Username, usr.ID, usr.Role)

	token, expireAt, e = uc.genTok(usr.ID, usr.Username, usr.Role)
	if e != nil {
		err = e
		span.RecordError(err)
		span.SetStatus(codes.Error, "generate token failed")
		l.Errorf("Login generate token failed user_id=%d username=%s err=%v", usr.ID, usr.Username, err)
		return "", time.Time{}, nil, err
	}

	span.SetAttributes(attribute.Int64("auth.token_expires_at", expireAt.Unix()))

	if e := uc.repo.UpdateUserLastLogin(ctx, usr.ID, time.Now()); e != nil {
		span.RecordError(e)
		l.Warnf("Login update last_login_at failed user_id=%d err=%v", usr.ID, e)
	}

	span.SetStatus(codes.Ok, "OK")
	l.Infof("Login success user_id=%d username=%s", usr.ID, usr.Username)

	return token, expireAt, usr, nil
}

func (uc *AuthUsecase) LoginWithOAuth(ctx context.Context, identity OAuthIdentity) (token string, expireAt time.Time, u *User, err error) {
	identity = normalizeOAuthIdentity(identity)
	ctx, span := uc.Tracer().Start(ctx, "auth.oauth_login",
		trace.WithAttributes(
			attribute.String("auth.oauth_provider", identity.Provider),
		),
	)
	defer span.End()

	l := uc.log.WithContext(ctx)
	if identity.Provider == "" || identity.Subject == "" {
		err = errors.New("missing oauth provider or subject")
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid argument")
		return "", time.Time{}, nil, err
	}

	usr, e := uc.repo.GetUserByOAuthIdentity(ctx, identity.Provider, identity.Subject)
	if e != nil || usr == nil {
		usr, e = uc.findBindOrCreateUserByOAuthIdentity(ctx, identity)
		if e != nil {
			err = e
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			l.Warnf("OAuth user login failed provider=%s err=%v", identity.Provider, err)
			return "", time.Time{}, nil, err
		}
	}

	span.SetAttributes(attribute.Int("auth.user_id", usr.ID))
	if usr.Disabled {
		err = ErrUserDisabled
		span.SetStatus(codes.Error, err.Error())
		l.Infof("OAuth user disabled user_id=%d username=%s provider=%s", usr.ID, usr.Username, identity.Provider)
		return "", time.Time{}, nil, err
	}

	usr.Role = int8(RoleUser)
	token, expireAt, e = uc.genTok(usr.ID, usr.Username, usr.Role)
	if e != nil {
		err = e
		span.RecordError(err)
		span.SetStatus(codes.Error, "generate token failed")
		l.Errorf("OAuth user generate token failed user_id=%d username=%s err=%v", usr.ID, usr.Username, err)
		return "", time.Time{}, nil, err
	}

	span.SetAttributes(attribute.Int64("auth.token_expires_at", expireAt.Unix()))
	if e := uc.repo.UpdateUserLastLogin(ctx, usr.ID, time.Now()); e != nil {
		span.RecordError(e)
		l.Warnf("OAuth user update last_login_at failed user_id=%d err=%v", usr.ID, e)
	}

	span.SetStatus(codes.Ok, "OK")
	l.Infof("OAuth user login success user_id=%d username=%s provider=%s", usr.ID, usr.Username, identity.Provider)
	return token, expireAt, usr, nil
}

func (uc *AuthUsecase) findBindOrCreateUserByOAuthIdentity(ctx context.Context, identity OAuthIdentity) (*User, error) {
	candidates := []string{identity.Email, identity.PreferredUsername}
	seen := map[string]struct{}{}
	for _, username := range candidates {
		username = strings.TrimSpace(username)
		if username == "" {
			continue
		}
		if _, ok := seen[username]; ok {
			continue
		}
		seen[username] = struct{}{}

		usr, err := uc.repo.GetUserByUsername(ctx, username)
		if err != nil || usr == nil {
			continue
		}
		if err := uc.repo.BindUserOAuthIdentity(ctx, usr.ID, identity); err != nil {
			return nil, err
		}
		usr.OAuthProvider = optionalStringPtr(identity.Provider)
		usr.OAuthSubject = optionalStringPtr(identity.Subject)
		usr.OAuthEmail = optionalStringPtr(identity.Email)
		usr.OAuthName = optionalStringPtr(identity.Name)
		return usr, nil
	}

	passwordHash, err := generateUnusableOAuthPasswordHash()
	if err != nil {
		return nil, err
	}
	for attempt := 0; attempt < 10; attempt++ {
		created, err := uc.repo.CreateUser(ctx, &User{
			Username:      oauthUsernameCandidate(identity, attempt),
			PasswordHash:  passwordHash,
			OAuthProvider: optionalStringPtr(identity.Provider),
			OAuthSubject:  optionalStringPtr(identity.Subject),
			OAuthEmail:    optionalStringPtr(identity.Email),
			OAuthName:     optionalStringPtr(identity.Name),
		})
		if err == nil {
			created.Role = int8(RoleUser)
			return created, nil
		}
		if !errors.Is(err, ErrUserExists) {
			return nil, err
		}
	}
	return nil, ErrUserExists
}

func generateUnusableOAuthPasswordHash() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	raw := "oauth:" + base64.RawURLEncoding.EncodeToString(b)
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func optionalStringPtr(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

// GetCurrentUser 获取当前登录用户信息（用于 auth.me 等接口）
func (uc *AuthUsecase) GetCurrentUser(ctx context.Context, userID int) (*User, error) {
	ctx, span := uc.Tracer().Start(ctx, "auth.get_current_user",
		trace.WithAttributes(
			attribute.Int("auth.user_id", userID),
		),
	)
	defer span.End()

	l := uc.log.WithContext(ctx)
	if userID <= 0 {
		err := errors.New("user_id is required")
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid argument")
		l.Warnf("GetCurrentUser invalid user_id=%d", userID)
		return nil, err
	}

	u, err := uc.repo.GetUserByID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get user failed")
		l.Errorf("GetCurrentUser failed user_id=%d err=%v", userID, err)
		return nil, err
	}

	span.SetStatus(codes.Ok, "OK")
	return u, nil
}
