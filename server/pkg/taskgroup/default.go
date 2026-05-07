// 管理默认的后台任务组，确保进程退出时，任务可以被正确收口。
package taskgroup

import (
	"context"
	"errors"
	"time"
)

var errRepeatedInit = errors.New("errRepeatedInit")
var defaultGroup *Group = nil

const (
	// 程序退出时，给后台任务 30s 的优雅收口时间。
	defaultGroupStopTimeout = time.Second * 30
)

func Init() func() {
	if defaultGroup != nil {
		panic(errRepeatedInit)
	}
	defaultGroup = New()
	return func() {
		defaultGroup.Stop(true, defaultGroupStopTimeout)
		// 清理后重置默认实例，便于测试和进程内重复初始化。
		defaultGroup = nil
	}
}

// 新建后台任务，对默认任务组进行管理，程序退出前会有收口等待时间。
//
// ctx: 调用位置的上下文
// run(ctx): 回调中的 ctx 脱离了原上下文的 WithCancel，但是包含了 Group 的 WithCancel
func Go(ctx context.Context, run func(ctx context.Context), panicFunc ...func(ctx context.Context, err interface{})) {
	defaultGroup.Go(ctx, run, panicFunc...)
}
