// server/internal/data/token.go
package data

import (
	"time"

	"server/internal/biz"
	"server/internal/conf"
	jwtutil "server/pkg/jwt"

	"github.com/go-kratos/kratos/v2/log"
)

// NewTokenGenerator 提供 biz.TokenGenerator 给 wire
// 目标：token 方案可替换（JWT/Session/第三方）而不改 biz
func NewTokenGenerator(c *conf.Data, logger log.Logger) biz.TokenGenerator {
	l := log.NewHelper(log.With(logger, "module", "data.token"))

	// 避免 nil pointer
	if c == nil || c.Auth == nil || c.Auth.JwtSecret == "" {
		panic("NewTokenGenerator: missing data.auth.jwt_secret in config")
	}

	secret := []byte(c.Auth.JwtSecret)

	// token 过期时间：默认 7 天
	exp := 7 * 24 * time.Hour
	if c.Auth.JwtExpireSeconds > 0 {
		exp = time.Duration(c.Auth.JwtExpireSeconds) * time.Second
	}

	cfg := jwtutil.Config{
		Secret:         secret,
		ExpireDuration: exp,
	}

	l.Infof("token generator init ok, expire=%s", exp)

	// 返回闭包：符合 biz.TokenGenerator(userID, username, role)
	return func(userID int, username string, role int8) (string, time.Time, error) {
		l.Infof("gen token uid=%d uname=%s role=%d", userID, username, role)
		return jwtutil.NewToken(cfg, userID, username, role)
	}
}
