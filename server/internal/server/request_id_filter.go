package server

import (
	"crypto/rand"
	"encoding/hex"
	stdhttp "net/http"
	"time"

	pkglogger "server/pkg/logger"

	httpx "github.com/go-kratos/kratos/v2/transport/http"
)

const requestIDHeader = "X-Request-Id"

func newRequestID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err == nil {
		return hex.EncodeToString(buf[:])
	}
	// 兜底只在极端情况下使用，保证即使随机源异常也不会丢 request_id。
	return hex.EncodeToString([]byte(time.Now().UTC().Format("20060102150405.000000000")))
}

// RequestIDFilter 统一透传或生成 request_id，回写响应头并注入 context，避免所有链路都依赖上游服务显式传值。
func RequestIDFilter() httpx.FilterFunc {
	return func(next stdhttp.Handler) stdhttp.Handler {
		return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			requestID := requestIDFromRequest(r)
			if requestID == "" {
				requestID = newRequestID()
			}

			r.Header.Set(requestIDHeader, requestID)
			w.Header().Set(requestIDHeader, requestID)
			ctx := pkglogger.WithRequestID(r.Context(), requestID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
