// server/internal/data/auth_repo.go
package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"server/internal/biz"
	entmodel "server/internal/data/model/ent"
	entadminuser "server/internal/data/model/ent/adminuser"
	entuser "server/internal/data/model/ent/user"

	"github.com/go-kratos/kratos/v2/log"
)

type authRepo struct {
	data *Data
	log  *log.Helper
}

func NewAuthRepo(data *Data, logger log.Logger) *authRepo {
	return &authRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "data.auth_repo")),
	}
}

var _ biz.AuthRepo = (*authRepo)(nil)

// =======================
// user
// =======================

func (r *authRepo) GetUserByUsername(ctx context.Context, username string) (*biz.User, error) {
	l := r.log.WithContext(ctx)

	if username == "" {
		l.Warn("GetUserByUsername: empty username")
		return nil, errors.New("username is required")
	}

	u, err := r.data.postgres.User.
		Query().
		Where(entuser.Username(username)).
		Only(ctx)
	if err != nil {
		l.Infof("GetUserByUsername not found username=%s err=%v", username, err)
		return nil, err
	}

	return entUserToBizUser(u), nil
}

func (r *authRepo) GetUserByID(ctx context.Context, id int) (*biz.User, error) {
	l := r.log.WithContext(ctx)
	if id <= 0 {
		l.Warn("GetUserByID: invalid id")
		return nil, errors.New("user id is required")
	}

	u, err := r.data.postgres.User.
		Query().
		Where(entuser.ID(id)).
		Only(ctx)
	if err != nil {
		l.Infof("GetUserByID not found id=%d err=%v", id, err)
		return nil, err
	}

	return entUserToBizUser(u), nil
}

func (r *authRepo) GetUserByOAuthIdentity(ctx context.Context, provider, subject string) (*biz.User, error) {
	l := r.log.WithContext(ctx)
	if provider == "" || subject == "" {
		l.Warn("GetUserByOAuthIdentity: empty provider or subject")
		return nil, errors.New("oauth provider and subject are required")
	}

	var (
		id               int
		uname            string
		passwordHash     string
		oauthProvider    sql.NullString
		oauthSubject     sql.NullString
		oauthEmail       sql.NullString
		oauthDisplayName sql.NullString
		disabled         bool
		lastLoginAt      sql.NullTime
		createdAt        time.Time
		updatedAt        time.Time
	)
	err := r.data.sqldb.QueryRowContext(
		ctx,
		"SELECT id, username, password_hash, oauth_provider, oauth_subject, oauth_email, oauth_display_name, disabled, last_login_at, created_at, updated_at FROM users WHERE oauth_provider = $1 AND oauth_subject = $2 LIMIT 1",
		provider,
		subject,
	).Scan(&id, &uname, &passwordHash, &oauthProvider, &oauthSubject, &oauthEmail, &oauthDisplayName, &disabled, &lastLoginAt, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			l.Infof("GetUserByOAuthIdentity not found provider=%s", provider)
		} else {
			l.Errorf("GetUserByOAuthIdentity failed provider=%s err=%v", provider, err)
		}
		return nil, err
	}

	return userRowToBiz(id, uname, passwordHash, oauthProvider, oauthSubject, oauthEmail, oauthDisplayName, disabled, lastLoginAt, createdAt, updatedAt), nil
}

func (r *authRepo) CreateUser(ctx context.Context, in *biz.User) (*biz.User, error) {
	l := r.log.WithContext(ctx)

	l.Infof("CreateUser start username=%s", in.Username)

	// 关键兜底：账号名在 users/admin_users 间必须全局唯一，避免用户与管理员同名。
	if exists, err := r.isUsernameUsedByAdmin(ctx, in.Username); err != nil {
		l.Errorf("CreateUser check admin username failed username=%s err=%v", in.Username, err)
		return nil, err
	} else if exists {
		l.Warnf("CreateUser username conflicts with admin username=%s", in.Username)
		return nil, biz.ErrUserExists
	}

	m := r.data.postgres.User.
		Create().
		SetUsername(in.Username).
		SetPasswordHash(in.PasswordHash)
	m.SetNillableOauthProvider(in.OAuthProvider)
	m.SetNillableOauthSubject(in.OAuthSubject)
	m.SetNillableOauthEmail(in.OAuthEmail)
	m.SetNillableOauthDisplayName(in.OAuthName)

	u, err := m.Save(ctx)
	if err != nil {
		if isDuplicateUsernameConstraint(err) {
			l.Warnf("CreateUser duplicate username username=%s err=%v", in.Username, err)
			return nil, biz.ErrUserExists
		}
		l.Errorf("CreateUser failed err=%v", err)
		return nil, err
	}

	return entUserToBizUser(u), nil
}

func (r *authRepo) BindUserOAuthIdentity(ctx context.Context, id int, identity biz.OAuthIdentity) error {
	if id <= 0 {
		return errors.New("user id is required")
	}
	if identity.Provider == "" || identity.Subject == "" {
		return errors.New("oauth provider and subject are required")
	}

	_, err := r.data.sqldb.ExecContext(
		ctx,
		"UPDATE users SET oauth_provider = $1, oauth_subject = $2, oauth_email = $3, oauth_display_name = $4, updated_at = $5 WHERE id = $6",
		identity.Provider,
		identity.Subject,
		nullString(identity.Email),
		nullString(identity.Name),
		time.Now(),
		id,
	)
	if err != nil {
		r.log.WithContext(ctx).Errorf("BindUserOAuthIdentity failed user_id=%d provider=%s err=%v", id, identity.Provider, err)
	}
	return err
}

func (r *authRepo) UpdateUserLastLogin(ctx context.Context, id int, t time.Time) error {
	l := r.log.WithContext(ctx)

	_, err := r.data.postgres.User.
		UpdateOneID(id).
		SetLastLoginAt(t).
		SetUpdatedAt(time.Now()).
		Save(ctx)

	if err != nil {
		l.Errorf("UpdateUserLastLogin failed user_id=%d err=%v", id, err)
	}

	return err
}

func (r *authRepo) isUsernameUsedByAdmin(ctx context.Context, username string) (bool, error) {
	if username == "" {
		return false, nil
	}
	return r.data.postgres.AdminUser.
		Query().
		Where(entadminuser.Username(username)).
		Exist(ctx)
}

func entUserToBizUser(u *entmodel.User) *biz.User {
	if u == nil {
		return nil
	}
	var lastLoginAt *time.Time
	if u.LastLoginAt != nil {
		t := *u.LastLoginAt
		lastLoginAt = &t
	}
	return &biz.User{
		ID:            u.ID,
		Username:      u.Username,
		PasswordHash:  u.PasswordHash,
		Disabled:      u.Disabled,
		OAuthProvider: u.OauthProvider,
		OAuthSubject:  u.OauthSubject,
		OAuthEmail:    u.OauthEmail,
		OAuthName:     u.OauthDisplayName,
		Role:          int8(biz.RoleUser),
		LastLoginAt:   lastLoginAt,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}

func userRowToBiz(id int, username, passwordHash string, oauthProvider, oauthSubject, oauthEmail, oauthDisplayName sql.NullString, disabled bool, lastLoginAt sql.NullTime, createdAt, updatedAt time.Time) *biz.User {
	var lastLogin *time.Time
	if lastLoginAt.Valid {
		t := lastLoginAt.Time
		lastLogin = &t
	}
	return &biz.User{
		ID:            id,
		Username:      username,
		PasswordHash:  passwordHash,
		Disabled:      disabled,
		OAuthProvider: stringPtrFromNull(oauthProvider),
		OAuthSubject:  stringPtrFromNull(oauthSubject),
		OAuthEmail:    stringPtrFromNull(oauthEmail),
		OAuthName:     stringPtrFromNull(oauthDisplayName),
		Role:          int8(biz.RoleUser),
		LastLoginAt:   lastLogin,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}
