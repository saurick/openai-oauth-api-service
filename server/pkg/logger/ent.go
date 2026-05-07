package logger

import (
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/jwalton/gchalk"
)

// 将ent输出的日志流解析成结构化日志输出
func NewEntLogger(l log.Logger) func(...any) {
	return func(a ...any) {
		s := fmt.Sprint(a...)

		msg, s, ok := strings.Cut(s, ": ")
		if !ok {
			_ = l.Log(log.LevelDebug, a...)
			return
		}

		s, arg, ok := strings.Cut(s, " args=")
		if !ok {
			_ = l.Log(log.LevelDebug, a...)
			return
		}

		_, query, ok := strings.Cut(s, "query=")
		if !ok {
			_ = l.Log(log.LevelDebug, a...)
			return
		}

		if arg == "" || query == "" {
			_ = l.Log(log.LevelDebug, a...)
			return
		}

		_ = l.Log(
			log.LevelDebug,
			"msg", msg,
			"query", gchalk.BgBrightBlack(query), // 添加高亮灰色背景
			"args", arg,
		)
	}
}
