# 可观测性与健康检查说明

本文档只描述当前项目真实存在的服务端观测基线，不写理想化规范。

## 当前已有的基线

### 日志

- `cmd/server/main.go` 启动时会初始化全局 logger
- `service`、`biz`、`data` 层默认都会打日志
- HTTP 层已内置 `request_id` 过滤器，日志会自动带上 `request_id`
- 日志会额外输出 `trace_sampled`；只有 trace 真正被采样时才输出 `trace_link_id`，供 Loki 安全跳转 Jaeger
- 关键鉴权与后台账号链路已保留成功 / 失败日志

### Trace

- HTTP JSON-RPC 路由已接入 tracing middleware
- gRPC 服务已接入 tracing middleware
- 自定义健康检查和静态路由已走统一观测包装，会补 span、recover 和收尾日志
- `biz.auth`、`biz.useradmin` 等关键 usecase 已显式创建 span
- 启动时会打一条 `startup-span`，方便排查进程是否成功初始化 tracer provider

### 健康检查

- `/ping`
- `/healthz`
- `/readyz`

其中：

- `/healthz` 只做浅检查
- `/readyz` 当前只检查 PostgreSQL 连通性
- 健康检查路由已有最小回归测试

### 启动韧性

- `data.NewData(...)` 在初始化 PostgreSQL 时会做短暂重试
- 目标是避免宿主机或数据库刚恢复时，服务因瞬时连接拒绝直接退出

## 当前已知盲区

以下点目前仍不算理想，需要后续业务按需补：

- `/readyz` 失败时虽然已有结构化日志，但响应体仍是简单文本
- JSON-RPC 入口日志仍以文本 `Infof/Warnf` 为主，字段化程度一般
- 当前 `request_id` 自动生成只覆盖 HTTP 链路，gRPC 和异步任务还没有统一 request id 策略

## 对部署模板的影响

### Compose

- Compose 模板当前保留 PostgreSQL `healthcheck`
- 业务容器默认依赖 `/healthz`、`/readyz` 作为发布后 smoke 检查入口

### Kubernetes

- `startupProbe` -> `/readyz`
- `readinessProbe` -> `/readyz`
- `livenessProbe` -> `/healthz`

## 后续业务常见补法

- 如果项目长期跑在 Kubernetes，优先把 `readyz` 扩展到真实依赖，并为失败响应补更细的 JSON 细节
- 如果项目依赖 Redis、MQ、OSS、第三方 API，再把这些依赖纳入 `/readyz`
- 如果项目有统一日志平台，优先把关键 JSON-RPC 链路改成结构化字段日志

## 约束来源

仓库级观测要求见：

- `AGENTS.md`

其中要求的重点是：

- 改服务端关键链路时，必须同时检查 trace 和 log
- 最终交付时要说明观测覆盖结果和剩余盲区
