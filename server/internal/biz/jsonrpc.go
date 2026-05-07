package biz

import (
	"context"
	v1 "server/api/jsonrpc/v1"

	"github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/protobuf/types/known/structpb"
)

// JsonrpcRepo 是 biz 层看到的“data 抽象接口”
type JsonrpcRepo interface {
	Handle(ctx context.Context,
		url, jsonrpc, method, id string,
		params *structpb.Struct,
	) (string, *v1.JsonrpcResult, error)
}

// JsonrpcUsecase 负责业务逻辑：日志 + 校验 + 调用 Repo
type JsonrpcUsecase struct {
	repo JsonrpcRepo
	log  *log.Helper
	tp   *trace.TracerProvider
}

func NewJsonrpcUsecase(
	repo JsonrpcRepo,
	logger log.Logger,
	tp *trace.TracerProvider,
) *JsonrpcUsecase {
	return &JsonrpcUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
		tp:   tp,
	}
}

// Handle 是 service 层调用的统一入口（GET/POST 最终都到这里）
func (uc *JsonrpcUsecase) Handle(
	ctx context.Context,
	url, jsonrpc, method, id string,
	params *structpb.Struct,
) (string, *v1.JsonrpcResult, error) {
	if jsonrpc == "" {
		jsonrpc = "2.0"
	}

	uc.log.WithContext(ctx).Infof(
		"[biz] Jsonrpc Handle url=%s jsonrpc=%s method=%s id=%s",
		url, jsonrpc, method, id,
	)

	// 这里可以做统一的业务校验、鉴权等，然后交给 repo
	return uc.repo.Handle(ctx, url, jsonrpc, method, id, params)
}
