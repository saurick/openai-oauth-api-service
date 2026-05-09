// server/internal/biz/admin_auth.go
package biz

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
)

type AdminAuthRepo interface {
	GetAdminByUsername(ctx context.Context, username string) (*AdminUser, error)
	GetAdminByOAuthIdentity(ctx context.Context, provider, subject string) (*AdminUser, error)
	BindAdminOAuthIdentity(ctx context.Context, id int, identity OAuthIdentity) error
	UpdateAdminLastLogin(ctx context.Context, id int, t time.Time) error
}

type AdminUser struct {
	ID            int
	Username      string
	PasswordHash  string
	OAuthProvider *string
	OAuthSubject  *string
	OAuthEmail    *string
	OAuthName     *string
	Disabled      bool
}

type AdminAuthUsecase struct {
	log    *log.Helper
	logger log.Logger
	tp     *tracesdk.TracerProvider
	tracer trace.Tracer
	repo   AdminAuthRepo
	genTok AdminTokenGenerator
}

func NewAdminAuthUsecase(repo AdminAuthRepo, genTok AdminTokenGenerator, logger log.Logger, tp *tracesdk.TracerProvider) *AdminAuthUsecase {
	helper := log.NewHelper(log.With(logger, "module", "biz.admin_auth"))

	var tr trace.Tracer
	if tp != nil {
		tr = tp.Tracer("biz.admin_auth")
	} else {
		tr = otel.Tracer("biz.admin_auth")
	}

	return &AdminAuthUsecase{
		repo:   repo,
		genTok: genTok,
		log:    helper,
		logger: logger,
		tp:     tp,
		tracer: tr,
	}
}

func (uc *AdminAuthUsecase) Tracer(opts ...trace.TracerOption) trace.Tracer {
	if uc.tracer != nil {
		return uc.tracer
	}
	return otel.Tracer("biz.admin_auth", opts...)
}

func (uc *AdminAuthUsecase) Login(ctx context.Context, username, password string) (token string, expireAt time.Time, u *AdminUser, err error) {
	ctx, span := uc.Tracer().Start(ctx, "admin_auth.login",
		trace.WithAttributes(
			attribute.String("admin_auth.username", username),
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

	admin, e := uc.repo.GetAdminByUsername(ctx, username)
	if e != nil || admin == nil {
		err = ErrUserNotFound
		span.RecordError(e)
		span.SetStatus(codes.Error, err.Error())
		l.Infof("Login admin not found username=%s err=%v", username, e)
		return "", time.Time{}, nil, err
	}

	span.SetAttributes(attribute.Int("admin_auth.admin_id", admin.ID))

	if admin.Disabled {
		err = ErrUserDisabled
		span.SetStatus(codes.Error, err.Error())
		l.Infof("Login admin disabled admin_id=%d username=%s", admin.ID, username)
		return "", time.Time{}, nil, err
	}

	if bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)) != nil {
		err = ErrInvalidPassword
		span.SetStatus(codes.Error, err.Error())
		l.Infof("Login admin invalid password admin_id=%d username=%s", admin.ID, username)
		return "", time.Time{}, nil, err
	}

	token, expireAt, e = uc.genTok(admin.ID, admin.Username, int8(RoleAdmin))
	if e != nil {
		err = e
		span.RecordError(err)
		span.SetStatus(codes.Error, "generate token failed")
		l.Errorf("Login admin generate token failed admin_id=%d username=%s err=%v", admin.ID, admin.Username, err)
		return "", time.Time{}, nil, err
	}

	span.SetAttributes(attribute.Int64("admin_auth.token_expires_at", expireAt.Unix()))

	if e := uc.repo.UpdateAdminLastLogin(ctx, admin.ID, time.Now()); e != nil {
		span.RecordError(e)
		l.Warnf("Login admin update last_login_at failed admin_id=%d err=%v", admin.ID, e)
	}

	span.SetStatus(codes.Ok, "OK")
	l.Infof("Login admin success admin_id=%d username=%s", admin.ID, admin.Username)

	return token, expireAt, admin, nil
}

func (uc *AdminAuthUsecase) LoginWithOAuth(ctx context.Context, identity OAuthIdentity) (token string, expireAt time.Time, u *AdminUser, err error) {
	ctx, span := uc.Tracer().Start(ctx, "admin_auth.login_with_oauth",
		trace.WithAttributes(
			attribute.String("admin_auth.oauth_provider", identity.Provider),
			attribute.String("admin_auth.oauth_email", identity.Email),
		),
	)
	defer span.End()

	l := uc.log.WithContext(ctx)
	if identity.Provider == "" || identity.Subject == "" {
		err = errors.New("missing oauth provider or subject")
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid argument")
		l.Warn("LoginWithOAuth invalid oauth identity")
		return "", time.Time{}, nil, err
	}

	admin, e := uc.repo.GetAdminByOAuthIdentity(ctx, identity.Provider, identity.Subject)
	if e != nil || admin == nil {
		if identity.Email == "" {
			err = ErrUserNotFound
			span.RecordError(e)
			span.SetStatus(codes.Error, err.Error())
			l.Infof("LoginWithOAuth admin not found provider=%s", identity.Provider)
			return "", time.Time{}, nil, err
		}

		admin, e = uc.repo.GetAdminByUsername(ctx, identity.Email)
		if e != nil || admin == nil {
			err = ErrUserNotFound
			span.RecordError(e)
			span.SetStatus(codes.Error, err.Error())
			l.Infof("LoginWithOAuth admin not found by email provider=%s email=%s", identity.Provider, identity.Email)
			return "", time.Time{}, nil, err
		}

		if bindErr := uc.repo.BindAdminOAuthIdentity(ctx, admin.ID, identity); bindErr != nil {
			err = bindErr
			span.RecordError(err)
			span.SetStatus(codes.Error, "bind oauth identity failed")
			l.Errorf("LoginWithOAuth bind identity failed admin_id=%d provider=%s err=%v", admin.ID, identity.Provider, err)
			return "", time.Time{}, nil, err
		}
	}

	span.SetAttributes(attribute.Int("admin_auth.admin_id", admin.ID))
	if admin.Disabled {
		err = ErrUserDisabled
		span.SetStatus(codes.Error, err.Error())
		l.Infof("LoginWithOAuth admin disabled admin_id=%d username=%s", admin.ID, admin.Username)
		return "", time.Time{}, nil, err
	}

	token, expireAt, e = uc.genTok(admin.ID, admin.Username, int8(RoleAdmin))
	if e != nil {
		err = e
		span.RecordError(err)
		span.SetStatus(codes.Error, "generate token failed")
		l.Errorf("LoginWithOAuth generate token failed admin_id=%d username=%s err=%v", admin.ID, admin.Username, err)
		return "", time.Time{}, nil, err
	}

	span.SetAttributes(attribute.Int64("admin_auth.token_expires_at", expireAt.Unix()))
	if e := uc.repo.UpdateAdminLastLogin(ctx, admin.ID, time.Now()); e != nil {
		span.RecordError(e)
		l.Warnf("LoginWithOAuth update last_login_at failed admin_id=%d err=%v", admin.ID, e)
	}

	span.SetStatus(codes.Ok, "OK")
	l.Infof("LoginWithOAuth success admin_id=%d username=%s provider=%s", admin.ID, admin.Username, identity.Provider)
	return token, expireAt, admin, nil
}
