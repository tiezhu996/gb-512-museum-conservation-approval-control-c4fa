# 验收记录

- 日期：2026-09-19
- 检测快照门禁：审批 `draft -> review` 按关联编码定位处理方案并冻结最新已核验检测（编码+版本）；缺方案返回 422「未匹配到处理方案」、无已核验检测返回 422「没有已核验的材料检测」，状态与意见保持不变（服务测试断言版本与意见数不变）。
- 批准复核：冻结检测被改判或换版时批准返回 422 具体原因，记录保持 `review`，仅 `gate_verdict` 落库并写入 `gate-blocked` 审计；检测保持已核验则批准通过并记录「门禁通过」。HTTP 冒烟确认 SA-001 拦截、SA-002/SA-CONC 通过。
- 只完成一次：并发双批准（同 expectedVersion）HTTP 实测 1 成功 1 拒绝；重复批准返回非法迁移；服务测试覆盖乐观锁冲突与状态机拒绝两条路径。
- 页面：审批列表新增「检测门禁」列（冻结检测+版本+结论着色），方案页新增 `GateVerdictPanel` 展示关联审批快照；迁移失败原因经 `store.error` 在页面告警展示。
- 回归：`go test ./...`、`go vet ./...`、`go build ./...` 通过；`npm run typecheck`、`npm run build` 通过；登录、既有状态流与启动方式未改动。

# 历史验收（2026-08-22）

- 静态检查：Go 1.22 下 `go test ./...`、`go test -race ./...`、`go vet ./...`、`go build ./...` 通过；Vue 类型检查和 Vite 生产构建通过；`docker compose config --quiet` 通过。
- 容器启动：MySQL、Redis、MinIO、backend、frontend 从空数据卷启动成功并达到 healthy，`GET /healthz` 返回 200，后端运行日志未见异常。
- API 流程：管理员登录、概览、4 个实体列表、创建文物、合法状态迁移、会话、脱敏运行配置、审计列表及审计汇总均通过。
- RBAC 与审批版本：viewer 写操作返回 403；operator 可提交 `draft -> review`，但批准返回 422；reviewer 批准后生成第二条独立意见。意见 v2/v3 均保留 actor、request ID 和原文，审批开始后普通字段更新被阻止。
- 内置 Browser：验证文物、处理方案、材料检测、阶段审批、审计记录 5 个页面；`RiskTag`、双页 `ApprovalTimeline` 和 `EmptyState` 接入正常；实际批准 `SA-002` 后时间线新增 v2/admin 意见并在审计回显请求 ID；控制台 0 error / 0 warning，桌面截图未见遮挡或错位。
- 规模：3074 行 Go 功能代码，38 个非测试 `.go` 文件。
- 清理：验收完成后执行 `docker compose down -v --remove-orphans`，清除本项目容器和数据卷。
