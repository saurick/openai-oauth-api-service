// server/internal/data/admin_user_init.go
package data

import (
	"context"
	"errors"
	"time"

	"server/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/crypto/bcrypt"
)

const adminUserInitSQL = "INSERT INTO admin_users (username, password_hash, disabled, created_at, updated_at) VALUES ($1, $2, FALSE, $3, $4) ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash, updated_at = EXCLUDED.updated_at"

func InitAdminUsersIfNeeded(ctx context.Context, d *Data, cfg *conf.Data, l *log.Helper) error {
	if d == nil || d.sqldb == nil {
		return errors.New("InitAdminUsersIfNeeded: missing db")
	}

	if cfg == nil || cfg.Auth == nil || cfg.Auth.Admin == nil {
		return nil
	}

	username := cfg.Auth.Admin.Username
	password := cfg.Auth.Admin.Password

	if username == "" || password == "" {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	now := time.Now()
	// 配置里的默认管理员密码是初始化账号的当前真源；账号已存在时只同步密码，不改禁用状态。
	result, err := d.sqldb.ExecContext(
		ctx,
		adminUserInitSQL,
		username,
		string(hash),
		now,
		now,
	)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		l.Warnf("admin_users init rows affected unavailable username=%s err=%v", username, err)
		l.Info("admin_users init completed without rows-affected detail")
		return nil
	}
	if affected == 0 {
		l.Infof("admin_users admin sync affected no rows username=%s", username)
		return nil
	}

	l.Info("sync admin_users admin success")
	return nil
}
