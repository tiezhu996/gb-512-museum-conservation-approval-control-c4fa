请生成 `museum-conservation-approval-control`「文物保护处理审批」Go 全栈项目，服务博物馆保护部门管理文物、处理方案、材料检测和阶段性批准。它是文化遗产保护流程，不做内容社区、博客、展览预约或交易。

## 项目主要需求

复杂度下限：核心实体不少于 3 个、核心页面不少于 4 个、横切关注点不少于 2 个、共享前端组件不少于 3 个、自定义 hooks/utils 不少于 2 个、后端中间件不少于 2 个。

### 核心实体

`Artifact`（文物及风险等级）、`TreatmentPlan`（保护处理方案）、`MaterialTest`（材料检测）、`StageApproval`（阶段审批与意见）必须贯穿数据库、Go model/service/handler 和前端。

### 核心页面

`/artifacts` 文物清单；`/plans` 方案编排；`/tests` 材料检测；`/approvals` 阶段审批；`/audit` 审计。`RiskTag` 在文物和方案页共用，`ApprovalTimeline` 在方案和审批页共用。

### 横切关注点

RBAC 联动角色、Go 中间件、前端守卫和操作显隐；审批意见和版本不可覆盖，审计记录操作者与 request ID；实现全局错误处理和限流。

### 共享枚举/组件

同步 `ArtifactState`（registered/stable/treatment/closed）与 `ApprovalState`（draft/review/approved/rejected）。共享 `StatusBadge`、`ApprovalTimeline`、`EmptyState`，hooks 为 `useAuth`、`usePagination`。

### 技术与规模要求

前端 Vue 3 + TypeScript + Vite + Element Plus；后端 Go 1.22 + Gin + GORM；MySQL + Redis + MinIO。目标 2700–3900 行、26–38 个 `.go` 文件。

### 文件结构强制清单

前端 `api/stores/types/components/common/hooks/pages/router/utils`；后端 `model/dto/repository/service/handler/router/middleware/constants/util`；README 列明共享枚举位置，禁止合并职责。

### 结构红线

严禁合并职责到单一文件；方案、检测、审批和审计各层独立组织。

### 部署与交付

根目录必须提供 `docker-compose.yml`（顶层 `name: museum-conservation-approval-control`，且不写 `version:`）、`.env` 和 `.env.example`（均含 `COMPOSE_PROJECT_NAME=museum-conservation-approval-control`）、`README.md`、`frontend/Dockerfile`、`backend/Dockerfile` 和 `frontend/nginx.conf`。前端端口 `18512`、后端端口 `19512`；Nginx `/api` 代理，数据库健康检查、命名卷和 `condition: service_healthy` 齐全，提供真实 `/healthz` 和 Git 初始化。
