package logger

import (
	"context"
	"os"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type TaskIDKey struct{}
type RequestIDKey struct{}

func NewDefaultLogger(id, name, version string, debug bool) log.Logger {
	return log.With(NewStdColorLogger(os.Stdout, true, debug),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", name,
		"service.version", version,
		"request_id", RequestID(),
		"trace.id", tracing.TraceID(),
		"trace_sampled", TraceSampled(),
		"trace_link_id", TraceLinkID(),
		"span.id", tracing.SpanID(),
		"task.id", TaskID(),
	)
}

// 用于测试的logger
func NewDefaultLoggerForTest() log.Logger {
	return NewDefaultLogger("test-id", "test", "test-version", true)
}

// 自动输出任务ID
func TaskID() log.Valuer {
	return func(ctx context.Context) interface{} {
		v, ok := ctx.Value(TaskIDKey{}).(int64)
		if !ok {
			return ""
		}
		return v
	}
}

func WithTaskID(ctx context.Context, taskId int64) context.Context {
	return context.WithValue(ctx, TaskIDKey{}, taskId)
}

func RequestID() log.Valuer {
	return func(ctx context.Context) interface{} {
		v, ok := ctx.Value(RequestIDKey{}).(string)
		if !ok {
			return ""
		}
		return v
	}
}

func RequestIDFromContext(ctx context.Context) string {
	v, ok := ctx.Value(RequestIDKey{}).(string)
	if !ok {
		return ""
	}
	return v
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey{}, requestID)
}

func TraceSampled() log.Valuer {
	return func(ctx context.Context) interface{} {
		if ctx == nil {
			return false
		}
		return oteltrace.SpanContextFromContext(ctx).IsSampled()
	}
}

func TraceLinkID() log.Valuer {
	return func(ctx context.Context) interface{} {
		if ctx == nil {
			return ""
		}

		spanCtx := oteltrace.SpanContextFromContext(ctx)
		// 只给已采样链路暴露跳转用 trace id，避免 Loki 在低采样场景里生成点进去就是 404 的假链接。
		if !spanCtx.IsValid() || !spanCtx.IsSampled() {
			return ""
		}
		return spanCtx.TraceID().String()
	}
}
