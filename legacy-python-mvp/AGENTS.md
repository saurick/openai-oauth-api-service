# Project Notes

This legacy MVP is an OpenAI-compatible API forwarding and usage metering reference.

- Keep request/response body logging off by default. Usage monitoring should store metadata, status, latency, byte counts, and token usage only.
- Prefer small, explicit FastAPI modules and SQLite-backed state until the project outgrows a single-node deployment.
- When deploying this legacy MVP to a shared low-disk Docker host, build images locally or in CI and let the server only load and run them. After the new container is healthy, clean only unused images and build cache with `docker image prune -a -f` and `docker builder prune -f`; do not prune volumes or delete database/config directories such as `/data`, compose `.env`, or upload folders.

## GPT 与 Codex 协作

本项目允许通过 GPT 进行需求澄清、架构讨论、方案比较和 Codex prompt 生成，但 GPT 输出不能直接替代本仓库真源。

当 Codex 接收来自 GPT 的执行 prompt 时，必须先审查：

- 是否符合本项目 `AGENTS.md`
- 是否符合当前 README、docs、Makefile、构建脚本和真实目录结构
- 是否误改禁止路径、生成产物、敏感配置或扩大任务范围
- 是否把规划、schema、迁移、runtime、前端接入、测试补齐、部署等多阶段内容混在一轮执行
- 是否需要先拆成更小的可验证阶段

Codex 应优先遵循本仓库真实代码、项目文档和当前工作区状态。若 GPT prompt 与项目真源冲突，应收窄或修正执行范围，并在最终回复中说明原因。

大型任务默认拆阶段执行，每一轮只完成一个可验证闭环。执行后应反馈已完成内容、未做内容、验证结果、剩余风险，以及建议下一步交给 GPT 分析的问题。
