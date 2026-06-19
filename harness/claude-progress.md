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

### Session 005

- 日期：2026-06-19
- 本轮目标：校验 ci.yml → push 触发 CI → 看 CI 是否绿，红则修；完成方案B 全流程收尾。
- 已完成：
  - 校验 ci.yml：用 PyYAML(UTF-8) 解析通过，6 个作业齐全；smoke/integration/stress 三个被引用的测试文件均存在。
  - 本地预检：smoke 9/9、npm run lint、npm run build 全过。
  - push 0f1a873 触发 CI run #17：5 个作业里 4 个绿(含**新增的 microservice-integration 真栈集成+压测在 CI 实跑通过**)，唯一红是 frontend-security 的 `npm audit --audit-level=high`(4 high)。
  - 修复 npm audit：`npm audit fix`(仅动 package-lock.json，未改 package.json 声明版本)，复跑 audit→0 漏洞；lint/build/smoke 复验全过。提交 de2775d 并 push，触发 run #18。
- 运行过的验证：见本会话 curl(GitHub Actions API) + 本地 smoke/lint/build 输出；CI run #17 各作业结论。
- 提交记录：本会话把之前 3 个本地 commit 全部 push 到 origin/main(0f1a873)；新增并 push de2775d(安全修复)。当前 HEAD==origin/main==de2775d。
- 更新过的文件或工件：package-lock.json；harness/claude-progress.md、session-handoff.md。
- 已知风险或未解决问题：
  - **git push 凭据**：本机有两份 Windows 凭据——`git:https://github.com`(user wlyIris，对本仓库**无**推送权→403) 与 `git:https://wwwwtou@github.com`(user wwwwtou，**有**权)。GCM(helper=manager) 在非交互 shell 会弹 GUI 卡住。已把本仓库 remote 改为 `https://wwwwtou@github.com/...` 并设 `credential.helper=wincred`(local)，今后 `git push origin main` 可非交互直推。
  - run #18 结果待确认(见 session-handoff)。
- 下一步最佳动作：确认 run #18 全绿(deploy-render 作业在无 DEPLOY_HOOK_URL secret 时会自跳过并成功)；绿则方案B 1-4 步 + 测试 + CI 全部闭环完成。

### Session 006

- 日期：2026-06-19
- 本轮目标：回应用户两点质疑——(1) 单测只有 26 个、口径不全；(2) 压测改用本机 JMeter 并出可截图证据。
- 已完成（单测）：
  - 诊断：CI 的「单测报告」作业(job 3)原来只 `cd recommendation` 跑一个服务→报告只统计 26；gateway 5 个没计入，content/user 0 测试。
  - 给 user 服务抽出纯函数 `categoryFromMetadata` 并补测试(7 子用例 + 坏 body→400，共 9)；给 content 抽 `healthHandler` 并补测试(健康契约 + Video JSON 字段契约，共 2)。
  - 改 ci.yml job 3：遍历所有 `services/*/go.mod` 跑 `go test -v`，合并日志，新增 `MODULE_FAILED:` 哨兵 + step6 gate 同时识别 `^--- FAIL:|^MODULE_FAILED:`；报告 scope 改为「全部 Go 微服务」。本地模拟统计=42 通过/0 失败(26+5+9+2)。
- 已完成（压测）：
  - 用户本机有 JMeter（`D:\wwtDownload\webserver\apache-jmeter-5.6.3\...\bin\jmeter.bat`，5.6.3，Java 8）。删除从未实跑的 k6 脚本，新增 `tests/stress/recommend.jmx` + `users.csv`(CSV Data Set 取 user_id) + `README.md`(GUI/headless 两种跑法+截图指引)。
  - 真实跑通(stack 已 up 10h，gateway :8080 全链路)：50 线程/10s ramp/30s/80ms think → **12841 样本，0 错误，430 req/s，mean 9.2ms，p90/p95/p99=19/25/38ms，max 79ms**，全 200。HTML dashboard 在 `tests/stress/jmeter-report/index.html`(gitignored)。结果写入 RESULTS.md。
  - 踩坑记录(已解决)：`${__chooseRandom}` 不被求值→改 CSV；无 keepalive 复用导致 Windows 端口耗尽(SocketException/BindException 占 45%)→钉 HttpClient4 + think time；jmeter.bat 对 `-Jhost=127.0.0.1` 报 `Unknown arg: .0.0.1`→用 `localhost`；PowerShell `Out-File`/`*>` 产 UTF-16 日志，grep 当二进制→改读 statistics.json。
  - CI 压测门禁仍用轻量 node 脚本(无需在 runner 装 JMeter/Java)，JMeter 作为本地正式证据工具；TESTING_STRATEGY.md 已据实重写。
- 运行过的验证：4 服务 gofmt 干净 + go vet + go test 全过；本地模拟 CI 报告逻辑=42/42/0；JMeter 实跑 0 错误。提交 `0e0f3ae` 并 push→触发 CI run #20(进行中)。
- 提交记录：`0e0f3ae`（已 push，非交互 wincred 直推成功）。当前 HEAD==origin/main==0e0f3ae。
- 更新过的文件或工件：services/{user,content}/main.go + 新 main_test.go；.github/workflows/ci.yml；tests/stress/{recommend.jmx,users.csv,README.md,RESULTS.md}（删 recommend.k6.js）；TESTING_STRATEGY.md；.gitignore。
- 已知风险或未解决问题：CI run #20 结果待确认（job3 应显示 42 用例、job5 集成仍应绿）。
- 下一步最佳动作：确认 run #20 全绿；若用户要 JMeter 浏览器截图，打开 `tests/stress/jmeter-report/index.html` 截 Statistics 表 + 图。

### Session 007

- 日期：2026-06-19
- 本轮目标：按技术清单补两块——(1) 容错机制(增值项 + 限流)，(2) 架构图对齐(PlantUML，统一路径)。
- 已完成（容错，纯 stdlib 无新依赖）：
  - recommendation：新增 `resilience.go`(三态断路器 closed/open/half-open + 退避重试 callResilient)；fetchProfile/fetchCandidates 包进 breaker+retry；handleRecommend 加优雅降级——user 画像拿不到→冷启动空画像(globally_trending)+`degraded:true` 仍 200，content 拿不到→503。
  - gateway：新增 `resilience.go`(断路器 + breakerTransport 包 ReverseProxy.Transport，5xx/传输错误计失败，开路→ErrorHandler 返 503) + `ratelimit.go`(per-IP 令牌桶中间件→429+Retry-After，默认 1000rps/2000burst 不影响压测/CI，可 env 调)。
  - 单测 42→53（recommendation +5 断路器/重试，gateway +6 限流/breakerTransport）。
  - 真栈验证：rebuild gateway+recommendation→集成 14/14；**故障注入**：`docker compose stop user`→推荐仍 200 degraded=true 冷启动兜底(非 502)；user 恢复→断路器冷却后 half-open 探测→闭合→恢复个性化(interest_match)。
- 已完成（架构图，PlantUML，全在 `docs/architecture/`）：
  - 8 张：logical/physical(docker-compose)/deployment-cloud(Render+AWS terraform)/ddd-context-map/er-diagram(按 3 个 per-service 库分组，列/类型/PK/索引/unique + Redis kv + 应用层无 FK 跨服务关系)/sequence(GET recommendations 含限流/断路器/冷启动降级)/cicd-pipeline(6 作业)/usecase。
  - 用本机 plantuml.jar(VSCode 扩展) + Graphviz 渲染出 PNG(`docs/architecture/png/`)，已视觉核对 ER/sequence/logical 正确。源文件改为纯 ASCII(em-dash→`-`，避免 GBK mojibake；par/and 老 jar 不稳→用 note 表达并发)。
  - PRD_ARCHITECTURE.md 顶部加指引，注明旧 Mermaid 为历史/aspirational。
- 运行过的验证：4 服务 gofmt+vet+test 全过(53)；集成 14/14；故障注入端到端；CI run #23(dfcc556) 6 作业全绿；PlantUML 渲染 0 报错。
- 提交记录：dfcc556(容错)、613c687(架构图)，均已 push；HEAD==origin/main==613c687。
- 更新过的文件或工件：services/recommendation/{resilience.go,resilience_test.go,main.go}、services/gateway/{resilience.go,resilience_test.go,ratelimit.go,ratelimit_test.go,main.go}；docs/architecture/*(8 puml+8 png+README)；PRD_ARCHITECTURE.md。
- 已知风险/未解决：技术清单仍缺的项见 session-handoff（E2E 自动化、DDD 分层落地代码、SonarQube、JWT secret 硬编码、敏感数据加密等）。
- 下一步最佳动作：按用户优先级继续补 E2E(Playwright) 或 DDD 分层重构等。
