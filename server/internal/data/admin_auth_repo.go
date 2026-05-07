// server/internal/data/admin_auth_repo.go
package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"server/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type adminAuthRepo struct {
	data *Data
	log  *log.Helper
}

func NewAdminAuthRepo(data *Data, logger log.Logger) *adminAuthRepo {
	return &adminAuthRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "data.admin_auth_repo")),
	}
}

var _ biz.AdminAuthRepo = (*adminAuthRepo)(nil)

func (r *adminAuthRepo) GetAdminByID(ctx context.Context, id int) (*biz.AdminUser, error) {
	l := r.log.WithContext(ctx)
	if id <= 0 {
		l.Warn("GetAdminByID: invalid id")
		return nil, errors.New("admin id is required")
	}

	var (
		adminID      int
		uname        string
		passwordHash string
		disabled     bool
	)

	err := r.data.sqldb.QueryRowContext(
		ctx,
		"SELECT id, username, password_hash, disabled FROM admin_users WHERE id = $1 LIMIT 1",
		id,
	).Scan(&adminID, &uname, &passwordHash, &disabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			l.Infof("GetAdminByID not found id=%d", id)
		} else {
			l.Errorf("GetAdminByID failed id=%d err=%v", id, err)
		}
		return nil, err
	}

	return &biz.AdminUser{ID: adminID, Username: uname, PasswordHash: passwordHash, Disabled: disabled}, nil
}

func (r *adminAuthRepo) GetAdminByUsername(ctx context.Context, username string) (*biz.AdminUser, error) {
	l := r.log.WithContext(ctx)
	if username == "" {
		l.Warn("GetAdminByUsername: empty username")
		return nil, errors.New("username is required")
	}

	var (
		id           int
		uname        string
		passwordHash string
		disabled     bool
	)

	err := r.data.sqldb.QueryRowContext(
		ctx,
		"SELECT id, username, password_hash, disabled FROM admin_users WHERE username = $1 LIMIT 1",
		username,
	).Scan(&id, &uname, &passwordHash, &disabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			l.Infof("GetAdminByUsername not found username=%s", username)
		} else {
			l.Errorf("GetAdminByUsername failed username=%s err=%v", username, err)
		}
		return nil, err
	}

	return &biz.AdminUser{ID: id, Username: uname, PasswordHash: passwordHash, Disabled: disabled}, nil
}

func (r *adminAuthRepo) GetAdminByOAuthIdentity(ctx context.Context, provider, subject string) (*biz.AdminUser, error) {
	l := r.log.WithContext(ctx)
	if provider == "" || subject == "" {
		l.Warn("GetAdminByOAuthIdentity: empty provider or subject")
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
	)

	err := r.data.sqldb.QueryRowContext(
		ctx,
		"SELECT id, username, password_hash, oauth_provider, oauth_subject, oauth_email, oauth_display_name, disabled FROM admin_users WHERE oauth_provider = $1 AND oauth_subject = $2 LIMIT 1",
		provider,
		subject,
	).Scan(&id, &uname, &passwordHash, &oauthProvider, &oauthSubject, &oauthEmail, &oauthDisplayName, &disabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			l.Infof("GetAdminByOAuthIdentity not found provider=%s", provider)
		} else {
			l.Errorf("GetAdminByOAuthIdentity failed provider=%s err=%v", provider, err)
		}
		return nil, err
	}

	return adminRowToBiz(id, uname, passwordHash, oauthProvider, oauthSubject, oauthEmail, oauthDisplayName, disabled), nil
}

func (r *adminAuthRepo) BindAdminOAuthIdentity(ctx context.Context, id int, identity biz.OAuthIdentity) error {
	if id <= 0 {
		return errors.New("admin id is required")
	}
	if identity.Provider == "" || identity.Subject == "" {
		return errors.New("oauth provider and subject are required")
	}

	_, err := r.data.sqldb.ExecContext(
		ctx,
		"UPDATE admin_users SET oauth_provider = $1, oauth_subject = $2, oauth_email = $3, oauth_display_name = $4, updated_at = $5 WHERE id = $6",
		identity.Provider,
		identity.Subject,
		nullString(identity.Email),
		nullString(identity.Name),
		time.Now(),
		id,
	)
	if err != nil {
		r.log.WithContext(ctx).Errorf("BindAdminOAuthIdentity failed admin_id=%d provider=%s err=%v", id, identity.Provider, err)
	}
	return err
}

func (r *adminAuthRepo) UpdateAdminLastLogin(ctx context.Context, id int, t time.Time) error {
	if id <= 0 {
		return errors.New("admin id is required")
	}

	_, err := r.data.sqldb.ExecContext(
		ctx,
		"UPDATE admin_users SET last_login_at = $1, updated_at = $2 WHERE id = $3",
		t,
		time.Now(),
		id,
	)
	if err != nil {
		r.log.WithContext(ctx).Errorf("UpdateAdminLastLogin failed admin_id=%d err=%v", id, err)
	}
	return err
}

func adminRowToBiz(id int, username, passwordHash string, oauthProvider, oauthSubject, oauthEmail, oauthDisplayName sql.NullString, disabled bool) *biz.AdminUser {
	return &biz.AdminUser{
		ID:            id,
		Username:      username,
		PasswordHash:  passwordHash,
		OAuthProvider: stringPtrFromNull(oauthProvider),
		OAuthSubject:  stringPtrFromNull(oauthSubject),
		OAuthEmail:    stringPtrFromNull(oauthEmail),
		OAuthName:     stringPtrFromNull(oauthDisplayName),
		Disabled:      disabled,
	}
}

func stringPtrFromNull(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func nullString(v string) sql.NullString {
	if v == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: v, Valid: true}
}
