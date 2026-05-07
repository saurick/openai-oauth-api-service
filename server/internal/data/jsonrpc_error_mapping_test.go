package data

import (
	"context"
	"io"
	"testing"

	"server/internal/biz"
	"server/internal/errcode"

	"github.com/go-kratos/kratos/v2/log"
)

type stubAdminAccountReader struct {
	admin *biz.AdminUser
	err   error
}

func (s stubAdminAccountReader) GetAdminByID(_ context.Context, _ int) (*biz.AdminUser, error) {
	return s.admin, s.err
}

func TestJsonrpcData_AuthMe_UnauthorizedUsesAuthRequired(t *testing.T) {
	j := &JsonrpcData{
		log: log.NewHelper(log.With(log.NewStdLogger(io.Discard), "module", "data.jsonrpc.test")),
	}

	_, res, err := j.handleAuth(context.Background(), "me", "1", nil)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if res == nil {
		t.Fatalf("expected result not nil")
	}
	if res.Code != errcode.AuthRequired.Code {
		t.Fatalf("expected code=%d, got %d", errcode.AuthRequired.Code, res.Code)
	}
}

func TestJsonrpcData_ErrorMappers_NoPermissionUsesPermissionDenied(t *testing.T) {
	j := &JsonrpcData{
		log: log.NewHelper(log.With(log.NewStdLogger(io.Discard), "module", "data.jsonrpc.test")),
	}

	userRes := j.mapUserAdminError(context.Background(), biz.ErrNoPermission)
	if userRes == nil {
		t.Fatalf("expected user result not nil")
	}
	if userRes.Code != errcode.PermissionDenied.Code {
		t.Fatalf("expected user code=%d, got %d", errcode.PermissionDenied.Code, userRes.Code)
	}
}

func TestJsonrpcData_RequireAdmin_DisabledAdminUsesAdminDisabled(t *testing.T) {
	j := &JsonrpcData{
		log:         log.NewHelper(log.With(log.NewStdLogger(io.Discard), "module", "data.jsonrpc.test")),
		adminReader: stubAdminAccountReader{admin: &biz.AdminUser{ID: 1, Username: "admin", Disabled: true}},
	}

	ctx := biz.NewContextWithClaims(context.Background(), &biz.AuthClaims{
		UserID:   1,
		Username: "admin",
		Role:     biz.RoleAdmin,
	})

	_, adminRes := j.requireAdmin(ctx)
	if adminRes == nil {
		t.Fatalf("expected admin result not nil")
	}
	if adminRes.Code != errcode.AdminDisabled.Code {
		t.Fatalf("expected admin code=%d, got %d", errcode.AdminDisabled.Code, adminRes.Code)
	}
}
