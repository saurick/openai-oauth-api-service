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
)

type memAdminAuthRepo struct {
	mu     sync.Mutex
	admins map[string]*AdminUser
}

func newMemAdminAuthRepo() *memAdminAuthRepo {
	return &memAdminAuthRepo{admins: map[string]*AdminUser{}}
}

func (r *memAdminAuthRepo) put(admin *AdminUser) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.admins[admin.Username] = admin
}

func (r *memAdminAuthRepo) GetAdminByUsername(ctx context.Context, username string) (*AdminUser, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	admin := r.admins[username]
	if admin == nil {
		return nil, errors.New("not found")
	}
	cp := *admin
	return &cp, nil
}

func (r *memAdminAuthRepo) GetAdminByOAuthIdentity(ctx context.Context, provider, subject string) (*AdminUser, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, admin := range r.admins {
		if admin.OAuthProvider != nil && admin.OAuthSubject != nil && *admin.OAuthProvider == provider && *admin.OAuthSubject == subject {
			cp := *admin
			return &cp, nil
		}
	}
	return nil, errors.New("not found")
}

func (r *memAdminAuthRepo) BindAdminOAuthIdentity(ctx context.Context, id int, identity OAuthIdentity) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, admin := range r.admins {
		if admin.ID == id {
			admin.OAuthProvider = optionalStringPtr(identity.Provider)
			admin.OAuthSubject = optionalStringPtr(identity.Subject)
			admin.OAuthEmail = optionalStringPtr(identity.Email)
			admin.OAuthName = optionalStringPtr(identity.Name)
			return nil
		}
	}
	return errors.New("not found")
}

func (r *memAdminAuthRepo) UpdateAdminLastLogin(ctx context.Context, id int, t time.Time) error {
	return nil
}

func TestAdminAuthUsecase_LoginWithOAuth_BindsExistingAdmin(t *testing.T) {
	repo := newMemAdminAuthRepo()
	repo.put(&AdminUser{ID: 1, Username: "admin@example.com"})
	uc := NewAdminAuthUsecase(repo, func(userID int, username string, role int8) (string, time.Time, error) {
		return "admin-oauth-token", time.Now().Add(time.Hour), nil
	}, log.NewStdLogger(io.Discard), tracesdk.NewTracerProvider())

	token, _, admin, err := uc.LoginWithOAuth(context.Background(), OAuthIdentity{
		Provider:          "oidc",
		Subject:           "subject-1",
		Email:             "admin@example.com",
		PreferredUsername: "admin",
	})
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if token != "admin-oauth-token" {
		t.Fatalf("unexpected token: %s", token)
	}
	if admin == nil || admin.Username != "admin@example.com" || admin.OAuthSubject == nil || *admin.OAuthSubject != "subject-1" {
		t.Fatalf("unexpected admin: %+v", admin)
	}
}

func TestAdminAuthUsecase_LoginWithOAuth_RejectsUnknownAdmin(t *testing.T) {
	uc := NewAdminAuthUsecase(newMemAdminAuthRepo(), func(userID int, username string, role int8) (string, time.Time, error) {
		return "", time.Time{}, nil
	}, log.NewStdLogger(io.Discard), tracesdk.NewTracerProvider())

	_, _, _, err := uc.LoginWithOAuth(context.Background(), OAuthIdentity{
		Provider: "oidc",
		Subject:  "subject-1",
		Email:    "unknown@example.com",
	})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
