# 历史 FastAPI API 转发 MVP

一个早期 OpenAI 兼容 API 转发与用量记录 MVP，用官方 OpenAI API key 作为上游凭据，向下游分发独立 API key，并记录每个 key 的请求量、状态码、延迟、字节数和 token 用量。

## 边界

这个项目和 `sub2api` 在“统一入口、统一代理、分发 key、计量监控”这些形态上相似，但认证真源不同：

| 项目 | 说明 |
| --- | --- |
| 支持 | 使用 `OPENAI_API_KEY` 转发到 `https://api.openai.com/v1` |
| 支持 | 给下游用户创建独立的 `ogw_...` API key |
| 支持 | 记录用量汇总、最近请求、上游状态码、延迟和 token |
| 支持 | 通过 `UPSTREAM_PROXY_URL` 统一配置上游 HTTP/SOCKS 代理 |
| 不支持 | 抓取、复用或分享 Codex/ChatGPT 登录态、Cookie、设备码、个人账号 token |
| 不支持 | 把个人订阅账号包装成多人共享 API |

OpenAI 当前 API 真源是官方 API endpoint，例如 `POST /v1/responses` 使用 `Authorization: Bearer $OPENAI_API_KEY`。OpenAI 权限文档也把 Project API Keys、Service Accounts、Responses API、Usage 等列为平台权限项。

## 快速开始

```bash
cd <repo>
python3 -m venv .venv
source .venv/bin/activate
pip install -e '.[dev]'
cp .env.example .env
```

编辑 `.env`：

```bash
OPENAI_API_KEY=sk-proj-your-real-key
ADMIN_TOKEN=use-a-long-random-admin-token
UPSTREAM_PROXY_URL=socks5://127.0.0.1:7890
```

初始化一个下游 key：

```bash
source .env
openai-oauth-api-service create-key --name alice --rpm-limit 60 --daily-token-limit 100000
```

启动服务：

```bash
source .env
uvicorn api_service.app:app --host 127.0.0.1 --port 8080 --reload
```

下游按 OpenAI 兼容方式调用：

```bash
curl http://127.0.0.1:8080/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ogw_your_downstream_key" \
  -d '{
    "model": "gpt-5.4",
    "input": "hello"
  }'
```

也可以调用旧版兼容路径：

```bash
curl http://127.0.0.1:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ogw_your_downstream_key" \
  -d '{
    "model": "gpt-5.4",
    "messages": [{"role": "user", "content": "hello"}]
  }'
```

## 管理接口

管理接口用 `ADMIN_TOKEN` 鉴权，支持 `Authorization: Bearer ...` 或 `X-Admin-Token`。

创建下游 key：

```bash
curl http://127.0.0.1:8080/admin/keys \
  -H "Content-Type: application/json" \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -d '{"name":"bob","rpm_limit":30,"daily_token_limit":50000}'
```

查看最近 24 小时汇总：

```bash
curl "http://127.0.0.1:8080/admin/usage/summary?hours=24" \
  -H "X-Admin-Token: $ADMIN_TOKEN"
```

查看最近请求：

```bash
curl "http://127.0.0.1:8080/admin/usage/recent?limit=50" \
  -H "X-Admin-Token: $ADMIN_TOKEN"
```

吊销 key：

```bash
curl -X POST "http://127.0.0.1:8080/admin/keys/<key_id>/revoke" \
  -H "X-Admin-Token: $ADMIN_TOKEN"
```

## CLI

```bash
openai-oauth-api-service list-keys
openai-oauth-api-service usage --hours 24
openai-oauth-api-service recent --limit 50
openai-oauth-api-service revoke-key <key_id>
```

## 用量口径

- 非流式 JSON 响应会读取 `usage.input_tokens/output_tokens/total_tokens` 或 `usage.prompt_tokens/completion_tokens/total_tokens`。
- Responses API 流式响应会从 `response.completed` SSE 事件中提取 `response.usage`。
- Chat Completions 流式响应只有在上游返回包含 `usage` 的 chunk 时才能记录 token；否则仍记录请求数、状态码、延迟和字节数。
- SQLite 时间口径使用 UTC。

## 测试

```bash
pytest
```

测试使用 `httpx.MockTransport` 模拟上游，不会调用真实 OpenAI API。
