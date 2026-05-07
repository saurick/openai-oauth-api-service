// server/cmd/dbcheck/main.go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// 默认参数：本机开发 / docker-compose
// - 本机直接连 127.0.0.1:5433
// - 在 compose 容器里跑的话，把 host 改成 postgres
const (
	defaultHost = "127.0.0.1"
	defaultPort = "5433"
	defaultDB   = "openai_oauth_api_service"
)

// 方便通过环境变量覆盖
func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type postgresAccount struct {
	Name string
	DSN  string
}

func main() {
	host := getenv("POSTGRES_HOST", defaultHost)
	port := getenv("POSTGRES_PORT", defaultPort)
	db := getenv("POSTGRES_DB", defaultDB)

	accounts := []postgresAccount{
		{
			Name: "postgres",
			DSN:  buildPostgresDSN("postgres", getenv("POSTGRES_PASSWORD", "dev-postgres-password"), host, port, db),
		},
	}
	if appUser := os.Getenv("POSTGRES_APP_USER"); appUser != "" {
		accounts = append(accounts, postgresAccount{
			Name: appUser,
			DSN:  buildPostgresDSN(appUser, getenv("POSTGRES_APP_PASSWORD", ""), host, port, db),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, acc := range accounts {
		fmt.Println("-------------------------------------------------")
		fmt.Printf("Testing PostgreSQL account: %s\n", acc.Name)
		fmt.Printf("DSN (masked): %s\n", maskPassword(acc.DSN))

		if err := testPostgres(ctx, acc.DSN); err != nil {
			fmt.Printf("❌ %s connect failed: %v\n", acc.Name, err)
		} else {
			fmt.Printf("✅ %s connect OK\n", acc.Name)
		}
	}
}

func buildPostgresDSN(user, password, host, port, db string) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		url.QueryEscape(user),
		url.QueryEscape(password),
		host,
		port,
		db,
	)
}

func testPostgres(ctx context.Context, dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("sql.Open: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("PingContext: %w", err)
	}

	// 简单再跑个查询确认权限
	var now time.Time
	if err := db.QueryRowContext(ctx, "SELECT NOW()").Scan(&now); err != nil {
		return fmt.Errorf("SELECT NOW(): %w", err)
	}
	fmt.Println("server time:", now.Format(time.RFC3339))
	return nil
}

func maskPassword(dsn string) string {
	// 格式: postgres://user:pass@host/db
	// 思路:
	//   1) 找到最后一个 '@' 作为“用户信息和地址的分界”
	//   2) 在这个 @ 之前，找到最后一个 ':' 当成“密码起点”
	at := strings.LastIndex(dsn, "@")
	if at == -1 {
		// 没有 @，就不处理了
		return dsn
	}

	beforeAt := dsn[:at]
	afterAt := dsn[at:] // 包含 @

	colon := strings.LastIndex(beforeAt, ":")
	if colon == -1 {
		// 没有 :，也不处理
		return dsn
	}

	// user:***@tcp(...)
	masked := beforeAt[:colon] + ":***" + afterAt
	return masked
}
