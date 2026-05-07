// server/internal/service/jsonrpc.go
package service

import (
	"context"
	"time"

	v1 "server/api/jsonrpc/v1"
	"server/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

// JsonrpcService 实现 v1.JsonrpcServer 接口。
type JsonrpcService struct {
	v1.UnimplementedJsonrpcServer

	uc  *biz.JsonrpcUsecase
	log *log.Helper
}

func NewJsonrpcService(uc *biz.JsonrpcUsecase, logger log.Logger) *JsonrpcService {
	return &JsonrpcService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

// GetJsonrpc 对应 GET /rpc/{url}
func (s *JsonrpcService) GetJsonrpc(ctx context.Context, req *v1.GetJsonrpcRequest) (*v1.GetJsonrpcReply, error) {
	s.log.WithContext(ctx).Infof(
		"GetJsonrpc: url=%s jsonrpc=%s method=%s id=%s",
		req.GetUrl(), req.GetJsonrpc(), req.GetMethod(), req.GetId(),
	)

	id, result, bizErr := s.uc.Handle(
		ctx,
		req.GetUrl(),
		req.GetJsonrpc(),
		req.GetMethod(),
		req.GetId(),
		req.GetParams(),
	)

	reply := &v1.GetJsonrpcReply{
		Jsonrpc: "2.0",
		Id:      id,
		Result:  result,
	}

	if bizErr != nil {
		reply.Error = bizErr.Error()
	}

	return reply, nil
}

// PostJsonrpc 对应 POST /rpc/{url}
func (s *JsonrpcService) PostJsonrpc(ctx context.Context, req *v1.PostJsonrpcRequest) (*v1.PostJsonrpcReply, error) {
	start := time.Now()
	defer func() {
		s.log.WithContext(ctx).Infof(
			"PostJsonrpc: done url=%s method=%s id=%s cost=%s",
			req.GetUrl(), req.GetMethod(), req.GetId(), time.Since(start),
		)
	}()

	s.log.WithContext(ctx).Infof(
		"PostJsonrpc: url=%s jsonrpc=%s method=%s id=%s",
		req.GetUrl(), req.GetJsonrpc(), req.GetMethod(), req.GetId(),
	)

	id, result, bizErr := s.uc.Handle(
		ctx,
		req.GetUrl(),
		req.GetJsonrpc(),
		req.GetMethod(),
		req.GetId(),
		req.GetParams(),
	)

	reply := &v1.PostJsonrpcReply{
		Jsonrpc: "2.0",
		Id:      id,
		Result:  result,
	}

	if bizErr != nil {
		reply.Error = bizErr.Error()
	}

	return reply, nil
}
