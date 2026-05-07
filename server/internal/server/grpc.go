package server

import (
	v1 "server/api/jsonrpc/v1"
	"server/internal/conf"
	"server/internal/data"
	"server/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"go.opentelemetry.io/otel/sdk/trace"
)

// NewGRPCServer 创建并注册 Jsonrpc gRPC 服务。
func NewGRPCServer(
	c *conf.Server,
	logger log.Logger,
	jsonrpcSvc *service.JsonrpcService,
	tracerProvider *trace.TracerProvider,
	data *data.Data,
) *grpc.Server {
	var opts []grpc.ServerOption

	// 全局中间件
	opts = append(opts,
		grpc.Middleware(
			recovery.Recovery(),
			tracing.Server(
				tracing.WithTracerProvider(tracerProvider),
			),
			logging.Server(logger),
			ratelimit.Server(),
		),
	)

	// 端口 / 网络配置
	if c.Grpc != nil {
		if c.Grpc.Network != "" {
			opts = append(opts, grpc.Network(c.Grpc.Network))
		}
		if c.Grpc.Addr != "" {
			opts = append(opts, grpc.Address(c.Grpc.Addr))
		}
		if c.Grpc.Timeout != nil {
			opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
		}
	}

	opts = append(opts, grpc.Logger(logger))

	srv := grpc.NewServer(opts...)

	// 这里用的是 protoc --go-grpc_out 生成的注册函数
	v1.RegisterJsonrpcServer(srv, jsonrpcSvc)

	return srv
}
