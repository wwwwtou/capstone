# Harness 工件说明

这些文件构成本仓库的 harness engineering 基础结构，用于支撑长时运行的 coding agent 工作流。除根目录的 `AGENTS.md` 外，所有 harness 工件都放在 `harness/` 目录下。

## 核心四件套（先从这里开始）

1. **AGENTS.md / CLAUDE.md** — 根指令，定义工作规则、开工前要做什么、过程纪律和完成检查。agent 每轮会话最先读它。`AGENTS.md` 在仓库根目录，`CLAUDE.md` 在 `harness/`，两者规则一致。

2. **init.sh** — 启动脚本，自动完成依赖安装、基础验证，并打印启动命令。脚本中需要按项目实际情况设置 `INSTALL_CMD`、`VERIFY_CMD`、`START_CMD` 三个命令。本项目分别为 `npm install`、`npm run lint`、`npm run dev`。设置 `RUN_START_COMMAND=1` 可让脚本直接启动应用。

3. **claude-progress.md** — 会话日志，记录已验证的项目状态、当前优先级、blocker，以及逐会话的目标、运行过的验证和下一步。

4. **feature_list.json** — 机器可读的功能清单，每个条目跟踪状态（`not_started`、`in_progress`、`blocked`、`passing`）、验证步骤和证据。任意时刻只能有一个功能处于 `in_progress`。

## 附加文件（用于较长项目）

- **session-handoff.md** — 会话结束时的交接摘要，覆盖已验证功能、本轮改动、已知问题和推荐的下一步动作。

- **clean-state-checklist.md** — 完成前的核对清单，确保仓库可交给下一轮会话：启动路径可用、验证可跑、日志已更新、没有未记录的半成品。

- **evaluator-rubric.md** — 六维度评分（正确性、验证、范围纪律、可靠性、可维护性、交接准备度），用于评估 agent 产出质量。

- **quality-document.md** — 跨产品领域和架构层的纵向健康快照，跟踪代码库随时间变强还是变弱。

## 使用方式

每轮会话遵循 `AGENTS.md` 中的“开工流程”：确认目录 → 读 `harness/claude-progress.md` → 读 `harness/feature_list.json` → 看 git log → 运行 `./harness/init.sh` → 跑基础验证。然后只选一个未完成功能推进，验证通过并记录证据后才能标记为 `passing`，最后按“收尾”更新进度并提交。
