package taskgroup

import (
	"context"
	"fmt"
	"testing"
	"time"
)

const (
	testAsyncWait    = time.Second
	testNoSignalWait = 120 * time.Millisecond
)

func waitForSignal(t *testing.T, ch <-chan struct{}, timeout time.Duration, msg string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(timeout):
		t.Fatalf("%s", msg)
	}
}

func ensureNoSignal(t *testing.T, ch <-chan struct{}, timeout time.Duration, msg string) {
	t.Helper()

	select {
	case <-ch:
		t.Fatalf("%s", msg)
	case <-time.After(timeout):
	}
}

func TestDefaultGroup(t *testing.T) {
	cleanupGroup := Init()

	done := make(chan struct{})
	Go(context.Background(), func(ctx context.Context) {
		close(done)
	})
	waitForSignal(t, done, testAsyncWait, "default taskgroup did not run task")

	cleanupGroup()

	// cleanup 后允许再次初始化，避免测试或进程内重建时卡在重复初始化。
	cleanupGroup = Init()
	cleanupGroup()
}

func TestGoPreservesContextValueAndIgnoresParentCancel(t *testing.T) {
	type testKey string

	parent := context.WithValue(context.Background(), testKey("key"), "test_value")
	ctxOld, cancel := context.WithCancel(parent)
	cancel()

	result := make(chan error, 1)
	thread := New()
	thread.Go(ctxOld, func(ctx context.Context) {
		if ctxOld == ctx {
			result <- fmt.Errorf("expected a detached context instance")
			return
		}
		if ctx.Err() != nil {
			result <- fmt.Errorf("expected detached context to stay active, got %v", ctx.Err())
			return
		}
		v, ok := ctx.Value(testKey("key")).(string)
		if !ok || v != "test_value" {
			result <- fmt.Errorf("expected context value to be preserved, got %v", ctx.Value(testKey("key")))
			return
		}
		result <- nil
	})

	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(testAsyncWait):
		t.Fatal("goroutine did not finish context validation")
	}

	thread.Stop(true, testAsyncWait)
}

func TestStopWaitBlocksUntilRunningTaskFinishes(t *testing.T) {
	thread := New()
	started := make(chan struct{})
	release := make(chan struct{})

	thread.Go(context.Background(), func(ctx context.Context) {
		close(started)
		<-release
	})
	waitForSignal(t, started, testAsyncWait, "goroutine did not start")

	stopped := make(chan struct{})
	go func() {
		thread.Stop(true, testAsyncWait)
		close(stopped)
	}()

	ensureNoSignal(t, stopped, testNoSignalWait, "Stop returned before running task finished")

	close(release)
	waitForSignal(t, stopped, testAsyncWait, "Stop did not return after task finished")
}

func TestGoAfterStopPanics(t *testing.T) {
	thread := New()
	thread.Stop(true, 0)

	defer func() {
		r := recover()
		if r != ErrStopped {
			t.Fatalf("expected panic ErrStopped, got: %v", r)
		}
	}()

	thread.Go(context.Background(), func(ctx context.Context) {})
}
