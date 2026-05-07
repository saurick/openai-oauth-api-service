# 🤖 AI 助手数据库变更操作手册 (必读)

**STOP! 在修改数据库或创建 `.sql` 文件之前，请务必阅读本指南。**

本项目使用 **Ent** 和 **Atlas** 进行版本化的数据库迁移。
**严禁** 在 `migrate/` 目录下手动编写 `.sql` 文件。
**严禁** 手动执行 `ALTER TABLE` 或 `CREATE TABLE` 语句。

## 🟢 正确的工作流 (HOW TO DO IT)

1.  **修改 Ent Schema (Go 代码)**:
    修改位于 `server/internal/data/model/schema/*.go` 的 Go 文件。
    例如：在 `Fields()` 方法中添加 `field.String("new_col")`。

2.  **生成迁移文件**:
    在 `server/` 目录下运行以下命令：
    ```bash
    make data
    ```
    *解释：此命令会自动运行 `atlas migrate diff` (根据你的 schema 变更生成 `.sql` 文件) 和 `ent generate` (更新 Go 客户端代码)。*

3.  **应用迁移**:
    运行：
    ```bash
    make migrate_apply
    ```
    *解释：此命令会将生成的 SQL 应用到实际数据库，并更新 `atlas_schema_revisions` 表。*

4.  **只补齐当前开发库已有迁移时的做法**:
    如果问题已经明确定位为“代码和迁移文件都已存在，但当前开发库还没 apply 到最新版本”，不要重新生成 migration，也不要手动改库；直接在 `server/` 目录执行：
    ```bash
    make migrate_apply
    ```
    执行后再做只读确认，至少核对：
    - `migrate_apply` 已成功应用到目标 revision，没有 checksum / drift 报错
    - 目标字段 / 索引 / 表已在当前开发库中可见

    **注意：**
    - 这里只适用于当前仓库开发配置命中的非生产库，例如 `server/configs/dev/config.yaml`、`server/configs/dev/config.local.yaml` 或用户明确指定的开发/个人测试库。
    - 如果当前 shell 或 `server/.env` 里带着 `DB_URL`、`POSTGRES_DSN` 或其他连接环境变量，必须先执行 `make print_db_url` 确认实际命中的库，再执行 `make migrate_apply`。
    - 如果目标库可能是生产库、共享测试库，或当前无法明确判断数据库归属，必须先说明将命中的库和风险，再等待确认。

## 🔴 严格禁止的操作 (WHAT NOT TO DO)

*   ❌ **绝对不要** 手动创建类似 `2024..._migrate.sql` 的文件。这会破坏 checksum 哈希校验 (`atlas.sum`)。
*   ❌ **绝对不要** 试图通过 `INSERT INTO` 或 `ALTER TABLE` 直接“修复”数据库结构而不走迁移流程。Atlas 会检测到结构漂移 (drift) 并报错。
*   ❌ **绝对不要** 随意删除迁移文件，除非你完全理解后果（这会破坏迁移历史图谱）。

## 🛠 常见问题处理

*   **Checksum Mismatch (校验和不匹配)**: 如果遇到此错误，请运行 `make migrate_hash`。
*   **开发库只是落后于仓库已有 migration**: 直接执行 `make migrate_apply`，不要因为“缺字段”就重新 `make data`，也不要跳版本。
*   **Drift Detected / Duplicate Column (字段已存在)**: 这通常表示数据库曾被手动改过，或当前库状态已经偏离迁移历史；不要把它和“开发库单纯还没 apply 最新 migration”混为一谈。只有在确认数据库已被手动改动、且你明确理解后果时，才考虑使用 `make migrate_set` 跳过版本，或先清理数据库中的脏状态。

---
**请严格遵守此流程以保证数据库完整性。**
