// mock
package biz

import (
	"context"
	"testing"

	v1 "server/api/jsonrpc/v1"
	"server/pkg/logger"

	"google.golang.org/protobuf/types/known/structpb"
)

type mockJsonrpcRepo struct {
	lastURL    string
	lastMethod string
	result     *v1.JsonrpcResult
	err        error
}

func (m *mockJsonrpcRepo) Handle(
	ctx context.Context,
	url, jsonrpc, method, id string,
	params *structpb.Struct,
) (string, *v1.JsonrpcResult, error) {
	m.lastURL = url
	m.lastMethod = method
	return id, m.result, m.err
}

func TestJsonrpcUsecase_Ping(t *testing.T) {
	// 构造一个 mock repo，预设返回值
	mrepo := &mockJsonrpcRepo{
		result: &v1.JsonrpcResult{
			Code:    0,
			Message: "OK",
		},
	}

	logger := logger.NewDefaultLogger("", "test", "test-version", true)
	uc := NewJsonrpcUsecase(mrepo, logger, nil)

	ctx := context.Background()
	id, res, err := uc.Handle(ctx,
		"system",  // url
		"2.0",     // jsonrpc
		"ping",    // method
		"test-id", // id
		nil,       // params
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "test-id" {
		t.Fatalf("unexpected id: %s", id)
	}
	if res.GetCode() != 0 || res.GetMessage() != "OK" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if mrepo.lastURL != "system" || mrepo.lastMethod != "ping" {
		t.Fatalf("repo was called with wrong args: url=%s method=%s",
			mrepo.lastURL, mrepo.lastMethod)
	}
}
