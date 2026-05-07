# Compose 部署

本目录是 OpenAI OAuth API Service 当前主部署路径，包含 PostgreSQL 和后端服务。

## 文件

| 文件 | 说明 |
| --- | --- |
| `compose.yml` | PostgreSQL + app-server |
| `.env.example` | 环境变量示例，复制为 `.env` 后填写真实值 |

## 启动

```bash
cd server/deploy/compose/prod
cp .env.example .env
docker compose -f compose.yml up -d
```

至少替换以下值：

```bash
POSTGRES_PASSWORD=...
POSTGRES_DSN=...
OAUTH_API_JWT_SECRET=...
OAUTH_API_ADMIN_PASSWORD=...
OPENAI_API_KEY=...
```

如需开启 OAuth/OIDC 登录，还需要配置：

```bash
OAUTH_API_OAUTH_ENABLED=true
OAUTH_API_OAUTH_PROVIDER_NAME=OIDC
OAUTH_API_OAUTH_CLIENT_ID=...
OAUTH_API_OAUTH_CLIENT_SECRET=...
OAUTH_API_OAUTH_AUTH_URL=https://idp.example.com/oauth2/authorize
OAUTH_API_OAUTH_TOKEN_URL=https://idp.example.com/oauth2/token
OAUTH_API_OAUTH_USERINFO_URL=https://idp.example.com/oauth2/userinfo
OAUTH_API_OAUTH_REDIRECT_URL=https://your-domain.example.com/auth/oauth/callback
OAUTH_API_OAUTH_SCOPES=openid,profile,email
```

如需统一上游代理：

```bash
UPSTREAM_PROXY_URL=socks5://127.0.0.1:7890
```

## 说明

- Compose 主路径不包含 Jaeger，`TRACE_ENDPOINT` 默认为空；后续需要 tracing 时接入外部 OTLP endpoint。
- 真实 `.env`、OpenAI API key、代理认证信息和数据库密码不得提交到仓库。
- 当前不保留远端 SSH 发布脚本；发布流程确定后再补自动化脚本。
