// server/internal/data/data.go
package data

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"

	"server/internal/biz"
	"server/internal/conf"
	"server/internal/data/model/ent"
	entLogger "server/pkg/logger"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/XSAM/otelsql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.opentelemetry.io/otel/attribute"
)

// ProviderSet 是 data 层对外暴露的依赖注入集合。
var ProviderSet = wire.NewSet(
	NewData,

	// auth
	NewAuthRepo,
	wire.Bind(new(biz.AuthRepo), new(*authRepo)),
	NewTokenGenerator,
	biz.NewAuthUsecase,

	// admin auth / manage
	NewAdminAuthRepo,
	wire.Bind(new(biz.AdminAuthRepo), new(*adminAuthRepo)),
	NewAdminTokenGenerator,

	// user admin
	NewUserAdminRepo,
	wire.Bind(new(biz.UserAdminRepo), new(*userAdminRepo)),

	// gateway
	NewGatewayRepo,
	wire.Bind(new(biz.GatewayRepo), new(*gatewayRepo)),

	// jsonrpc
	NewJsonrpcData,
	wire.Bind(new(biz.JsonrpcRepo), new(*JsonrpcData)),
)

// Data 聚合所有外部资源（DB、JsonrpcData 等）。
type Data struct {
	log      *log.Helper
	postgres *ent.Client
	sqldb    *sql.DB
	conf     *conf.Data
}

const (
	postgresDriverName    = "pgx"
	postgresReadyTimeout  = 60 * time.Second
	postgresRetryInterval = 2 * time.Second
)

type pingContexter interface {
	PingContext(ctx context.Context) error
}

// waitForPostgresReady 在启动阶段为数据库预留短暂恢复窗口，避免宿主机重启后的瞬时连接拒绝导致应用直接退出。
func waitForPostgresReady(ctx context.Context, pinger pingContexter, interval time.Duration, l *log.Helper) error {
	if interval <= 0 {
		interval = time.Second
	}

	attempt := 0
	var lastErr error
	for {
		attempt++
		if err := pinger.PingContext(ctx); err == nil {
			if attempt > 1 {
				l.Infof("postgres ready after retry, attempt=%d", attempt)
			}
			return nil
		} else {
			lastErr = err
			l.Warnf("postgres not ready yet, attempt=%d err=%v", attempt, err)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("postgres not ready before timeout: %w, last_err=%v", ctx.Err(), lastErr)
		case <-time.After(interval):
		}
	}
}

// SQLDB 返回底层 DB，用于健康检查与原生 SQL 查询。
func (d *Data) SQLDB() *sql.DB {
	return d.sqldb
}

// NewData 由 wire 调用，用来统一管理资源和 cleanup。
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	l := log.NewHelper(log.With(logger, "logger.name", "data"))

	l.Info("init postgres(otelsql) start...")
	db, err := otelsql.Open(
		postgresDriverName,
		c.Postgres.Dsn,
		otelsql.WithSpanOptions(otelsql.SpanOptions{
			OmitConnResetSession: true,
			OmitConnPrepare:      true,
			OmitConnQuery:        false,
			OmitRows:             true,
			OmitConnectorConnect: true,
		}),
		otelsql.WithAttributesGetter(func(
			ctx context.Context,
			method otelsql.Method,
			query string,
			args []driver.NamedValue,
		) []attribute.KeyValue {
			attrs := make([]attribute.KeyValue, 0, 1+len(args))
			attrs = append(attrs, attribute.String("db.statement", query))
			for _, a := range args {
				key := fmt.Sprintf("db.sql.arg.%d", a.Ordinal)
				if a.Name != "" {
					key = "db.sql.arg." + a.Name
				}
				attrs = append(attrs, attribute.String(key, fmt.Sprint(a.Value)))
			}
			return attrs
		}),
	)
	if err != nil {
		l.Errorf("failed to open postgres connection: %v", err)
		return nil, nil, err
	}

	// 启动兜底：给 Postgres 预留就绪窗口，避免重启后短暂不可达直接触发 panic。
	pingCtx, cancelPing := context.WithTimeout(context.Background(), postgresReadyTimeout)
	defer cancelPing()
	if err := waitForPostgresReady(pingCtx, db, postgresRetryInterval, l); err != nil {
		_ = db.Close()
		l.Errorf("postgres ping failed: %v", err)
		return nil, nil, err
	}
	l.Info("init postgres(otelsql) done")

	postgresClient := ent.NewClient(
		ent.Log(entLogger.NewEntLogger(logger)),
		ent.Driver(entsql.OpenDB(dialect.Postgres, db)),
	)
	if postgresClient == nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("failed to create postgres client")
	}

	if c.Postgres.Debug {
		postgresClient = postgresClient.Debug()
	}

	data := &Data{
		log:      l,
		sqldb:    db,
		postgres: postgresClient,
		conf:     c,
	}

	if err := InitAdminUsersIfNeeded(context.Background(), data, c, l); err != nil {
		return nil, nil, err
	}
	if err := NewGatewayRepo(data, log.With(logger, "module", "data.api.seed")).EnsureDefaultModels(context.Background()); err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		if postgresClient != nil {
			_ = postgresClient.Close()
		}
		if db != nil {
			_ = db.Close()
		}
	}

	return data, cleanup, nil
}
