// server/cmd/server/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"server/internal/conf"
	"server/pkg/logger"
	"server/pkg/taskgroup"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	_ "go.uber.org/automaxprocs"
	"go.uber.org/automaxprocs/maxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// 使用项目专属服务标识，避免与其他项目的日志与链路追踪混淆。
	Name      string = "openai-oauth-api-service-server"
	TraceName string = "openai-oauth-api-service-server.service"
	Version   string

	flagconf string

	id, _ = os.Hostname()
)

func init() {
	// 自动设置 GOMAXPROCS，关闭它自带的日志
	_, _ = maxprocs.Set(maxprocs.Logger(nil))

	// 默认给空，真正用的时候再自动探测
	flag.StringVar(&flagconf, "conf", "", "config path, eg: -conf ./server/configs/dev or -conf ./server/configs/prod")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
		),
	)
}

// 既兼容 kratos run（仓库根）又兼容 cd server/go run
func resolveConfPath(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}

	// 返回的是 config.yaml，而不是目录
	candidates := []string{
		"./configs/dev/config.yaml",
		"./server/configs/dev/config.yaml",
		"../configs/dev/config.yaml",
		"../../configs/dev/config.yaml",
	}

	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}

	return "./configs/dev/config.yaml"
}

func resolveLocalConfPath(confPath string) string {
	ext := filepath.Ext(confPath)
	if ext == "" {
		return ""
	}
	if strings.HasSuffix(confPath, ".local"+ext) {
		return ""
	}
	localPath := strings.TrimSuffix(confPath, ext) + ".local" + ext
	if fi, err := os.Stat(localPath); err == nil && !fi.IsDir() {
		return localPath
	}
	return ""
}

func overrideFromEnv(dataCfg *conf.Data, baseLogger log.Logger) {
	helper := log.NewHelper(baseLogger)

	if dataCfg.Postgres == nil {
		dataCfg.Postgres = &conf.Data_Postgres{}
	}
	if v := strings.TrimSpace(os.Getenv("POSTGRES_DSN")); v != "" {
		// 关键兜底：只记录覆盖来源，不输出 DSN 明文，避免数据库密码进入日志。
		dataCfg.Postgres.Dsn = v
		helper.Info("postgres dsn overridden from env")
	}

	if dataCfg.Auth == nil {
		dataCfg.Auth = &conf.Data_Auth{}
	}
	if v := strings.TrimSpace(os.Getenv("OAUTH_API_JWT_SECRET")); v != "" {
		// JWT 密钥走运行时 Secret 注入，避免生产密钥进入仓库。
		dataCfg.Auth.JwtSecret = v
		helper.Info("jwt secret overridden from env")
	}

	if dataCfg.Auth.Admin == nil {
		dataCfg.Auth.Admin = &conf.Data_Auth_Admin{}
	}
	if v := strings.TrimSpace(os.Getenv("OAUTH_API_ADMIN_USERNAME")); v != "" {
		dataCfg.Auth.Admin.Username = v
		helper.Info("admin username overridden from env")
	}
	if v := strings.TrimSpace(os.Getenv("OAUTH_API_ADMIN_PASSWORD")); v != "" {
		dataCfg.Auth.Admin.Password = v
		helper.Info("admin password overridden from env")
	}

	if dataCfg.Auth.Oauth == nil {
		dataCfg.Auth.Oauth = &conf.Data_Auth_OAuth{}
	}
	if v := strings.TrimSpace(os.Getenv("OAUTH_API_OAUTH_ENABLED")); v != "" {
		dataCfg.Auth.Oauth.Enabled = strings.EqualFold(v, "true") || v == "1" || strings.EqualFold(v, "yes")
		helper.Info("oauth enabled overridden from env")
	}
	if v := strings.TrimSpace(os.Getenv("OAUTH_API_OAUTH_PROVIDER_NAME")); v != "" {
		dataCfg.Auth.Oauth.ProviderName = v
		helper.Info("oauth provider name overridden from env")
	}
	if v := strings.TrimSpace(os.Getenv("OAUTH_API_OAUTH_CLIENT_ID")); v != "" {
		dataCfg.Auth.Oauth.ClientId = v
		helper.Info("oauth client id overridden from env")
	}
	if v := strings.TrimSpace(os.Getenv("OAUTH_API_OAUTH_CLIENT_SECRET")); v != "" {
		dataCfg.Auth.Oauth.ClientSecret = v
		helper.Info("oauth client secret overridden from env")
	}
	if v := strings.TrimSpace(os.Getenv("OAUTH_API_OAUTH_AUTH_URL")); v != "" {
		dataCfg.Auth.Oauth.AuthUrl = v
		helper.Info("oauth auth url overridden from env")
	}
	if v := strings.TrimSpace(os.Getenv("OAUTH_API_OAUTH_TOKEN_URL")); v != "" {
		dataCfg.Auth.Oauth.TokenUrl = v
		helper.Info("oauth token url overridden from env")
	}
	if v := strings.TrimSpace(os.Getenv("OAUTH_API_OAUTH_USERINFO_URL")); v != "" {
		dataCfg.Auth.Oauth.UserInfoUrl = v
		helper.Info("oauth userinfo url overridden from env")
	}
	if v := strings.TrimSpace(os.Getenv("OAUTH_API_OAUTH_REDIRECT_URL")); v != "" {
		dataCfg.Auth.Oauth.RedirectUrl = v
		helper.Info("oauth redirect url overridden from env")
	}
	if v := strings.TrimSpace(os.Getenv("OAUTH_API_OAUTH_SCOPES")); v != "" {
		dataCfg.Auth.Oauth.Scopes = splitCSV(v)
		helper.Info("oauth scopes overridden from env")
	}

	if dataCfg.Openai == nil {
		dataCfg.Openai = &conf.Data_OpenAI{}
	}
	if v := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); v != "" {
		dataCfg.Openai.ApiKey = v
		helper.Info("openai api key overridden from env")
	}
	if v := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")); v != "" {
		dataCfg.Openai.BaseUrl = v
		helper.Info("openai base url overridden from env")
	}
	if v := strings.TrimSpace(os.Getenv("UPSTREAM_PROXY_URL")); v != "" {
		dataCfg.Openai.UpstreamProxyUrl = v
		helper.Info("openai upstream proxy overridden from env")
	}

	if dataCfg.Api == nil {
		dataCfg.Api = &conf.Data_Api{}
	}
	if v := strings.TrimSpace(os.Getenv("OAUTH_API_RATE_LIMIT_ENABLED")); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			dataCfg.Api.RateLimitEnabled = parsed
			helper.Info("api rate limit switch overridden from env")
		}
	}
	if v := strings.TrimSpace(os.Getenv("OAUTH_API_EXPORT_MAX_DAYS")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			dataCfg.Api.ExportMaxDays = int32(parsed)
			helper.Info("api export max days overridden from env")
		}
	}
	if v := strings.TrimSpace(os.Getenv("OAUTH_API_MODEL_SYNC_TIMEOUT_SECONDS")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			dataCfg.Api.ModelSyncTimeoutSeconds = int32(parsed)
			helper.Info("api model sync timeout overridden from env")
		}
	}
	if v := strings.TrimSpace(os.Getenv("OAUTH_API_ALERT_RETENTION_DAYS")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			dataCfg.Api.AlertRetentionDays = int32(parsed)
			helper.Info("api alert retention days overridden from env")
		}
	}
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func buildConfigSources(confPath string) []config.Source {
	paths := []string{confPath}
	// 本地联调兜底：若存在未跟踪的 config.local.yaml，则覆盖公共 dev 配置，避免把私有 DSN 提交进仓库。
	if localPath := resolveLocalConfPath(confPath); localPath != "" {
		paths = append(paths, localPath)
	}
	sources := make([]config.Source, 0, len(paths))
	for _, path := range paths {
		sources = append(sources, file.NewSource(path))
	}
	return sources
}

// 采样率统一夹到 [0,1]，避免配置脏值把 tracing 行为搞成不可预期。
func normalizeTraceRatio(ratio float64) float64 {
	switch {
	case math.IsNaN(ratio), math.IsInf(ratio, 0):
		return 0
	case ratio <= 0:
		return 0
	case ratio >= 1:
		return 1
	default:
		return ratio
	}
}

// 统一走 ParentBased，既保留上游采样决策，也允许当前服务按比例控制根 span。
func buildTraceSampler(ratio float64) tracesdk.Sampler {
	return tracesdk.ParentBased(tracesdk.TraceIDRatioBased(normalizeTraceRatio(ratio)))
}

// 初始化 TracerProvider：优先远端 OTLP（异步 Batch），失败或未配置就本地 provider。
func initTracerProvider(traceName, traceEndpoint string, traceRatio float64, baseLogger log.Logger) *tracesdk.TracerProvider {
	helper := log.NewHelper(baseLogger)
	var tp *tracesdk.TracerProvider
	normalizedRatio := normalizeTraceRatio(traceRatio)
	resourceOption := tracesdk.WithResource(resource.NewSchemaless(
		semconv.ServiceNameKey.String(traceName),
	))
	samplerOption := tracesdk.WithSampler(buildTraceSampler(normalizedRatio))

	// 1) 有 endpoint → 尝试 OTLP HTTP exporter
	if traceEndpoint != "" {
		exp, err := otlptracehttp.New(
			context.Background(),
			otlptracehttp.WithEndpoint(traceEndpoint),
			otlptracehttp.WithInsecure(),
		)
		if err != nil {
			helper.Warnw(
				"msg", "init otlp exporter failed, fallback to local tracer",
				"trace_endpoint", traceEndpoint,
				"trace_ratio", normalizedRatio,
				"error", err,
			)
		} else {
			tp = tracesdk.NewTracerProvider(
				samplerOption,
				tracesdk.WithBatcher(exp), // ✅ 异步批量导出，不阻塞请求
				resourceOption,
			)
			helper.Infow(
				"msg", "tracer provider initialized",
				"mode", "otlp-http",
				"trace_endpoint", traceEndpoint,
				"trace_ratio", normalizedRatio,
			)
		}
	}

	// 2) 没配 endpoint 或 exporter 初始化失败 → 本地 provider，仅保留进程内 span 语义。
	if tp == nil {
		tp = tracesdk.NewTracerProvider(
			samplerOption,
			resourceOption,
		)
		helper.Infow(
			"msg", "tracer provider initialized without remote exporter",
			"mode", "local",
			"trace_ratio", normalizedRatio,
		)
	}

	otel.SetTracerProvider(tp) // 设置全局tp
	return tp
}

func main() {
	flag.Parse()

	confPath := resolveConfPath(flagconf)
	fmt.Println("using conf path:", confPath)

	// ===== 1. 加载配置文件 =====
	sources := buildConfigSources(confPath)
	c := config.New(
		config.WithSource(
			sources...,
		),
	)
	defer func() {
		_ = c.Close()
	}()

	if err := c.Load(); err != nil {
		panic(fmt.Errorf("load config failed: %w (conf=%s)", err, confPath))
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(fmt.Errorf("scan bootstrap config failed: %w", err))
	}

	// ===== 2. 安全地读取 Log 配置 =====
	debug := false
	if bc.Log != nil {
		debug = bc.Log.Debug
	}

	logger := logger.NewDefaultLogger(id, Name, Version, debug)
	log.SetLogger(logger) // 设置全局日志

	// ===== 3. 安全地读取 Trace 配置（避免 nil pointer） =====
	traceName := TraceName
	traceEndpoint := ""
	traceRatio := 0.0

	if bc.Trace != nil && bc.Trace.Jaeger != nil {
		if bc.Trace.Jaeger.TraceName != "" {
			traceName = bc.Trace.Jaeger.TraceName
		}
		traceEndpoint = bc.Trace.Jaeger.Endpoint
		traceRatio = bc.Trace.Jaeger.Ratio
	}
	if v := strings.TrimSpace(os.Getenv("TRACE_ENDPOINT")); v != "" {
		// 运行时允许接入外部 tracing；空值表示只使用本地 noop provider。
		traceEndpoint = v
	}

	// ===== 4. 初始化后台任务组 =====
	cleanupTaskGroup := taskgroup.Init()
	defer cleanupTaskGroup()

	// ===== 5. 初始化 OpenTelemetry（带兜底，不会因为没连上 Jaeger 就阻塞） =====
	tp := initTracerProvider(traceName, traceEndpoint, traceRatio, logger)
	// 进程退出前 flush 一下（不阻塞请求，只在退出时）
	defer func() {
		_ = tp.ForceFlush(context.Background())
	}()

	// ===== 5.5 启动时打一个 span，方便在 Jaeger 里排查 =====
	{
		tr := otel.Tracer("bootstrap")
		ctx, span := tr.Start(context.Background(), "startup-span")
		span.SetAttributes(
			semconv.ServiceNameKey.String(traceName),
		)
		span.End()
		_ = ctx
	}

	// ===== 6. 严格检查 Server / Data 配置，缺了就直接报错 =====
	serverCfg := bc.Server
	if serverCfg == nil {
		panic(fmt.Errorf("bootstrap server config is nil, please check %s", confPath))
	}

	dataCfg := bc.Data
	if dataCfg == nil {
		panic(fmt.Errorf("bootstrap data config is nil, please check %s", confPath))
	}
	overrideFromEnv(dataCfg, logger)

	// ===== 7. 组装应用（wireApp） =====
	// 这里 wireApp 里用到的 TracerProvider 类型要记得是 *tracesdk.TracerProvider
	app, cleanup, err := wireApp(serverCfg, dataCfg, logger, tp)
	if err != nil {
		panic(fmt.Errorf("wireApp init failed: %w", err))
	}
	if app == nil {
		panic("wireApp returned nil app")
	}
	defer cleanup()

	// ===== 8. 启动应用 =====
	if err := app.Run(); err != nil {
		panic(fmt.Errorf("app run failed: %w", err))
	}
}
