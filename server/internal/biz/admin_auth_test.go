package biz

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/crypto/bcrypt"
)

type memAdminAuthRepo struct {
	mu       sync.Mutex
	byName   map[string]*AdminUser
	byOAuth  map[string]*AdminUser
	bindings int
}

func newMemAdminAuthRepo() *memAdminAuthRepo {
	return &memAdminAuthRepo{
		byName:  make(map[string]*AdminUser),
		byOAuth: make(map[string]*AdminUser),
	}
}

func adminOAuthKey(provider, subject string) string {
	return provider + "\x00" + subject
}

func (r *memAdminAuthRepo) put(username string, disabled bool) *AdminUser {
	r.mu.Lock()
	defer r.mu.Unlock()

	hash, _ := bcrypt.GenerateFromPassword([]byte("p@ss"), bcrypt.DefaultCost)
	admin := &AdminUser{
		ID:           len(r.byName) + 1,
		Username:     username,
		PasswordHash: string(hash),
		Disabled:     disabled,
	}
	r.byName[username] = admin
	return admin
}

func (r *memAdminAuthRepo) GetAdminByUsername(ctx context.Context, username string) (*AdminUser, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	admin := r.byName[username]
	if admin == nil {
		return nil, errors.New("not found")
	}
	cp := *admin
	return &cp, nil
}

func (r *memAdminAuthRepo) GetAdminByOAuthIdentity(ctx context.Context, provider, subject string) (*AdminUser, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	admin := r.byOAuth[adminOAuthKey(provider, subject)]
	if admin == nil {
		return nil, errors.New("not found")
	}
	cp := *admin
	return &cp, nil
}

func (r *memAdminAuthRepo) BindAdminOAuthIdentity(ctx context.Context, id int, identity OAuthIdentity) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, admin := range r.byName {
		if admin.ID == id {
			admin.OAuthProvider = &identity.Provider
			admin.OAuthSubject = &identity.Subject
			admin.OAuthEmail = &identity.Email
			admin.OAuthName = &identity.Name
			r.byOAuth[adminOAuthKey(identity.Provider, identity.Subject)] = admin
			r.bindings++
			return nil
		}
	}
	return errors.New("not found")
}

func (r *memAdminAuthRepo) UpdateAdminLastLogin(ctx context.Context, id int, t time.Time) error {
	return nil
}

func TestAdminAuthUsecase_LoginWithOAuth_BindsExistingEmailAdmin(t *testing.T) {
	repo := newMemAdminAuthRepo()
	repo.put("alice@example.com", false)

	uc := NewAdminAuthUsecase(repo, func(userID int, username string, role int8) (string, time.Time, error) {
		if username != "alice@example.com" {
			t.Fatalf("unexpected username %q", username)
		}
		return "admin-oauth-token", time.Now().Add(time.Hour), nil
	}, log.NewStdLogger(io.Discard), tracesdk.NewTracerProvider())

	token, _, admin, err := uc.LoginWithOAuth(context.Background(), OAuthIdentity{
		Provider: "google",
		Subject:  "sub-1",
		Email:    "alice@example.com",
		Name:     "Alice",
	})
	if err != nil {
		t.Fatalf("LoginWithOAuth err = %v", err)
	}
	if token != "admin-oauth-token" {
		t.Fatalf("token = %q", token)
	}
	if admin == nil || admin.Username != "alice@example.com" {
		t.Fatalf("admin = %+v", admin)
	}
	if repo.bindings != 1 {
		t.Fatalf("bindings = %d, want 1", repo.bindings)
	}
}

func TestAdminAuthUsecase_LoginWithOAuth_UnknownEmailRejected(t *testing.T) {
	repo := newMemAdminAuthRepo()
	uc := NewAdminAuthUsecase(repo, func(int, string, int8) (string, time.Time, error) {
		return "", time.Time{}, nil
	}, log.NewStdLogger(io.Discard), tracesdk.NewTracerProvider())

	_, _, _, err := uc.LoginWithOAuth(context.Background(), OAuthIdentity{
		Provider: "google",
		Subject:  "sub-1",
		Email:    "missing@example.com",
	})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("err = %v, want ErrUserNotFound", err)
	}
}
