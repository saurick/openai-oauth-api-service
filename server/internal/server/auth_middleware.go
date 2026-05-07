// server/internal/server/auth_middleware.go
package server

import (
	"context"
	"errors"
	"strings"

	"server/internal/biz"
	"server/internal/conf"
	jwtutil "server/pkg/jwt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/golang-jwt/jwt/v5"
)

func bearerToken(auth string) string {
	auth = strings.TrimSpace(auth)
	if auth == "" {
		return ""
	}
	if len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

// AuthClaimsMiddleware：解析 JWT -> 注入 ctx claims（不做授权）
func AuthClaimsMiddleware(dc *conf.Data, logger log.Logger) middleware.Middleware {
	helper := log.NewHelper(log.With(logger, "module", "server.auth"))

	if dc == nil || dc.Auth == nil || dc.Auth.JwtSecret == "" {
		helper.Warn("auth middleware disabled (missing data.auth.jwt_secret)")
		return func(next middleware.Handler) middleware.Handler {
			return func(ctx context.Context, req any) (any, error) {
				// 未开启鉴权：视为无登录状态
				ctx = biz.WithAuthState(ctx, biz.AuthNone)
				return next(ctx, req)
			}
		}
	}

	// 单系统统一使用 data.auth.jwtSecret，避免维护两套密钥。
	secret := []byte(dc.Auth.JwtSecret)

	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			// 默认：没登录
			ctx = biz.WithAuthState(ctx, biz.AuthNone)

			tr, ok := transport.FromServerContext(ctx)
			if !ok || tr == nil {
				return next(ctx, req)
			}

			auth := tr.RequestHeader().Get("Authorization")
			tok := bearerToken(auth)
			if tok == "" {
				// 没带 token：AuthNone
				return next(ctx, req)
			}

			claims, err := jwtutil.ParseToken(secret, tok)
			if err == nil && claims != nil {
				ctx = biz.NewContextWithClaims(ctx, &biz.AuthClaims{
					UserID:   claims.UserID,
					Username: claims.Username,
					Role:     biz.Role(claims.Role),
				})
				ctx = biz.WithAuthState(ctx, biz.AuthOK)
				return next(ctx, req)
			}

			// 带了 token 但解析失败：过期 or 无效
			if errors.Is(err, jwt.ErrTokenExpired) {
				ctx = biz.WithAuthState(ctx, biz.AuthExpired)
				helper.WithContext(ctx).Warn("token expired")
			} else {
				ctx = biz.WithAuthState(ctx, biz.AuthInvalid)
				helper.WithContext(ctx).Warnf("parse token failed: %v", err)
			}

			return next(ctx, req)
		}
	}
}

// AdminAuthClaimsMiddleware：解析管理员 JWT -> 注入 ctx claims（不做授权）
// 当前与普通用户统一使用 data.auth.jwtSecret。
func AdminAuthClaimsMiddleware(dc *conf.Data, logger log.Logger) middleware.Middleware {
	helper := log.NewHelper(log.With(logger, "module", "server.admin_auth"))

	if dc == nil || dc.Auth == nil || dc.Auth.JwtSecret == "" {
		helper.Warn("admin auth middleware disabled (missing data.auth.jwt_secret)")
		return func(next middleware.Handler) middleware.Handler {
			return func(ctx context.Context, req any) (any, error) {
				ctx = biz.WithAuthState(ctx, biz.AuthNone)
				return next(ctx, req)
			}
		}
	}

	secret := []byte(dc.Auth.JwtSecret)

	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			ctx = biz.WithAuthState(ctx, biz.AuthNone)

			tr, ok := transport.FromServerContext(ctx)
			if !ok || tr == nil {
				return next(ctx, req)
			}

			auth := tr.RequestHeader().Get("Authorization")
			tok := bearerToken(auth)
			if tok == "" {
				return next(ctx, req)
			}

			claims, err := jwtutil.ParseToken(secret, tok)
			if err == nil && claims != nil {
				ctx = biz.NewContextWithClaims(ctx, &biz.AuthClaims{
					UserID:   claims.UserID,
					Username: claims.Username,
					Role:     biz.Role(claims.Role),
				})
				ctx = biz.WithAuthState(ctx, biz.AuthOK)
				return next(ctx, req)
			}

			if errors.Is(err, jwt.ErrTokenExpired) {
				ctx = biz.WithAuthState(ctx, biz.AuthExpired)
				helper.WithContext(ctx).Warn("token expired")
			} else {
				ctx = biz.WithAuthState(ctx, biz.AuthInvalid)
				helper.WithContext(ctx).Warnf("parse token failed: %v", err)
			}

			return next(ctx, req)
		}
	}
}
