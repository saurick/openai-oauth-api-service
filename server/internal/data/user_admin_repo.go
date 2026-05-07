// server/internal/data/user_admin_repo.go
package data

import (
	"context"
	"strings"

	"server/internal/biz"
	"server/internal/data/model/ent"
	entadminuser "server/internal/data/model/ent/adminuser"
	"server/internal/data/model/ent/user"

	"github.com/go-kratos/kratos/v2/log"
)

type userAdminRepo struct {
	log  *log.Helper
	data *Data
}

func NewUserAdminRepo(d *Data, logger log.Logger) *userAdminRepo {
	return &userAdminRepo{
		log:  log.NewHelper(log.With(logger, "module", "data.useradmin_repo")),
		data: d,
	}
}

var _ biz.UserAdminRepo = (*userAdminRepo)(nil)

func (r *userAdminRepo) CreateUser(ctx context.Context, username, passwordHash string) (*biz.User, error) {
	l := r.log.WithContext(ctx)
	username = strings.TrimSpace(username)
	if username == "" || passwordHash == "" {
		return nil, biz.ErrBadParam
	}
	l.Infof("CreateUser start username=%s", username)
	if exists, err := r.data.postgres.AdminUser.Query().
		Where(entadminuser.Username(username)).
		Exist(ctx); err != nil {
		return nil, err
	} else if exists {
		return nil, biz.ErrUserExists
	}

	u, err := r.data.postgres.User.Create().
		SetUsername(username).
		SetPasswordHash(passwordHash).
		Save(ctx)
	if err != nil {
		if isDuplicateUsernameConstraint(err) {
			return nil, biz.ErrUserExists
		}
		return nil, err
	}
	return &biz.User{
		ID:           u.ID,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		Disabled:     u.Disabled,
		Role:         int8(biz.RoleUser),
		LastLoginAt:  u.LastLoginAt,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}, nil
}

func (r *userAdminRepo) ListUsers(ctx context.Context, limit, offset int, usernameLike string) ([]*biz.User, int, error) {
	l := r.log.WithContext(ctx)

	if limit <= 0 {
		limit = 30
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	usernameLike = strings.TrimSpace(usernameLike)

	l.Infof("ListUsers start limit=%d offset=%d username_like=%q", limit, offset, usernameLike)

	q := r.data.postgres.User.Query()
	if usernameLike != "" {
		q = q.Where(user.UsernameContains(usernameLike))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		l.Errorf("ListUsers count failed err=%v", err)
		return nil, 0, err
	}

	rows, err := q.
		Order(ent.Desc(user.FieldID)).
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		l.Errorf("ListUsers query failed err=%v", err)
		return nil, 0, err
	}

	out := make([]*biz.User, 0, len(rows))
	for _, u := range rows {
		out = append(out, &biz.User{
			ID:          u.ID,
			Username:    u.Username,
			Disabled:    u.Disabled,
			Role:        int8(biz.RoleUser),
			LastLoginAt: u.LastLoginAt,
			CreatedAt:   u.CreatedAt,
			UpdatedAt:   u.UpdatedAt,
		})
	}

	l.Infof("ListUsers success count=%d total=%d", len(out), total)
	return out, total, nil
}

func (r *userAdminRepo) SetUserDisabled(ctx context.Context, userID int, disabled bool) error {
	l := r.log.WithContext(ctx)
	l.Infof("SetUserDisabled start user_id=%d disabled=%v", userID, disabled)

	if userID <= 0 {
		l.Warnf("SetUserDisabled bad user_id=%d", userID)
		return biz.ErrBadParam
	}

	_, err := r.data.postgres.User.UpdateOneID(userID).SetDisabled(disabled).Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			l.Warnf("SetUserDisabled not found user_id=%d", userID)
			return biz.ErrUserNotFound
		}
		l.Errorf("SetUserDisabled failed user_id=%d err=%v", userID, err)
		return err
	}

	l.Infof("SetUserDisabled success user_id=%d disabled=%v", userID, disabled)
	return nil
}

func (r *userAdminRepo) ResetUserPassword(ctx context.Context, userID int, passwordHash string) error {
	if userID <= 0 || passwordHash == "" {
		return biz.ErrBadParam
	}
	_, err := r.data.postgres.User.UpdateOneID(userID).
		SetPasswordHash(passwordHash).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return biz.ErrUserNotFound
		}
		return err
	}
	return nil
}
