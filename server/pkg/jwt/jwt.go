// server/pkg/jwt/jwt.go
package jwtutil

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   int    `json:"uid"`
	Username string `json:"uname"`

	// 0=user, 1=admin
	Role int8 `json:"role"`

	jwt.RegisteredClaims
}

type Config struct {
	Secret         []byte        // HMAC 秘钥
	ExpireDuration time.Duration // 过期时间，比如 7 * 24 * time.Hour
}

func NewToken(cfg Config, userID int, username string, role int8) (string, time.Time, error) {
	expireAt := time.Now().Add(cfg.ExpireDuration)

	claims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "access_token",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(cfg.Secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expireAt, nil
}

func ParseToken(secret []byte, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}
