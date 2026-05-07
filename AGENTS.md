# OpenAI OAuth API Service 协作约定

本仓库是长期维护的 OAuth 登录、API key 生成与 token/usage 统计管理项目，不是网关项目，也不是单纯转发项目。

## 当前真源

- 后端主路径：`server/`
- 前端主路径：`web/`
- 单机部署主路径：`server/deploy/compose/prod/`
- 历史 Python MVP 仅作参考：`legacy-python-mvp/`
- 设计与演进说明：`docs/`
- 进度记录：`progress.md`

涉及代码修改前，先按任务阅读对应入口：

- OAuth 登录、API 转发、下游 key、usage、限流：先读 `docs/architecture.md`
- 服务端运行、配置、迁移、观测：先读 `server/README.md` 与 `server/docs/README.md`
- 前端后台、鉴权、页面：先读 `web/README.md`
- 部署：先读 `server/deploy/README.md`

## 合规边界

- 上游认证只允许使用 OpenAI 官方 API key，优先使用 Project API key 或 Service Account key。
- 禁止实现、引导或保留任何抓取、存储、复用、分享 Codex / ChatGPT 登录态、Cookie、设备码、浏览器会话、个人账号 token 的逻辑。
- 禁止把个人订阅账号包装成多人共享 API。
- 下游用户只能拿到本系统签发的 API key；真实 OpenAI API key 不返回给前端或下游调用方。

## 工程基线

- 保留质量门禁、错误码治理、健康检查、基础可观测性、数据库迁移工作流。
- Schema 变更必须走 Ent + Atlas 迁移流程，不直接手写结构性 SQL。
- 生产配置中的密钥、数据库密码、OpenAI API key、代理凭据必须通过环境变量或部署 Secret 注入，不写入仓库。
- 请求体和响应体默认不落库；usage 监控优先记录 key、模型、状态码、延迟、字节数、token 用量和错误类型。
- 结构化日志禁止记录完整 token、OpenAI API key、代理认证信息、用户 prompt 或模型输出正文。

## 部署边界

- 当前仓库主部署方式是 Docker Compose。
- Kubernetes、dashboard、lab-ha 和远端 SSH 发布脚本已从主路径裁剪；后续确实需要时再按真实环境新增，不从旧模板回填占位清单。
- Compose 环境变量以 `server/deploy/compose/prod/.env.example` 为入口，真实 `.env` 不提交。

## 代码修改约定

- 修改已有代码先遵循当前分层：`server -> service -> biz -> data`。
- 前端优先复用现有鉴权、请求封装、错误提示 helper 和布局组件。
- 新增业务错误码时，服务端错误码目录是唯一真源，并同步生成前端码表。
- 每轮触达代码、文档或部署配置后，更新 `progress.md`，至少写明：完成、下一步、阻塞/风险。
