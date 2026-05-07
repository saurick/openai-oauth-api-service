# taskgroup

`taskgroup` 是一个轻量的后台 goroutine 生命周期管理器，目标不是做通用线程池，而是把“启动、取消、等待、panic 收口”集中到一个点，方便服务退出和问题排查。

## 适用场景

- 需要在请求流程里顺手启动一个后台任务，但服务退出时要能统一收口。
- 某个模块自己维护少量周期性任务，希望有一个明确的停止入口。
- 需要在 panic 时统一记录日志，而不是让 goroutine 直接把进程打崩。

## 不适用场景

- 需要控制并发数的 worker pool。
- 需要任务队列、优先级、重试、结果汇总的调度系统。
- 希望“强制杀死” goroutine。Go 本身不支持这一点，这个包只能发取消信号，不能强杀执行中的逻辑。

## 生命周期语义

### `Go(ctx, run, panicFunc...)`

- 会保留原始 `ctx` 里的 `Value`。
- 会主动脱离原始 `ctx` 的取消链路。
- 也会一并脱离原始 `ctx` 的 deadline。
- 会为新任务重新包一层 `WithCancel`，供 `Stop(...)` 统一取消。

这意味着：

- 调用方传进来的 `ctx` 更像“携带上下文值的来源”，不是后台任务的直接生死开关。
- 真正的停止信号来自 `taskgroup` 自己注入的新 `ctx`。

### `Stop(wait, timeout)`

- `Stop(false, 0)`：立即发取消信号，不等待任务退出。
- `Stop(true, timeout)`：先等任务自行结束；超过 `timeout` 还没结束，就取消剩余任务。
- `timeout <= 0`：当前实现等价于立即取消，不做等待。
- 一旦 `Stop(...)` 调用过，再继续 `Go(...)` 会 panic `ErrStopped`。

## 正确用法

### 1. 任务内部必须配合 `ctx.Done()`

后台任务想要被及时停止，核心不是外面有没有 `Stop(...)`，而是任务里面有没有监听取消信号。

```go
group := taskgroup.New()
group.Go(ctx, func(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(10 * time.Second):
	}

	// do work
})
```

### 2. 延迟任务不要直接裸 `time.Sleep(...)`

下面这种写法在收到取消后不会立即退出：

```go
group := taskgroup.New()
group.Go(ctx, func(ctx context.Context) {
	time.Sleep(30 * time.Second)
	doWork()
})
```

更稳妥的写法是把等待也做成可取消：

```go
group := taskgroup.New()
group.Go(ctx, func(ctx context.Context) {
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}

	doWork()
})
```

### 3. 周期任务要显式 `Stop()` ticker

```go
group := taskgroup.New()
group.Go(ctx, func(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			doWork()
		}
	}
})
```

### 4. goroutine 里只用局部变量，不要回写外层 `err`

后台 goroutine 与外层流程共享 `err`、`resp` 这类变量，很容易引入数据竞争，也会让错误归因变混乱。优先在闭包内部重新定义局部变量，必要时通过 channel 回传结果。

## 默认实例

如果服务只需要一个全局后台任务管理器，可以用默认实例：

```go
cleanup := taskgroup.Init()
defer cleanup()

taskgroup.Go(ctx, func(ctx context.Context) {
	// background work
})
```

默认实例适合应用级的后台任务；模块内部更推荐用 `taskgroup.New()` 自己持有实例，边界更清楚。

## 三种常见方案对照

下面用同一个目标做对照：用户点击“预览 / 下载 PDF”，系统需要准备数据、生成文件，并在合适的层级做收口。

### 1. 请求内并发准备数据：用 `errgroup`

如果 PDF 预览或下载完全发生在一次 HTTP / JSON-RPC 请求里，最适合的是 `errgroup`。它的生命周期跟当前请求同生共死，请求结束就 `Wait()` 收口。

```go
func (s *ReportService) PreviewPDF(ctx context.Context, reportID int64) ([]byte, error) {
	g, ctx := errgroup.WithContext(ctx)

	var (
		report *Report
		assets []Asset
	)

	g.Go(func() error {
		var err error
		report, err = s.repo.GetReport(ctx, reportID)
		return err
	})
	g.Go(func() error {
		var err error
		assets, err = s.repo.ListAssets(ctx, reportID)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return s.renderer.RenderPDF(ctx, report, assets)
}
```

这个场景下通常不需要 `taskgroup`：

- 请求一结束，任务就应该一起结束。
- 失败要直接向调用方返回。
- 没有“未来另一个时刻再统一 Stop”的需求。

### 2. 进程里跑多个长期组件：用 `oklog/run.Group`

如果你要跑的是“PDF worker + HTTP server + signal shutdown”这类长期组件，适合 `run.Group`。它管理的是应用组件生命周期，不是某次具体 PDF 请求。

```go
func runApp() error {
	var g run.Group

	{
		ln, _ := net.Listen("tcp", ":8080")
		httpServer := &http.Server{Handler: newHTTPHandler()}
		g.Add(func() error {
			return httpServer.Serve(ln)
		}, func(err error) {
			_ = httpServer.Shutdown(context.Background())
		})
	}

	{
		ctx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			return runPDFWorker(ctx)
		}, func(err error) {
			cancel()
		})
	}

	return g.Run()
}
```

这个场景适合：

- 后台有固定数量的长期组件。
- 任一组件退出时，希望整体开始收口。
- 重点是“应用进程怎么优雅退出”，不是“某个对象内部怎么动态起很多任务”。

### 3. 某个对象里动态起后台任务：用 `taskgroup`

如果某个 `DocumentSession` / `Device` / `Job` 对象在生命周期里会陆续起延迟任务、轮询任务、异步清理任务，适合 `taskgroup`。它管理的是对象级后台任务，而不是请求级并发。

```go
type DocumentSession struct {
	group *taskgroup.Group
	cache *PreviewCache
}

func NewDocumentSession() *DocumentSession {
	return &DocumentSession{
		group: taskgroup.New(),
		cache: NewPreviewCache(),
	}
}

func (s *DocumentSession) WarmPreview(ctx context.Context, docID int64) {
	ctx = taskgroup.WithOperation(ctx, "pdf-preview")
	ctx = taskgroup.WithTaskName(ctx, "warm-preview-cache")

	s.group.Go(ctx, func(ctx context.Context) {
		timer := time.NewTimer(3 * time.Second)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		pdfBytes, err := s.cache.Render(ctx, docID)
		if err != nil {
			log.WithContext(ctx, log.GetLogger()).Log(log.LevelError, "msg", "warm preview failed", "doc_id", docID, "err", err)
			return
		}

		s.cache.Store(docID, pdfBytes)
	})
}

func (s *DocumentSession) Close() {
	s.group.Stop(true, 5*time.Second)
}
```

这个场景适合：

- 后台任务绑定到某个对象或会话。
- 任务是动态出现的，不是固定 2-3 个组件。
- 未来要在对象销毁时统一 `Stop(...)`。

### 一句话决策

- PDF 预览 / 下载如果是请求内同步生成：优先 `errgroup`，很多时候甚至串行就够了。
- PDF 处理如果是独立 worker 服务：优先 `run.Group`。
- PDF 处理如果绑定到某个会话对象，且会动态起后台预热 / 清理任务：才考虑 `taskgroup`。

## panic 处理

- 没有显式传 `panicFunc` 时，会走 `DefaultPanicFunc`。
- 默认行为会记录当前 goroutine 的栈，避免 panic 被吞掉后完全无痕。
- 无论正常返回还是 panic，包内部都会做运行态清理，避免 `Stop(true, timeout)` 误判任务还活着。

## 最小观测建议

如果这个包承接的是服务里的关键后台任务，建议在调用侧补最小观测，而不是把每个 goroutine 都做成复杂框架。

优先级最高的是日志：

- 任务启动时：记录 `operation`、关键业务键、`request_id` / `trace_id`。
- 任务退出时：记录耗时和退出原因。
- `Stop(true, timeout)` 触发超时时：明确记录 timeout 和剩余任务。
- panic 时：记录 error log，避免只剩一条模糊的 `recover`。

trace 可以做轻量增强：

- 在已有 span 上加 event 或 attribute。
- 不需要为了每个短后台任务都单独起一套很重的 trace 结构。

当前实现额外提供了两个可选 helper，便于调用侧把业务语义带进日志：

- `taskgroup.WithOperation(ctx, "sync-cache")`
- `taskgroup.WithTaskName(ctx, "refresh-home-feed")`

## 测试建议

这个包的测试重点应该围绕生命周期语义，而不是拼抢时间窗口：

- 用 channel 驱动断言，不要依赖很紧的 `Sleep/timeout` 比例。
- 覆盖 panic 后的清理路径，确保 `Stop(true, timeout)` 不会平白等满 timeout。
- 覆盖 `Stop(false)` 的立即取消语义。
- 覆盖 `Stop(true)` 在任务未退出时会阻塞，在任务退出后会继续返回。
- 覆盖 `Stop(...)` 后再次 `Go(...)` 会 panic。

如果后续要增强这个包，优先考虑：

- 调用侧的观测一致性。
- 更明确的任务命名。
- 更稳的并发/收口测试。

不要一开始就把它扩成复杂调度系统。
