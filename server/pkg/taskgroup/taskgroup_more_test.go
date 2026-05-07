package taskgroup

import (
	"context"
	"testing"
	"time"
)

func TestStopImmediateCancel(t *testing.T) {
	thread := New()
	started := make(chan struct{})
	canceled := make(chan struct{})
	exited := make(chan struct{})

	thread.Go(context.Background(), func(ctx context.Context) {
		close(started)
		<-ctx.Done()
		close(canceled)
		close(exited)
	})
	waitForSignal(t, started, testAsyncWait, "goroutine did not start")

	thread.Stop(false, 0)

	waitForSignal(t, canceled, testAsyncWait, "expected cancellation but did not receive it")
	waitForSignal(t, exited, testAsyncWait, "goroutine did not exit after cancellation")
}

func TestStopWaitTimeoutCancels(t *testing.T) {
	thread := New()
	started := make(chan struct{})
	canceled := make(chan struct{})

	thread.Go(context.Background(), func(ctx context.Context) {
		close(started)
		<-ctx.Done()
		close(canceled)
	})
	waitForSignal(t, started, testAsyncWait, "goroutine did not start")

	stopped := make(chan struct{})
	go func() {
		thread.Stop(true, 200*time.Millisecond)
		close(stopped)
	}()

	ensureNoSignal(t, stopped, 80*time.Millisecond, "Stop returned before timeout elapsed")
	waitForSignal(t, stopped, testAsyncWait, "Stop did not return after timeout")
	waitForSignal(t, canceled, testAsyncWait, "expected cancellation after timeout")
}

func TestStopAfterPanicReturnsPromptly(t *testing.T) {
	thread := New()
	panicNotified := make(chan struct{})

	thread.Go(context.Background(), func(ctx context.Context) {
		panic("boom")
	}, func(ctx context.Context, err interface{}) {
		close(panicNotified)
	})
	waitForSignal(t, panicNotified, testAsyncWait, "panic handler was not invoked")

	stopped := make(chan struct{})
	go func() {
		thread.Stop(true, time.Second)
		close(stopped)
	}()

	waitForSignal(t, stopped, 400*time.Millisecond, "Stop waited for timeout after panic")
}

func TestStopMultipleCalls(t *testing.T) {
	thread := New()
	started := make(chan struct{})
	exited := make(chan struct{})

	thread.Go(context.Background(), func(ctx context.Context) {
		close(started)
		<-ctx.Done()
		close(exited)
	})
	waitForSignal(t, started, testAsyncWait, "goroutine did not start")

	thread.Stop(false, 0)
	waitForSignal(t, exited, testAsyncWait, "goroutine did not exit after first Stop")

	stoppedAgain := make(chan struct{})
	go func() {
		thread.Stop(true, time.Second)
		close(stoppedAgain)
	}()

	waitForSignal(t, stoppedAgain, 400*time.Millisecond, "second Stop call did not return promptly")
}
