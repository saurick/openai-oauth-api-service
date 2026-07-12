# OpenAI OAuth API Service 协作约定

本仓库长期维护 OAuth 登录、下游 API key 与 token/usage 管理；不是通用网关或纯转发项目。通用工程、Git、删除和文档规则使用全局 AGENTS。

## 当前真源

- 后端：`server/`；前端：`web/`；Compose：`server/deploy/compose/prod/`。
- `legacy-python-mvp/` 只作历史参考；设计与演进在 `docs/`；`progress.md` 只记过程。
- OAuth、key、usage、限流读 `docs/architecture.md`；服务端读 `server/README.md`、`server/docs/README.md`；页面读 `web/README.md`；部署读 `server/deploy/README.md`。

## 项目 Skills

- 项目 skills 位于 `.agents/skills/`，入口见其 README；只保留本服务专项 SOP。
- 默认选一个主 skill，跨页面、服务边界、测试或 operations 时再组合。
- 502/balance/usage 诊断、stale/日志、keys/tokens 安全、低配发布和回滚使用 `$openai-oauth-operations-governance`。
- 提示词整理显式使用全局 `$prompt-governance`；Git 收口使用 `$git-closeout-coordination`。
- 修改 skill 后同步 metadata/引用，运行 validator、YAML/metadata、引用扫描和 `git diff --check`。

## 工程与安全基线

- 保留质量门禁、错误码、health/ready、基础可观测和 Ent + Atlas migration；不直接手写结构性 SQL。
- 生产密钥、DB 密码、代理凭据和 token 只通过 env/Secret 注入；配置模板只能使用明显占位符。
- 涉及包管理器、CLI/SDK 或第三方配置时运行 secrets 扫描。
- 请求/响应正文默认不落库；usage 记录 key 标识、模型、状态、延迟、字节/token 和错误分类。
- 日志禁止完整 token、认证信息、prompt、模型输出正文和不必要 PII。
- 服务端按 `service -> biz -> data` 现有分层；前端复用鉴权、请求、错误 helper 和布局组件。
- 后台页面同时支持浅/暗色；目标表单/弹窗/表格/卡片变更用 `style:l1` 覆盖两种主题和溢出。
- 错误码以服务端目录为真源并生成前端码表；保持一码一义。

## 部署与迁移

- 主路径是 Docker Compose；低配服务器只 load、migration、启动和 smoke，不执行构建。
- Atlas 使用部署文档规定的宿主机 `/usr/local/bin/atlas`、目标 DSN 和 `flock`，不加入业务 Compose。
- 发布绑定 commit/image、migration、health/ready、admin 及本轮真实业务链和 rollback point。
- 清理前确认当前与回滚镜像保留策略，不把 `image prune -a` 作为无条件默认；禁止 volume/data/env 删除。
- 当前个人部署管理员账号口径按正式部署文档执行，不擅自生成或同步随机密码。
- Kubernetes、dashboard、lab-ha 和旧 SSH 发布脚本不在当前主路径；需要时重新评审。

## 过程、上下文与收口

- 触达代码、正式文档或部署配置后更新 `progress.md`：完成、下一步、阻塞/风险。
- GPT/ChatGPT 只作输入，执行前核对本文件、正式 docs、代码、migration、测试和 worktree。
- Codex 压缩/恢复测试必须显式使用本轮 JSONL 的 `thread_id`；不要用可能捡到其他会话的 `resume --last` 作验收。
- 精确 stage 本轮范围，push 前 fetch 并确认 upstream；提交信息使用简体中文。
