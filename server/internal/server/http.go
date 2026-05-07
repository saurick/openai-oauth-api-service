// server/internal/server/http.go
package server

import (
	httpx "github.com/go-kratos/kratos/v2/transport/http"

	v1 "server/api/jsonrpc/v1"
	"server/internal/biz"
	"server/internal/conf"
	"server/internal/data"
	"server/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func NewHTTPServer(
	c *conf.Server,
	logger log.Logger,
	jsonrpcSvc *service.JsonrpcService,
	gatewayUC *biz.GatewayUsecase,
	adminAuthUC *biz.AdminAuthUsecase,
	authUC *biz.AuthUsecase,
	tp *sdktrace.TracerProvider,
	data *data.Data,

	// Data 配置提供 JWT secret 等运行时依赖。
	dc *conf.Data,
) *httpx.Server {
	var opts = []httpx.ServerOption{
		httpx.Filter(RequestIDFilter()),
		httpx.Middleware(
			recovery.Recovery(),
			tracing.Server(tracing.WithTracerProvider(tp)),
			logging.Server(log.With(logger, "logger.name", "server.http")),
			// 默认 bbr limiter
			ratelimit.Server(),
			// 统一从请求头解析 JWT，并把 AuthClaims 写入请求上下文。
			AuthClaimsMiddleware(dc, logger),
		),
	}

	if c.Http != nil {
		if c.Http.Network != "" {
			opts = append(opts, httpx.Network(c.Http.Network))
		}
		if c.Http.Addr != "" {
			opts = append(opts, httpx.Address(c.Http.Addr))
		}
		if c.Http.Timeout != nil {
			opts = append(opts, httpx.Timeout(c.Http.Timeout.AsDuration()))
		}
	}

	opts = append(opts, httpx.Logger(logger))

	srv := httpx.NewServer(opts...)

	// ===== JSON-RPC HTTP 路由 =====
	// 这里用的是 protoc --go-http_out 生成的注册函数
	v1.RegisterJsonrpcHTTPServer(srv, jsonrpcSvc)

	registerHealthRoutes(srv, logger, tp, data.SQLDB())
	registerOpenAIGatewayRoutes(srv, logger, tp, gatewayUC, dc)
	registerOAuthRoutes(srv, logger, tp, dc, adminAuthUC, authUC)
	registerAdminExportRoutes(srv, logger, tp, gatewayUC, dc)
	registerStaticHandler(srv, logger, tp)

	return srv
}
