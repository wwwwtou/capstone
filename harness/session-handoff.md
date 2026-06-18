# 会话交接

## 当前已验证（截至 2026-06-19）

- 方案B 微服务全栈可真实交互：千人千面推荐、配置落库、JWT 鉴权、部署历史持久化——均经 curl + 自动化测试验证。
- 已写并通过的测试：gateway/recommendation Go 单测；server.ts mock 冒烟测试(9/9)；对真实栈的集成测试(14/14)；压测(1803 req/s, 0 错误, p99≈100ms，结果存 tests/stress/RESULTS.md)。
- 已提交(本地，尚未 push)：4 个 commit——harness 脚手架、微服务真实化、前端双模式+历史持久化、清理死代码。

## 本轮改动（尚未提交的部分）

- 新增 tests/{smoke,integration,stress}/ 与 k6 脚本；package.json 加 test:smoke/integration/stress 脚本。
- .github/workflows/ci.yml：webservice-build 加 API 冒烟测试步骤；新增 microservice-integration 作业(compose 起栈→集成+压测→拆栈)；deploy-render 依赖加 microservice-integration。
- 集成测试做了确定性修复（先设 engagement 再断言，因 DB 状态持久）。

## 仍未验证 / 风险

- ci.yml 改动**尚未做 YAML 校验**，也未 push 触发 CI 实跑——下次务必先校验再 push。
- 本机 8080 曾被 SmartFoxServer 占用(已 kill)；8090 也被占；干净端口用 18080。

## 下一步最佳动作

1. 校验 ci.yml（任意 YAML 解析器）。
2. git add -A 提交"测试 + CI"这一轮；然后 push 一次（会触发 CI，免费）。
3. push 后用 `gh run list/view` 看 CI 是否绿；红则修。
4. 之后整个约定计划(方案B 1-4 步 + 测试)就完成了。

## 命令

- 本地全栈：`cd tiktok-glocal-ecommerce-recsys-mvp && docker compose up -d`，再根目录 `GATEWAY_URL=http://localhost:8080 npm run dev`（前端 :3000）
- 纯前端 mock：根目录 `npm run dev`（不设 GATEWAY_URL）
- 测试：`npm run test:smoke`；栈起来后 `BASE=http://localhost:8080 npm run test:integration`；`BASE=... npm run test:stress`
- Go 测试：各服务目录 `go test ./...`
- 清理：`cd tiktok-glocal-ecommerce-recsys-mvp && docker compose down`
