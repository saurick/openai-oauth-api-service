package data

import (
	"strings"

	"server/internal/data/model/ent"
)

func isDuplicateUsernameConstraint(err error) bool {
	return isDuplicateUniqueConstraint(err, "user_username", "users.username", "username")
}

func isDuplicateOAuthIdentityConstraint(err error) bool {
	return isDuplicateUniqueConstraint(err, "oauth_provider", "oauth_subject")
}

func isDuplicateUniqueConstraint(err error, keys ...string) bool {
	if err == nil || !ent.IsConstraintError(err) {
		return false
	}

	// 仅识别“唯一键冲突”语义，兼容 Postgres/Ent 的 duplicate/unique 文案，避免把外键等其他约束错误误判为“名称重复”。
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "duplicate") && !strings.Contains(msg, "unique") {
		return false
	}

	for _, key := range keys {
		if strings.Contains(msg, strings.ToLower(key)) {
			return true
		}
	}
	return false
}
