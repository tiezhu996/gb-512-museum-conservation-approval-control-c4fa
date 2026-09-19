# 验收记录

- 日期：2026-08-22
- 静态检查：Go 1.22 下 `go test ./...`、`go test -race ./...`、`go vet ./...`、`go build ./...` 通过；Vue 类型检查和 Vite 生产构建通过；`docker compose config --quiet` 通过。
- 容器启动：MySQL、Redis、MinIO、backend、frontend 从空数据卷启动成功并达到 healthy，`GET /healthz` 返回 200，后端运行日志未见异常。
- API 流程：管理员登录、概览、4 个实体列表、创建文物、合法状态迁移、会话、脱敏运行配置、审计列表及审计汇总均通过。
- RBAC 与审批版本：viewer 写操作返回 403；operator 可提交 `draft -> review`，但批准返回 422；reviewer 批准后生成第二条独立意见。意见 v2/v3 均保留 actor、request ID 和原文，审批开始后普通字段更新被阻止。
- 内置 Browser：验证文物、处理方案、材料检测、阶段审批、审计记录 5 个页面；`RiskTag`、双页 `ApprovalTimeline` 和 `EmptyState` 接入正常；实际批准 `SA-002` 后时间线新增 v2/admin 意见并在审计回显请求 ID；控制台 0 error / 0 warning，桌面截图未见遮挡或错位。
- 规模：3074 行 Go 功能代码，38 个非测试 `.go` 文件。
- 清理：验收完成后执行 `docker compose down -v --remove-orphans`，清除本项目容器和数据卷。
