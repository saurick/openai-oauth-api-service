package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// dburl 统一解析当前仓库默认 PostgreSQL DSN，供迁移状态检查和打印当前命中库共用。
type bootstrapConfig struct {
	Data struct {
		Postgres struct {
			DSN string `yaml:"dsn"`
		} `yaml:"postgres"`
	} `yaml:"data"`
}

func main() {
	confPath := flag.String("conf", "./configs/dev/config.yaml", "config yaml path")
	flag.Parse()

	dsn := strings.TrimSpace(os.Getenv("POSTGRES_DSN"))
	if dsn == "" {
		raw, err := os.ReadFile(*confPath)
		if err != nil {
			fail("read config failed: %v", err)
		}

		var cfg bootstrapConfig
		if err := yaml.Unmarshal(raw, &cfg); err != nil {
			fail("parse config failed: %v", err)
		}

		dsn = strings.TrimSpace(cfg.Data.Postgres.DSN)
		// 本地联调优先允许未跟踪的 config.local.yaml 覆盖私有 DSN，避免公共 dev 配置被迫带密码。
		if localPath := resolveLocalConfPath(*confPath); localPath != "" {
			localRaw, err := os.ReadFile(localPath)
			if err != nil {
				fail("read local config failed: %v", err)
			}
			var localCfg bootstrapConfig
			if err := yaml.Unmarshal(localRaw, &localCfg); err != nil {
				fail("parse local config failed: %v", err)
			}
			if localDSN := strings.TrimSpace(localCfg.Data.Postgres.DSN); localDSN != "" {
				dsn = localDSN
			}
		}
	}

	if dsn == "" {
		fail("postgres dsn is empty in %s", *confPath)
	}

	normalized, err := normalizePostgresURL(dsn)
	if err != nil {
		fail("parse postgres dsn failed: %v", err)
	}

	fmt.Print(normalized)
}

func normalizePostgresURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return "", fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	if strings.TrimPrefix(u.Path, "/") == "" {
		return "", fmt.Errorf("postgres dsn missing db name")
	}
	q := u.Query()
	if q.Get("sslmode") == "" {
		q.Set("sslmode", "disable")
	}
	u.Scheme = "postgres"
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
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
