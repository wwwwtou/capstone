# 会话交接

## 当前状态（截至 2026-06-19，Session 008 结束）

**新增：容错机制(断路器+重试+限流) + PlantUML 架构图对齐，CI run #23 全绿，HEAD==origin/main==`613c687`。**

- 容错(纯 stdlib)：recommendation 对 user/content 调用加断路器+重试+冷启动降级(`degraded:true`)；gateway 加 per-IP 限流(429) + 反代断路器(503)。单测 42→53。真栈故障注入验证通过(stop user→200 兜底→恢复)。
- 架构图：`docs/architecture/` 下 8 张 PlantUML(源 .puml + 渲染 png)：logical/physical/deployment-cloud/ddd-context-map/er-diagram(详细，按 per-service 库分组)/sequence/cicd/usecase。本机渲染器：plantuml.jar=`C:\Users\Lenovo\.vscode\extensions\jebbs.plantuml-2.18.1\plantuml.jar`，Graphviz=`D:\wwtDownload\Graphviz\bin\dot.exe`。渲染命令见 docs/architecture/README.md。
- 技术清单仍缺(按性价比，下次可继续)：①E2E 自动化(Playwright，唯一缺的测试维度)；②DDD 分层落地到代码(目前服务多为扁平 main.go，class 图的 repository 接口未在代码实现)；③SonarQube(现用 gosec/golangci/govulncheck/npm audit 替代)；④JWT secret 去硬编码 fallback、加安全头、敏感数据加密；⑤水平扩展/负载均衡的具体配置。增值项(第7条)已通过容错机制满足。

---

## 历史状态（截至 Session 006 结束）

**方案B 全流程闭环 + 测试加强（单测全服务覆盖、压测改 JMeter），CI run #20 全绿（HEAD==origin/main==`0e0f3ae`）。**

- Session 006 增量：单测从「只 recommendation 26 个」扩到「全部 4 个 Go 服务 42 个」(CI job3 现遍历 `services/*/go.mod`)；压测从未实跑的 k6 换成本机 **JMeter**(`tests/stress/recommend.jmx`+`users.csv`+`README.md`)，并真实跑通：50 线程/30s/80ms think → 12841 样本 0 错误、430 req/s、p99 38ms，HTML dashboard 在 `tests/stress/jmeter-report/index.html`(gitignored，要截图就开这个)。CI 压测门禁仍用轻量 node 脚本。详见 claude-progress.md Session 006（含 JMeter 踩坑：CSV 取参、HttpClient4 keepalive 避免 Windows 端口耗尽、`-Jhost` 用 localhost 而非 127.0.0.1）。
- 本机 JMeter：`D:\wwtDownload\webserver\apache-jmeter-5.6.3\apache-jmeter-5.6.3\bin\jmeter.bat`（5.6.3，Java 8）。headless 跑法：`jmeter -n -t tests\stress\recommend.jmx -Jhost=localhost -Jport=8080 -Jthreads=50 -Jrampup=10 -Jduration=30 -Jthink=80 -l tests\stress\jmeter-results.jtl -e -o tests\stress\jmeter-report`。

---

## 历史状态（截至 Session 005 结束）

**方案B 全流程闭环完成，CI 全绿。**

- CI run #18（commit `de2775d`）= **completed success**，6 个作业全过：
  1. Go Quality and Security ✅
  2. Frontend Dependency Security（`npm audit --audit-level=high`）✅
  3. Unit Tests Report ✅
  4. Lint and Build Checks（含 API smoke test）✅
  5. **Microservice Integration Tests（docker compose 起真栈 → 集成 14 项 + 压测，CI 实跑通过）** ✅
  6. Deploy to Render（无 DEPLOY_HOOK_URL secret 时自跳过并成功）✅
- 本地 HEAD == origin/main == `de2775d`，工作树干净。

## 本轮（Session 005）做了什么

1. 校验 ci.yml：PyYAML(UTF-8) 解析通过，6 作业齐全；引用的 smoke/integration/stress 测试文件均在。
2. 本地预检 smoke 9/9 + lint + build 全过后 push（0f1a873）触发 CI run #17。
3. run #17 暴露唯一红：frontend-security 的 `npm audit` 有 4 个 high 漏洞（其余 4 作业含真栈集成全绿）。
4. `npm audit fix`（只动 package-lock.json，不改 package.json 声明版本）→ 0 漏洞；lint/build/smoke 复验全过；提交 `de2775d` 并 push → run #18 全绿。

## 重要：git push 凭据（下次会话照此即可非交互推送）

- 本机两份 Windows 凭据：
  - `git:https://github.com`（user **wlyIris**）→ 对 wwwwtou/capstone **无推送权**（push 报 403）。
  - `git:https://wwwwtou@github.com`（user **wwwwtou**）→ **有推送权**。
- GCM（helper=manager）在非交互 shell 会弹 GUI 卡死。已对本仓库做：
  - `git remote set-url origin https://wwwwtou@github.com/wwwwtou/capstone.git`
  - `git config --local credential.helper wincred`
- 因此现在 `git push origin main` 走 wincred + wwwwtou 凭据，**非交互直推**，不再卡 GCM。若再遇卡死：`Get-Process *credential* | Stop-Process -Force` 清掉残留 GCM 进程后重试。

## 没有 gh CLI —— 用 GitHub Actions REST API 查 CI

- 仓库公开，免鉴权可读：
  - 最近运行：`curl -fsS "https://api.github.com/repos/wwwwtou/capstone/actions/runs?per_page=3"`
  - 某次运行各作业：`curl -fsS ".../actions/runs/<RUN_ID>/jobs"`
- 用 python 解析 JSON（注意 Windows 默认 GBK，读本地文件要 `encoding='utf-8'`）。

## 仍未做 / 可选后续

- feature_list.json 的 4 个功能（dash-001/algo-001/algo-002/sim-001）是**浏览器端到端 UI 证据**轨道；其后端行为已被 CI 集成测试覆盖验证，但本会话未抓取浏览器级证据，故状态仍保留 not_started，未谎报 passing。如需收尾：起 `docker compose up -d` + 代理模式前端，按各 feature 的 verification 步骤在浏览器逐项截图取证。
- 端口提示：本机 8080 曾被 SmartFoxServer 占用，8090 也占；干净端口用 18080。

## 命令

- 本地全栈：`cd tiktok-glocal-ecommerce-recsys-mvp && docker compose up -d`，再根目录 `GATEWAY_URL=http://localhost:8080 npm run dev`（前端 :3000）
- 纯前端 mock：根目录 `npm run dev`（不设 GATEWAY_URL）
- 测试：`npm run test:smoke`；栈起来后 `BASE=http://localhost:8080 npm run test:integration`；`BASE=... npm run test:stress`
- Go 测试：各服务目录 `go test ./...`
- 清理：`cd tiktok-glocal-ecommerce-recsys-mvp && docker compose down`
