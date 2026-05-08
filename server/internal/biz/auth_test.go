// server/internal/biz/auth_test.go
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

type memAuthRepo struct {
	mu sync.Mutex

	usersByName map[string]*User
	lastLogin   map[int]time.Time
	nextUserID  int
}

func newMemAuthRepo() *memAuthRepo {
	return &memAuthRepo{
		usersByName: make(map[string]*User),
		lastLogin:   make(map[int]time.Time),
		nextUserID:  1,
	}
}

func (r *memAuthRepo) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	u := r.usersByName[username]
	if u == nil {
		return nil, errors.New("not found")
	}
	cp := *u
	return &cp, nil
}

func (r *memAuthRepo) GetUserByID(ctx context.Context, id int) (*User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, u := range r.usersByName {
		if u.ID == id {
			cp := *u
			return &cp, nil
		}
	}
	return nil, errors.New("not found")
}

func (r *memAuthRepo) CreateUser(ctx context.Context, u *User) (*User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.usersByName[u.Username]; ok {
		return nil, errors.New("duplicate username")
	}
	cp := *u
	cp.ID = r.nextUserID
	r.nextUserID++

	r.usersByName[cp.Username] = &cp
	return &cp, nil
}

func (r *memAuthRepo) UpdateUserLastLogin(ctx context.Context, id int, t time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastLogin[id] = t
	return nil
}

func TestAuthUsecase_Register_Success(t *testing.T) {
	repo := newMemAuthRepo()

	logger := log.NewStdLogger(io.Discard)
	tp := tracesdk.NewTracerProvider()
	genTok := func(userID int, username string, role int8) (string, time.Time, error) {
		return "tok-abc", time.Now().Add(7 * 24 * time.Hour), nil
	}
	uc := NewAuthUsecase(repo, genTok, logger, tp)

	token, exp, u, err := uc.Register(context.Background(), "alice", "p@ss")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if token != "tok-abc" {
		t.Fatalf("unexpected token: %s", token)
	}
	if u == nil || u.Username != "alice" || u.ID == 0 {
		t.Fatalf("unexpected user: %+v", u)
	}
	if exp.Before(time.Now()) {
		t.Fatalf("unexpected expireAt: %v", exp)
	}
}

func TestAuthUsecase_Register_UserExists(t *testing.T) {
	repo := newMemAuthRepo()

	hash, _ := bcrypt.GenerateFromPassword([]byte("x"), bcrypt.DefaultCost)
	_, _ = repo.CreateUser(context.Background(), &User{
		Username:     "alice",
		PasswordHash: string(hash),
	})

	logger := log.NewStdLogger(io.Discard)
	tp := tracesdk.NewTracerProvider()
	uc := NewAuthUsecase(repo, func(int, string, int8) (string, time.Time, error) {
		return "", time.Time{}, nil
	}, logger, tp)

	_, _, _, err := uc.Register(context.Background(), "alice", "p@ss")
	if !errors.Is(err, ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}
}

func TestAuthUsecase_Register_EmptyArgs(t *testing.T) {
	repo := newMemAuthRepo()

	logger := log.NewStdLogger(io.Discard)
	tp := tracesdk.NewTracerProvider()
	uc := NewAuthUsecase(repo, func(int, string, int8) (string, time.Time, error) {
		return "", time.Time{}, nil
	}, logger, tp)

	_, _, _, err := uc.Register(context.Background(), "", "p@ss")
	if err == nil {
		t.Fatalf("expected error for empty username")
	}

	_, _, _, err = uc.Register(context.Background(), "alice", "")
	if err == nil {
		t.Fatalf("expected error for empty password")
	}
}

func TestAuthUsecase_Register_TokenGenFailed(t *testing.T) {
	repo := newMemAuthRepo()

	logger := log.NewStdLogger(io.Discard)
	tp := tracesdk.NewTracerProvider()
	uc := NewAuthUsecase(repo, func(int, string, int8) (string, time.Time, error) {
		return "", time.Time{}, errors.New("token gen failed")
	}, logger, tp)

	_, _, _, err := uc.Register(context.Background(), "alice", "p@ss")
	if err == nil {
		t.Fatalf("expected error when token generation fails")
	}
}

func TestAuthUsecase_Login_Success(t *testing.T) {
	repo := newMemAuthRepo()

	hash, _ := bcrypt.GenerateFromPassword([]byte("p@ss"), bcrypt.DefaultCost)
	_, _ = repo.CreateUser(context.Background(), &User{
		Username:     "alice",
		PasswordHash: string(hash),
		Disabled:     false,
	})

	logger := log.NewStdLogger(io.Discard)
	tp := tracesdk.NewTracerProvider()
	uc := NewAuthUsecase(repo, func(userID int, username string, role int8) (string, time.Time, error) {
		return "tok-login", time.Now().Add(time.Hour), nil
	}, logger, tp)

	token, exp, u, err := uc.Login(context.Background(), "alice", "p@ss")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if token != "tok-login" {
		t.Fatalf("unexpected token: %s", token)
	}
	if u == nil || u.Username != "alice" {
		t.Fatalf("unexpected user: %+v", u)
	}
	if exp.Before(time.Now()) {
		t.Fatalf("unexpected expireAt: %v", exp)
	}
}

func TestAuthUsecase_Login_UserNotFound(t *testing.T) {
	repo := newMemAuthRepo()

	logger := log.NewStdLogger(io.Discard)
	tp := tracesdk.NewTracerProvider()
	uc := NewAuthUsecase(repo, func(int, string, int8) (string, time.Time, error) {
		return "", time.Time{}, nil
	}, logger, tp)

	_, _, _, err := uc.Login(context.Background(), "noone", "x")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestAuthUsecase_Login_UserDisabled(t *testing.T) {
	repo := newMemAuthRepo()

	hash, _ := bcrypt.GenerateFromPassword([]byte("p@ss"), bcrypt.DefaultCost)
	_, _ = repo.CreateUser(context.Background(), &User{
		Username:     "alice",
		PasswordHash: string(hash),
		Disabled:     true,
	})

	logger := log.NewStdLogger(io.Discard)
	tp := tracesdk.NewTracerProvider()
	uc := NewAuthUsecase(repo, func(int, string, int8) (string, time.Time, error) {
		return "", time.Time{}, nil
	}, logger, tp)

	_, _, _, err := uc.Login(context.Background(), "alice", "p@ss")
	if !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("expected ErrUserDisabled, got %v", err)
	}
}

func TestAuthUsecase_Login_InvalidPassword(t *testing.T) {
	repo := newMemAuthRepo()

	hash, _ := bcrypt.GenerateFromPassword([]byte("p@ss"), bcrypt.DefaultCost)
	_, _ = repo.CreateUser(context.Background(), &User{
		Username:     "alice",
		PasswordHash: string(hash),
		Disabled:     false,
	})

	logger := log.NewStdLogger(io.Discard)
	tp := tracesdk.NewTracerProvider()
	uc := NewAuthUsecase(repo, func(int, string, int8) (string, time.Time, error) {
		return "", time.Time{}, nil
	}, logger, tp)

	_, _, _, err := uc.Login(context.Background(), "alice", "wrong")
	if !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("expected ErrInvalidPassword, got %v", err)
	}
}

func TestAuthUsecase_Login_EmptyArgs(t *testing.T) {
	repo := newMemAuthRepo()

	logger := log.NewStdLogger(io.Discard)
	tp := tracesdk.NewTracerProvider()
	uc := NewAuthUsecase(repo, func(int, string, int8) (string, time.Time, error) {
		return "", time.Time{}, nil
	}, logger, tp)

	_, _, _, err := uc.Login(context.Background(), "", "p@ss")
	if err == nil {
		t.Fatalf("expected error for empty username")
	}

	_, _, _, err = uc.Login(context.Background(), "alice", "")
	if err == nil {
		t.Fatalf("expected error for empty password")
	}
}

func TestAuthUsecase_Login_TokenGenFailed(t *testing.T) {
	repo := newMemAuthRepo()

	hash, _ := bcrypt.GenerateFromPassword([]byte("p@ss"), bcrypt.DefaultCost)
	_, _ = repo.CreateUser(context.Background(), &User{
		Username:     "alice",
		PasswordHash: string(hash),
		Disabled:     false,
	})

	logger := log.NewStdLogger(io.Discard)
	tp := tracesdk.NewTracerProvider()
	uc := NewAuthUsecase(repo, func(int, string, int8) (string, time.Time, error) {
		return "", time.Time{}, errors.New("token gen failed")
	}, logger, tp)

	_, _, _, err := uc.Login(context.Background(), "alice", "p@ss")
	if err == nil {
		t.Fatalf("expected error when token generation fails")
	}
}
