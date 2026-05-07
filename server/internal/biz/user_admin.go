// server/internal/biz/user_admin.go
package biz

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
)

type UserAdminRepo interface {
	CreateUser(ctx context.Context, username, passwordHash string) (*User, error)
	ListUsers(ctx context.Context, limit, offset int, usernameLike string) (list []*User, total int, err error)
	SetUserDisabled(ctx context.Context, userID int, disabled bool) error
	ResetUserPassword(ctx context.Context, userID int, passwordHash string) error
}

type UserAdminUsecase struct {
	repo   UserAdminRepo
	log    *log.Helper
	tracer trace.Tracer
}

func NewUserAdminUsecase(repo UserAdminRepo, logger log.Logger, tp *tracesdk.TracerProvider) *UserAdminUsecase {
	helper := log.NewHelper(log.With(logger, "module", "biz.useradmin"))

	var tr trace.Tracer
	if tp != nil {
		tr = tp.Tracer("biz.useradmin")
	} else {
		tr = otel.Tracer("biz.useradmin")
	}

	return &UserAdminUsecase{
		repo:   repo,
		log:    helper,
		tracer: tr,
	}
}

func (uc *UserAdminUsecase) Tracer() trace.Tracer {
	if uc.tracer != nil {
		return uc.tracer
	}
	return otel.Tracer("biz.useradmin")
}

func (uc *UserAdminUsecase) requireAdmin(ctx context.Context) (*AuthClaims, error) {
	c, ok := GetClaimsFromContext(ctx)
	if !ok || c == nil {
		return nil, ErrForbidden
	}
	if c.Role != RoleAdmin {
		return nil, ErrForbidden
	}
	return c, nil
}

func (uc *UserAdminUsecase) List(
	ctx context.Context,
	limit, offset int,
	searchUsername string,
) (list []*User, total int, err error) {
	ctx, span := uc.Tracer().Start(ctx, "useradmin.list",
		trace.WithAttributes(
			attribute.Int("useradmin.limit", limit),
			attribute.Int("useradmin.offset", offset),
			attribute.String("useradmin.search_username", strings.TrimSpace(searchUsername)),
		),
	)
	defer span.End()

	l := uc.log.WithContext(ctx)

	admin, e := uc.requireAdmin(ctx)
	if e != nil {
		err = ErrForbidden
		span.SetStatus(codes.Error, err.Error())
		l.Warn("List forbidden")
		return nil, 0, err
	}
	span.SetAttributes(attribute.Int("auth.admin_uid", admin.UserID))

	if limit <= 0 {
		limit = 30
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	searchUsername = strings.TrimSpace(searchUsername)
	l.Infof("List start limit=%d offset=%d search=%q", limit, offset, searchUsername)

	list, total, err = uc.repo.ListUsers(ctx, limit, offset, searchUsername)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "repo.ListUsers failed")
		l.Errorf("List repo.ListUsers failed err=%v", err)
		return nil, 0, err
	}

	span.SetAttributes(
		attribute.Int("useradmin.count", len(list)),
		attribute.Int("useradmin.total", total),
	)
	span.SetStatus(codes.Ok, "OK")
	l.Infof("List success count=%d total=%d", len(list), total)

	return list, total, nil
}

func (uc *UserAdminUsecase) Create(ctx context.Context, username, password string) (*User, error) {
	ctx, span := uc.Tracer().Start(ctx, "useradmin.create",
		trace.WithAttributes(attribute.String("user.username", strings.TrimSpace(username))),
	)
	defer span.End()

	l := uc.log.WithContext(ctx)
	admin, e := uc.requireAdmin(ctx)
	if e != nil {
		span.SetStatus(codes.Error, ErrForbidden.Error())
		l.Warn("Create forbidden")
		return nil, ErrForbidden
	}
	span.SetAttributes(attribute.Int("auth.admin_uid", admin.UserID))

	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		span.SetStatus(codes.Error, ErrBadParam.Error())
		return nil, ErrBadParam
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "hash password failed")
		return nil, err
	}
	user, err := uc.repo.CreateUser(ctx, username, string(hash))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "repo.CreateUser failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "OK")
	return user, nil
}

func (uc *UserAdminUsecase) ResetPassword(ctx context.Context, userID int, password string) error {
	ctx, span := uc.Tracer().Start(ctx, "useradmin.reset_password",
		trace.WithAttributes(attribute.Int("user.id", userID)),
	)
	defer span.End()

	l := uc.log.WithContext(ctx)
	admin, e := uc.requireAdmin(ctx)
	if e != nil {
		span.SetStatus(codes.Error, ErrForbidden.Error())
		l.Warn("ResetPassword forbidden")
		return ErrForbidden
	}
	span.SetAttributes(attribute.Int("auth.admin_uid", admin.UserID))

	if userID <= 0 || password == "" {
		span.SetStatus(codes.Error, ErrBadParam.Error())
		return ErrBadParam
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "hash password failed")
		return err
	}
	if err := uc.repo.ResetUserPassword(ctx, userID, string(hash)); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "repo.ResetUserPassword failed")
		return err
	}
	span.SetStatus(codes.Ok, "OK")
	return nil
}

func (uc *UserAdminUsecase) SetDisabled(ctx context.Context, userID int, disabled bool) error {
	ctx, span := uc.Tracer().Start(ctx, "useradmin.set_disabled",
		trace.WithAttributes(
			attribute.Int("user.id", userID),
			attribute.Bool("user.disabled", disabled),
		),
	)
	defer span.End()

	l := uc.log.WithContext(ctx)

	admin, e := uc.requireAdmin(ctx)
	if e != nil {
		span.SetStatus(codes.Error, ErrForbidden.Error())
		l.Warn("SetDisabled forbidden")
		return ErrForbidden
	}
	span.SetAttributes(attribute.Int("auth.admin_uid", admin.UserID))

	if userID <= 0 {
		span.SetStatus(codes.Error, ErrBadParam.Error())
		l.Warn("SetDisabled bad userID")
		return ErrBadParam
	}

	l.Infof("SetDisabled start user_id=%d disabled=%v", userID, disabled)
	if err := uc.repo.SetUserDisabled(ctx, userID, disabled); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "repo.SetUserDisabled failed")
		l.Errorf("SetDisabled repo.SetUserDisabled failed user_id=%d err=%v", userID, err)
		return err
	}

	span.SetStatus(codes.Ok, "OK")
	l.Infof("SetDisabled success user_id=%d disabled=%v", userID, disabled)
	return nil
}
