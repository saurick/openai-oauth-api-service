package data

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type seqPinger struct {
	mu         sync.Mutex
	sequence   []error
	defaultErr error
	calls      int
}

func (p *seqPinger) PingContext(_ context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.calls++
	idx := p.calls - 1
	if idx < len(p.sequence) {
		return p.sequence[idx]
	}
	return p.defaultErr
}

func (p *seqPinger) CallCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

func TestWaitForPostgresReady_SucceedsAfterRetries(t *testing.T) {
	p := &seqPinger{
		sequence: []error{
			errors.New("connection refused"),
			errors.New("connection refused"),
			nil,
		},
		defaultErr: nil,
	}
	logger := log.NewHelper(log.NewStdLogger(io.Discard))

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := waitForPostgresReady(ctx, p, 10*time.Millisecond, logger)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got := p.CallCount(); got != 3 {
		t.Fatalf("expected 3 calls, got %d", got)
	}
}

func TestWaitForPostgresReady_Timeout(t *testing.T) {
	p := &seqPinger{
		defaultErr: errors.New("connection refused"),
	}
	logger := log.NewHelper(log.NewStdLogger(io.Discard))

	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Millisecond)
	defer cancel()

	err := waitForPostgresReady(ctx, p, 10*time.Millisecond, logger)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "postgres not ready before timeout") {
		t.Fatalf("expected timeout prefix, got %v", err)
	}
	if got := p.CallCount(); got < 2 {
		t.Fatalf("expected at least 2 ping attempts, got %d", got)
	}
}
