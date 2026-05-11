// server/internal/data/jsonrpc_auth_test.go
package data

import (
	"context"
	"errors"
	"io"
	"reflect"
	"sync"
	"testing"
	"time"

	"server/internal/biz"
	"server/internal/errcode"

	"github.com/go-kratos/kratos/v2/log"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRedactJSONRPCParamsMasksSensitiveFields(t *testing.T) {
	input := map[string]any{
		"username": "admin",
		"password": "adminadmin",
		"nested": map[string]any{
			"access_token": "token-value",
			"key_id":       7,
		},
		"items": []any{
			map[string]any{"plain_key": "ogw_secret"},
		},
	}

	got := redactJSONRPCParams(input)
	want := map[string]any{
		"username": "admin",
		"password": "[REDACTED]",
		"nested": map[string]any{
			"access_token": "[REDACTED]",
			"key_id":       7,
		},
		"items": []any{
			map[string]any{"plain_key": "[REDACTED]"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("redactJSONRPCParams() = %#v, want %#v", got, want)
	}
}

func TestJsonrpcData_AuthLogin_OK(t *testing.T) {
	repo := newMemAuthRepoForData()
	_ = repo.putUser("alice", "p@ss", false)

	logger := log.NewStdLogger(io.Discard)
	tp := tracesdk.NewTracerProvider()

	genTok := func(userID int, username string, role int8) (string, time.Time, error) {
		return "tok", time.Now().Add(time.Hour), nil
	}
	authUC := biz.NewAuthUsecase(repo, genTok, logger, tp)

	j := &JsonrpcData{
		log:    log.NewHelper(log.With(logger, "module", "data.jsonrpc.test")),
		authUC: authUC,
	}

	params, _ := structpb.NewStruct(map[string]any{
		"username": "alice",
		"password": "p@ss",
	})

	_, res, err := j.handleAuth(context.Background(), "login", "1", params)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if res == nil || res.Code != 0 {
		t.Fatalf("expected code=0, got %+v", res)
	}
	if res.Data == nil {
		t.Fatalf("expected data not nil")
	}
	m := res.Data.AsMap()
	if m["access_token"] != "tok" {
		t.Fatalf("expected access_token=tok, got %v", m["access_token"])
	}
	if m["user_id"] == nil {
		t.Fatalf("expected user_id not nil")
	}
}

func TestJsonrpcData_AuthRegister_Disabled(t *testing.T) {
	repo := newMemAuthRepoForData()

	logger := log.NewStdLogger(io.Discard)
	tp := tracesdk.NewTracerProvider()
	authUC := biz.NewAuthUsecase(repo, func(int, string, int8) (string, time.Time, error) {
		return "tok-reg", time.Now().Add(time.Hour), nil
	}, logger, tp)

	j := &JsonrpcData{
		log:    log.NewHelper(log.With(logger, "module", "data.jsonrpc.test")),
		authUC: authUC,
	}

	params, _ := structpb.NewStruct(map[string]any{
		"username": "bob",
		"password": "p@ss",
	})

	_, res, err := j.handleAuth(context.Background(), "register", "1", params)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if res == nil || res.Code == 0 {
		t.Fatalf("expected non-zero code for disabled register, got %+v", res)
	}
}

func TestJsonrpcData_AuthRegister_MissingArgs(t *testing.T) {
	repo := newMemAuthRepoForData()

	logger := log.NewStdLogger(io.Discard)
	tp := tracesdk.NewTracerProvider()
	authUC := biz.NewAuthUsecase(repo, func(int, string, int8) (string, time.Time, error) {
		return "tok", time.Now().Add(time.Hour), nil
	}, logger, tp)

	j := &JsonrpcData{
		log:    log.NewHelper(log.With(logger, "module", "data.jsonrpc.test")),
		authUC: authUC,
	}

	params, _ := structpb.NewStruct(map[string]any{
		"username": "alice",
	})

	_, res, err := j.handleAuth(context.Background(), "register", "1", params)
	if err != nil {
		t.Fatalf("expected nil err (jsonrpc should map to result), got %v", err)
	}
	if res == nil {
		t.Fatalf("expected result not nil")
	}
	if res.Code == 0 {
		t.Fatalf("expected non-zero code for missing args, got %+v", res)
	}
}

func TestJsonrpcData_AuthLogin_UserNotFound(t *testing.T) {
	repo := newMemAuthRepoForData()

	logger := log.NewStdLogger(io.Discard)
	tp := tracesdk.NewTracerProvider()
	authUC := biz.NewAuthUsecase(repo, func(int, string, int8) (string, time.Time, error) {
		return "tok", time.Now().Add(time.Hour), nil
	}, logger, tp)

	j := &JsonrpcData{
		log:    log.NewHelper(log.With(logger, "module", "data.jsonrpc.test")),
		authUC: authUC,
	}

	params, _ := structpb.NewStruct(map[string]any{
		"username": "notfound",
		"password": "p@ss",
	})

	_, res, err := j.handleAuth(context.Background(), "login", "1", params)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if res == nil || res.Code == 0 {
		t.Fatalf("expected non-zero code for user not found, got %+v", res)
	}
	if res.Code != errcode.AuthUserNotFound.Code {
		t.Fatalf("expected code=%d, got %d", errcode.AuthUserNotFound.Code, res.Code)
	}
}

func TestJsonrpcData_AuthLogin_InvalidPassword(t *testing.T) {
	repo := newMemAuthRepoForData()
	_ = repo.putUser("alice", "p@ss", false)

	logger := log.NewStdLogger(io.Discard)
	tp := tracesdk.NewTracerProvider()
	authUC := biz.NewAuthUsecase(repo, func(int, string, int8) (string, time.Time, error) {
		return "tok", time.Now().Add(time.Hour), nil
	}, logger, tp)

	j := &JsonrpcData{
		log:    log.NewHelper(log.With(logger, "module", "data.jsonrpc.test")),
		authUC: authUC,
	}

	params, _ := structpb.NewStruct(map[string]any{
		"username": "alice",
		"password": "wrong",
	})

	_, res, err := j.handleAuth(context.Background(), "login", "1", params)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if res == nil || res.Code == 0 {
		t.Fatalf("expected non-zero code for invalid password, got %+v", res)
	}
	if res.Code != errcode.AuthInvalidPassword.Code {
		t.Fatalf("expected code=%d, got %d", errcode.AuthInvalidPassword.Code, res.Code)
	}
}

func TestJsonrpcData_AuthLogout(t *testing.T) {
	repo := newMemAuthRepoForData()

	logger := log.NewStdLogger(io.Discard)
	tp := tracesdk.NewTracerProvider()
	authUC := biz.NewAuthUsecase(repo, func(int, string, int8) (string, time.Time, error) {
		return "tok", time.Now().Add(time.Hour), nil
	}, logger, tp)

	j := &JsonrpcData{
		log:    log.NewHelper(log.With(logger, "module", "data.jsonrpc.test")),
		authUC: authUC,
	}

	params, _ := structpb.NewStruct(map[string]any{})

	_, res, err := j.handleAuth(context.Background(), "logout", "1", params)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if res == nil || res.Code != 0 {
		t.Fatalf("expected code=0, got %+v", res)
	}
}

func TestJsonrpcData_AuthUnknownMethod(t *testing.T) {
	repo := newMemAuthRepoForData()

	logger := log.NewStdLogger(io.Discard)
	tp := tracesdk.NewTracerProvider()
	authUC := biz.NewAuthUsecase(repo, func(int, string, int8) (string, time.Time, error) {
		return "tok", time.Now().Add(time.Hour), nil
	}, logger, tp)

	j := &JsonrpcData{
		log:    log.NewHelper(log.With(logger, "module", "data.jsonrpc.test")),
		authUC: authUC,
	}

	params, _ := structpb.NewStruct(map[string]any{})

	_, res, err := j.handleAuth(context.Background(), "unknown", "1", params)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if res == nil || res.Code == 0 {
		t.Fatalf("expected non-zero code for unknown method, got %+v", res)
	}
	if res.Code != errcode.UnknownMethod.Code {
		t.Fatalf("expected code=%d, got %d", errcode.UnknownMethod.Code, res.Code)
	}
}

// ====== 内部：最小 mem repo（data 包里实现 biz.AuthRepo）======

type memAuthRepoForData struct {
	mu         sync.Mutex
	users      map[string]*biz.User
	oauthUsers map[string]*biz.User
	nextUserID int
}

func newMemAuthRepoForData() *memAuthRepoForData {
	return &memAuthRepoForData{
		users:      make(map[string]*biz.User),
		oauthUsers: make(map[string]*biz.User),
		nextUserID: 1,
	}
}

func oauthUserKey(provider, subject string) string {
	return provider + "\x00" + subject
}

func optionalTestStringPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func (r *memAuthRepoForData) putUser(username, password string, disabled bool) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[username] = &biz.User{
		ID:           r.nextUserID,
		Username:     username,
		PasswordHash: string(hash),
		Disabled:     disabled,
		Role:         0,
	}
	r.nextUserID++
	return nil
}

// ====== biz.AuthRepo 实现 ======
func (r *memAuthRepoForData) GetUserByUsername(ctx context.Context, username string) (*biz.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u := r.users[username]
	if u == nil {
		return nil, errors.New("not found")
	}
	cp := *u
	return &cp, nil
}

func (r *memAuthRepoForData) GetUserByID(ctx context.Context, id int) (*biz.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, u := range r.users {
		if u.ID == id {
			cp := *u
			return &cp, nil
		}
	}
	return nil, errors.New("not found")
}

func (r *memAuthRepoForData) GetUserByOAuthIdentity(ctx context.Context, provider, subject string) (*biz.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	u := r.oauthUsers[oauthUserKey(provider, subject)]
	if u == nil {
		return nil, errors.New("not found")
	}
	cp := *u
	return &cp, nil
}

func (r *memAuthRepoForData) CreateUser(ctx context.Context, u *biz.User) (*biz.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.users[u.Username]; ok {
		return nil, errors.New("duplicate username")
	}
	cp := *u
	cp.ID = r.nextUserID
	r.nextUserID++
	r.users[cp.Username] = &cp
	if cp.OAuthProvider != nil && cp.OAuthSubject != nil {
		r.oauthUsers[oauthUserKey(*cp.OAuthProvider, *cp.OAuthSubject)] = &cp
	}
	return &cp, nil
}

func (r *memAuthRepoForData) BindUserOAuthIdentity(ctx context.Context, id int, identity biz.OAuthIdentity) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, u := range r.users {
		if u.ID != id {
			continue
		}
		u.OAuthProvider = optionalTestStringPtr(identity.Provider)
		u.OAuthSubject = optionalTestStringPtr(identity.Subject)
		u.OAuthEmail = optionalTestStringPtr(identity.Email)
		u.OAuthName = optionalTestStringPtr(identity.Name)
		r.oauthUsers[oauthUserKey(identity.Provider, identity.Subject)] = u
		return nil
	}
	return errors.New("not found")
}

func (r *memAuthRepoForData) UpdateUserLastLogin(ctx context.Context, id int, t time.Time) error {
	// 测试中不需要实现
	return nil
}
