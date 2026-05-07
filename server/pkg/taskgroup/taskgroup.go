// 管理临时异步 goroutine，确保进程退出时，后台任务可以被正确收口。
package taskgroup

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var ErrStopped = errors.New("ErrStopped")

type contextKey string

const (
	componentName contextKey = "taskgroup"

	operationContextKey contextKey = "taskgroup.operation"
	taskNameContextKey  contextKey = "taskgroup.task_name"
)

const (
	taskResultCompleted = "completed"
	taskResultCanceled  = "canceled"
	taskResultPanic     = "panic"
)

type Group struct {
	lock    *sync.Mutex
	wait    *sync.WaitGroup
	stoped  bool // 是否已停止
	running map[context.Context]context.CancelFunc
}

func New() *Group {
	return &Group{
		lock:    &sync.Mutex{},
		wait:    &sync.WaitGroup{},
		running: make(map[context.Context]context.CancelFunc),
	}
}

func WithOperation(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, operationContextKey, operation)
}

func WithTaskName(ctx context.Context, taskName string) context.Context {
	return context.WithValue(ctx, taskNameContextKey, taskName)
}

func DefaultPanicFunc(ctx context.Context, err interface{}) {
	buf := make([]byte, 10240)
	runtime.Stack(buf, false)
	logEvent(ctx, log.LevelError, "taskgroup task panic recovered",
		"result", taskResultPanic,
		"panic_value", fmt.Sprintf("%v", err),
		"stack", string(buf),
	)
}

// 新建后台任务，对 goroutine 进行管理，程序退出前可设置退出等待时间。
//
// ctx: 调用位置的上下文
// run(ctx): 回调中的 ctx 脱离了原上下文的 WithCancel，但是包含了 Group 的 WithCancel
// panicFunc(err): painc 回调
func (g *Group) Go(ctx context.Context, run func(ctx context.Context), panicFunc ...func(ctx context.Context, err interface{})) {
	startedAt := time.Now()

	g.lock.Lock()
	if g.stoped {
		runningCount := len(g.running)
		g.lock.Unlock()
		// 关键观测：拒绝新任务时直接打点，方便排查“为什么后台任务没启动”。
		logEvent(ctx, log.LevelWarn, "taskgroup rejected task after stop",
			"result", "rejected",
			"running_count", runningCount,
		)
		addTraceEvent(ctx, "taskgroup.go.rejected", attribute.Int("running_count", runningCount))
		panic(ErrStopped)
	}
	ctxWithValue := context.WithoutCancel(ctx)         // 取消原上线文的 cancel，只保留 value
	ctxNew, cancel := context.WithCancel(ctxWithValue) // 重新套用 cancel，使其可在 Group 中可控
	// 任务一旦被接受，就要和 running 登记放在同一临界区里，避免 Stop 漏等这次执行。
	g.wait.Add(1)
	g.running[ctxNew] = cancel
	runningCount := len(g.running)
	g.lock.Unlock()

	logEvent(ctxNew, log.LevelDebug, "taskgroup accepted task",
		"running_count", runningCount,
	)
	addTraceEvent(ctxNew, "taskgroup.go.accepted", attribute.Int("running_count", runningCount))

	go func() {
		result := taskResultCompleted

		// 无论正常返回还是 panic，都必须释放运行态，否则 Stop 会误判仍有存活任务。
		defer g.finish(ctxNew, startedAt, &result)

		defer func() {
			if err := recover(); err != nil {
				result = taskResultPanic
				addTraceEvent(ctxNew, "taskgroup.task.panic")
				pf := []func(context.Context, interface{}){DefaultPanicFunc}
				if len(panicFunc) > 0 {
					pf = panicFunc
				}
				for _, f := range pf {
					f(ctxNew, err)
				}
			}
		}()

		// 在执行任务前检查是否已被取消，避免执行已取消的任务
		// if ctxNew.Err() != nil {
		// 	g.lock.Lock()
		// 	delete(g.running, ctxNew)
		// 	g.lock.Unlock()
		// 	g.wait.Done()
		// 	return
		// }

		run(ctxNew)
	}()
}

func (g *Group) finish(ctx context.Context, startedAt time.Time, result *string) {
	g.lock.Lock()
	delete(g.running, ctx)
	runningCount := len(g.running)
	g.lock.Unlock()

	if *result == taskResultCompleted && ctx.Err() != nil {
		*result = taskResultCanceled
	}

	logEvent(ctx, log.LevelDebug, "taskgroup task finished",
		"result", *result,
		"duration_ms", time.Since(startedAt).Milliseconds(),
		"running_count", runningCount,
	)
	addTraceEvent(ctx, "taskgroup.task.finished",
		attribute.String("result", *result),
		attribute.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		attribute.Int("running_count", runningCount),
	)

	g.wait.Done()
}

func (g *Group) cancelRunning() int {
	g.lock.Lock()
	defer g.lock.Unlock()

	runningCount := len(g.running)
	for _, cancel := range g.running {
		cancel()
	}
	return runningCount
}

func (g *Group) runningCount() int {
	g.lock.Lock()
	defer g.lock.Unlock()
	return len(g.running)
}

// 程序退出前，停止所有任务。
// wait: 等待所有任务执行完毕
// timeout: 等待超时，取消未执行完毕的任务
func (g *Group) Stop(wait bool, timeout time.Duration) {
	g.lock.Lock()
	g.stoped = true
	runningCount := len(g.running)
	g.lock.Unlock()

	logEvent(context.Background(), log.LevelInfo, "taskgroup stop begin",
		"wait", wait,
		"timeout_ms", timeout.Milliseconds(),
		"running_count", runningCount,
	)

	if !wait {
		// 如果不等待，直接取消所有任务。
		canceledCount := g.cancelRunning()
		logEvent(context.Background(), log.LevelInfo, "taskgroup stop canceled running tasks without wait",
			"wait", wait,
			"canceled_count", canceledCount,
		)
		return
	}

	if timeout <= 0 {
		// 关键边界：非正超时按立即取消处理，避免调用方误以为会继续等待。
		canceledCount := g.cancelRunning()
		logEvent(context.Background(), log.LevelInfo, "taskgroup stop used immediate cancel due to non-positive timeout",
			"wait", wait,
			"timeout_ms", timeout.Milliseconds(),
			"canceled_count", canceledCount,
		)
		return
	}

	cancelChan := make(chan struct{})
	go func() {
		g.wait.Wait()
		close(cancelChan)
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-cancelChan: // 等待所有任务完成
		logEvent(context.Background(), log.LevelInfo, "taskgroup stop finished before timeout",
			"wait", wait,
			"timeout_ms", timeout.Milliseconds(),
			"running_count", g.runningCount(),
		)
	case <-timer.C: // 超时后进入取消分支
		// 关键观测：超时是后台任务生命周期排查里最重要的异常信号之一。
		logEvent(context.Background(), log.LevelWarn, "taskgroup stop timed out",
			"result", "timeout",
			"wait", wait,
			"timeout_ms", timeout.Milliseconds(),
			"running_count", g.runningCount(),
		)
	}

	// 取消所有剩余任务
	canceledCount := g.cancelRunning()
	logEvent(context.Background(), log.LevelInfo, "taskgroup dispatched cancellation to remaining tasks",
		"wait", wait,
		"canceled_count", canceledCount,
	)
}

func operationFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	val, _ := ctx.Value(operationContextKey).(string)
	return val
}

func taskNameFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	val, _ := ctx.Value(taskNameContextKey).(string)
	return val
}

func traceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	spanCtx := oteltrace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return ""
	}

	return spanCtx.TraceID().String()
}

func addTraceEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	if ctx == nil {
		return
	}

	oteltrace.SpanFromContext(ctx).AddEvent(name, oteltrace.WithAttributes(attrs...))
}

func logEvent(ctx context.Context, level log.Level, msg string, keyvals ...interface{}) {
	if ctx == nil {
		ctx = context.Background()
	}

	fields := []interface{}{
		"component", string(componentName),
		"msg", msg,
	}

	if operation := operationFromContext(ctx); operation != "" {
		fields = append(fields, "operation", operation)
	}
	if taskName := taskNameFromContext(ctx); taskName != "" {
		fields = append(fields, "task_name", taskName)
	}
	if traceID := traceIDFromContext(ctx); traceID != "" {
		fields = append(fields, "trace_id", traceID)
	}

	fields = append(fields, keyvals...)
	_ = log.WithContext(ctx, log.GetLogger()).Log(level, fields...)
}
