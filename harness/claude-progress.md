# 进度日志

## 当前已验证状态

- 仓库根目录：D:\nus\intern\new\e-commerce-video-recsys-mvp
- 标准启动路径：`./harness/init.sh`（或 `npm run dev`）
- 标准验证路径：`npm run lint`（tsc --noEmit）
- 当前最高优先级未完成功能：dash-001（系统健康与微服务拓扑展示）
- 当前 blocker：无

## 会话记录

### Session 001

- 日期：2026-06-18
- 本轮目标：在本项目中按 harness engineering 模板搭建基础 harness 结构。
- 已完成：创建根目录 AGENTS.md，以及 harness/ 下的 CLAUDE.md、init.sh、feature_list.json、claude-progress.md、session-handoff.md、clean-state-checklist.md、evaluator-rubric.md、quality-document.md、index.md。
- 运行过的验证：暂未运行功能验证；harness 文件本身按模板核对。
- 已记录证据：无功能级证据；feature_list.json 中所有功能保持 not_started。
- 提交记录：（待提交）
- 更新过的文件或工件：见“已完成”。
- 已知风险或未解决问题：本项目无 test 脚本，basics 验证以 `npm run lint`（类型检查）代替；init.sh 为 bash 脚本，在 Windows 下需通过 Git Bash 运行。
- 下一步最佳动作：选择 dash-001，运行 ./harness/init.sh 启动应用，记录首页健康指标的端到端验证证据。

### Session 002

- 日期：2026-06-18
- 本轮目标：方案B 第 1 步——把 tiktok-glocal/ 微服务补连贯 + 真数据库 + 丰富 seed，做到“按 user_id 出不同推荐、配置改动落库”。
- 已完成：
  - recommendation：Video 加 Score/Reason；策略产出可解释分数；新增 GET /api/v1/configs；PUT /configs 改为接收 {strategy_name, weight} 并落库；推荐响应包成 {data:{videos:[...]}}。
  - gateway：新增 /api/v1/login（标准库 HS256 真签名 JWT）、/api/v1/health（真探活聚合 + 请求计数算吞吐）、路由 /api/v1/configs（PUT 需 JWT 校验）。
  - content：candidates 查询补 created_at。
  - postgres/init.sh：扩充到 10 个跨品类视频（staggered created_at）+ 为 user_123/user_fashion/user_foodie 灌互动数据，使画像不同；rec_db 默认配置带 strategy_name/weight。
  - docker-compose（monorepo）：修复 gateway 缺少下游服务地址（原默认 localhost，容器内不可达）的 bug，并传入 JWT_SECRET。
- 运行过的验证：四服务 go vet + recommendation go test 全绿；gofmt 已套用；docker compose up --build 起全栈，curl 验证：
  - user_123(电子/科技) 与 user_fashion(时尚) 推荐排序明显不同，reason=interest_match:*；
  - GET/PUT configs 落库；无 token PUT→401，有 token→200；
  - 切 chronological 后排序变为按 created_at 倒序(reason=recency)；已重置回 engagement。
- 已记录证据：见本会话 curl 输出（真实跨服务 + DB + Redis 链路）。
- 提交记录：（尚未提交，等用户确认）
- 更新过的文件或工件：tiktok-glocal/services/{gateway,content,recommendation}/*.go、postgres/init.sh、docker-compose.yml。
- 已知风险或未解决问题：主机 8080/5432 被其它进程占用，compose 未把 gateway/postgres 发布到主机端口（不影响容器内部链路）；第 2 步前端接 gateway 时需处理 gateway 对外端口。
- 下一步最佳动作：第 2 步——React 前端从只调 server.ts mock 改为走 gateway（含线上单服务的轻量 mock 开关）。

### Session 003

- 日期：2026-06-18
- 本轮目标：方案B 第 2 步——前端走 gateway，保留 mock/真后端开关。
- 已完成：
  - server.ts 加入双模式：设 GATEWAY_URL 时把 /api/v1/* 全部转发给 gateway（本地全栈）；不设时用原内存 mock（线上单服务）。前端 api.ts 保持同源 /api/v1 不变，避免 CORS。
  - monorepo docker-compose：gateway 主机端口改为 ${GATEWAY_HOST_PORT:-8080} 可配置。
- 运行过的验证：
  - npm run lint 通过；
  - 代理模式（GATEWAY_URL=http://localhost:8080）：经 :3000 同源调用，user_123/user_fashion 推荐不同、health 真聚合、login 返回 gateway 真 JWT、configs 来自 Postgres；
  - mock 模式（不设 GATEWAY_URL）：:3000 返回静态 mock（throughput 1250、3 条固定视频），回归正常。
- 已记录证据：见本会话 curl 输出。
- 提交记录：（尚未提交第 2 步）
- 更新过的文件或工件：server.ts、tiktok-glocal/docker-compose.yml。
- 已知风险或未解决问题：本机 8080 原被 SmartFoxServer(sfs2x-service.exe) 占用，已按用户许可 kill；端口覆盖 GATEWAY_HOST_PORT 已证实可用。
- 下一步最佳动作：第 3 步——清理死代码(backend/、backend-go/)、提交的 .exe、AI Studio 残留，整理项目结构 + .gitignore。

### Session 004

- 日期：2026-06-19
- 本轮目标：修复“配置变更记录离开页面后丢失”的问题 + 清理无用内容 + 重跑。
- 已完成（修复）：根因是 AlgoConfig 的 Deployment Logs 是组件内存状态（config 本身写库是好的）。改为 DB 持久化：rec_db 新增 config_history 表；recommendation 在 PUT 时写入历史并新增 GET /api/v1/configs/history；gateway 路由该端点；server.ts mock 模式也加了对应内存历史；前端 AlgoConfig 进页面 fetchHistory、部署后从库刷新。
- 已完成（清理）：删除 backend/、backend-go/(死代码)、根 docker-compose.yml(旧架构)、.github/modernize/(残留)、4 个 *.exe、go-test.tmp.log、metadata.json(AI Studio 残留)；.env.example 重写为真实变量(GATEWAY_URL/PORT/JWT_SECRET)；.gitignore 增加 *.exe；新增根 .dockerignore 缩小 Node 镜像构建上下文。
- 运行过的验证：gateway/recommendation go vet + recommendation go test 通过；npm run lint 通过；down -v 重建卷后 up --build，curl 验证 history 从 [] → 两次部署后两条(新→旧)，重复请求仍在；前端 :3000 代理模式下 /api/v1/configs/history 返回持久化记录。
- 已记录证据：见本会话 curl 输出。
- 提交记录：（第 2 步 + 本轮修复/清理均尚未提交）
- 更新过的文件或工件：recommendation/{db.go,main.go}、gateway/main.go、postgres/init.sh、server.ts、src/services/api.ts、src/pages/AlgoConfig.tsx；删除若干死文件；.env.example/.gitignore/.dockerignore。
- 已知风险或未解决问题：config_history 里有几条我测试时写入的记录(engagement 0.85/0.7、chronological 0.5)，无害；如需干净可清空。
- 下一步最佳动作：用户复查前端历史是否持久；确认后提交（第 2 步 + 修复 + 清理）并继续第 4 步 CI/CD 对齐。
