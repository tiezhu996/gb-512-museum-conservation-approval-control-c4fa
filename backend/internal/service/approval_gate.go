package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/blueship581/museum-conservation-approval-control/backend/internal/constants"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/model"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/repository"
	"gorm.io/gorm"
)

// GatePlanReader 只暴露门禁需要的方案查询能力，便于单测替换与边界收窄。
type GatePlanReader interface {
	FindByCode(context.Context, string) (model.TreatmentPlan, error)
	FindByCodes(context.Context, []string) (map[string]model.TreatmentPlan, error)
}

// GateTestReader 只暴露门禁需要的材料检测查询能力。
type GateTestReader interface {
	FindByCode(context.Context, string) (model.MaterialTest, error)
	FindByCodes(context.Context, []string) (map[string]model.MaterialTest, error)
	FindVerifiedByRelatedCodes(context.Context, []string) (map[string]model.MaterialTest, error)
}

// GateLiveState 是审批列表/方案页展示用的实时门禁结论，不写库。
type GateLiveState struct {
	Verdict string
	Reason  string
}

// ApprovalGateService 封装“检测快照门禁”的全部业务规则：
// 提交待复核时冻结、批准时复核快照、读路径上批量计算实时结论。
type ApprovalGateService interface {
	// FreezeForReview 在 draft -> review 时执行；缺方案或有效检测时返回门禁错误，
	// 成功则返回待写入审批的冻结快照。
	FreezeForReview(ctx context.Context, approval *model.StageApproval, at time.Time) (model.GateSnapshot, error)
	// VerifyForApprove 在 review -> approved 时执行；快照与现状不一致时返回
	// ErrGateSnapshotMismatch，错误消息包含具体原因。
	VerifyForApprove(ctx context.Context, approval *model.StageApproval) error
	// Hydrate 为一批审批（通常是列表页）批量填充 GateLiveVerdict/GateLiveReason，
	// 已删除、草稿等无需门禁的行保持零值。
	Hydrate(ctx context.Context, approvals []model.StageApproval) []model.StageApproval
	// EvaluateLive 计算单条审批的实时门禁结论，供详情接口复用。
	EvaluateLive(ctx context.Context, approval model.StageApproval) GateLiveState
}

type approvalGateService struct {
	plans GatePlanReader
	tests GateTestReader
}

// NewApprovalGateService 装配门禁服务，参数是处理方案与材料检测仓储。
func NewApprovalGateService(plans repository.TreatmentPlanRepository, tests repository.MaterialTestRepository) ApprovalGateService {
	return &approvalGateService{plans: plans, tests: tests}
}

func normalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func (g *approvalGateService) FreezeForReview(ctx context.Context, approval *model.StageApproval, at time.Time) (model.GateSnapshot, error) {
	planCode := normalizeCode(approval.RelatedCode)
	if planCode == "" {
		return model.GateSnapshot{}, fmt.Errorf("%w: 审批 %s 未填写关联方案编码", ErrGatePlanMissing, approval.Code)
	}
	plan, err := g.plans.FindByCode(ctx, planCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.GateSnapshot{}, fmt.Errorf("%w: 关联编码 %s 未找到处理方案", ErrGatePlanMissing, planCode)
		}
		return model.GateSnapshot{}, fmt.Errorf("查询处理方案 %s: %w", planCode, err)
	}
	verified, err := g.tests.FindVerifiedByRelatedCodes(ctx, []string{plan.Code})
	if err != nil {
		return model.GateSnapshot{}, fmt.Errorf("查询已核验材料检测: %w", err)
	}
	test, ok := verified[plan.Code]
	if !ok {
		return model.GateSnapshot{}, fmt.Errorf("%w: 处理方案 %s 下没有状态为已核验的材料检测", ErrGateTestMissing, plan.Code)
	}
	return model.GateSnapshot{
		PlanCode:    plan.Code,
		TestCode:    test.Code,
		TestName:    test.Name,
		TestVersion: test.Version,
		TestStatus:  test.Status,
		TestUpdated: test.UpdatedAt,
		FrozenAt:    at.UTC(),
	}, nil
}

func (g *approvalGateService) VerifyForApprove(ctx context.Context, approval *model.StageApproval) error {
	if !approval.HasGateSnapshot() {
		return fmt.Errorf("%w: 审批 %s 缺少冻结检测快照，需退回草稿重新提交复核", ErrGateSnapshotMismatch, approval.Code)
	}
	test, err := g.tests.FindByCode(ctx, approval.GateTestCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: 冻结检测 %s 已删除或不可读", ErrGateSnapshotMismatch, approval.GateTestCode)
		}
		return fmt.Errorf("读取冻结检测 %s: %w", approval.GateTestCode, err)
	}
	if test.Status != constants.MaterialTestVerified {
		return fmt.Errorf("%w: 冻结检测 %s 已改判为 %s，不再是已核验状态", ErrGateSnapshotMismatch, test.Code, test.Status)
	}
	if test.Version != approval.GateTestVersion {
		return fmt.Errorf("%w: 冻结检测 %s 已由 v%d 换版为 v%d", ErrGateSnapshotMismatch, test.Code, approval.GateTestVersion, test.Version)
	}
	return nil
}

// EvaluateLive 按当前数据库现状重放门禁，结论用于前端提示“现在能否批准”。
// 该函数只读，绝不改变审批状态、版本或意见。
func (g *approvalGateService) EvaluateLive(ctx context.Context, approval model.StageApproval) GateLiveState {
	switch approval.Status {
	case string(constants.ApprovalStateApproved):
		if approval.HasGateSnapshot() {
			return GateLiveState{Verdict: constants.GateVerdictFrozen, Reason: "批准时已通过检测快照门禁，快照随审批冻结"}
		}
		return GateLiveState{}
	case string(constants.ApprovalStateReview):
		// 继续走快照比对
	default:
		return GateLiveState{}
	}
	if !approval.HasGateSnapshot() {
		return GateLiveState{Verdict: constants.GateVerdictSnapshotEmpty, Reason: "该待复核审批缺少冻结检测快照，需退回草稿后重新提交"}
	}
	if _, err := g.plans.FindByCode(ctx, approval.GatePlanCode); err != nil {
		return GateLiveState{Verdict: constants.GateVerdictPlanMissing, Reason: fmt.Sprintf("关联处理方案 %s 已不存在", approval.GatePlanCode)}
	}
	test, err := g.tests.FindByCode(ctx, approval.GateTestCode)
	if err != nil {
		return GateLiveState{Verdict: constants.GateVerdictTestDeleted, Reason: fmt.Sprintf("冻结检测 %s 已删除或不可读", approval.GateTestCode)}
	}
	if test.Status != constants.MaterialTestVerified {
		return GateLiveState{Verdict: constants.GateVerdictTestInvalid, Reason: fmt.Sprintf("检测 %s 已改判为 %s，不再是已核验状态", test.Code, test.Status)}
	}
	if test.Version != approval.GateTestVersion {
		return GateLiveState{Verdict: constants.GateVerdictVersionChanged, Reason: fmt.Sprintf("检测 %s 已由 v%d 换版为 v%d，需退回重新冻结", test.Code, approval.GateTestVersion, test.Version)}
	}
	return GateLiveState{Verdict: constants.GateVerdictReady, Reason: fmt.Sprintf("冻结检测 %s v%d 仍为已核验，门禁放行", test.Code, test.Version)}
}

func (g *approvalGateService) Hydrate(ctx context.Context, approvals []model.StageApproval) []model.StageApproval {
	if len(approvals) == 0 {
		return approvals
	}
	planCodes := make(map[string]struct{})
	testCodes := make(map[string]struct{})
	reviewIndexes := make([]int, 0, len(approvals))
	for index := range approvals {
		item := &approvals[index]
		if item.Status != string(constants.ApprovalStateReview) {
			continue
		}
		reviewIndexes = append(reviewIndexes, index)
		if !item.HasGateSnapshot() {
			// 门禁上线前遗留的待复核审批：没有快照可比对，直接标记为快照缺失，
			// 批准时会被 VerifyForApprove 拦截，要求退回草稿重新提交。
			item.GateLiveVerdict = constants.GateVerdictSnapshotEmpty
			item.GateLiveReason = "该待复核审批缺少冻结检测快照，需退回草稿后重新提交"
			continue
		}
		if item.GatePlanCode != "" {
			planCodes[item.GatePlanCode] = struct{}{}
		}
		if item.GateTestCode != "" {
			testCodes[item.GateTestCode] = struct{}{}
		}
	}
	if len(reviewIndexes) == 0 {
		return approvals
	}
	plans := g.bulkPlans(ctx, planCodes)
	tests := g.bulkTests(ctx, testCodes)
	for _, index := range reviewIndexes {
		item := &approvals[index]
		state := g.evaluateFromMaps(item, plans, tests)
		item.GateLiveVerdict = state.Verdict
		item.GateLiveReason = state.Reason
	}
	return approvals
}

func (g *approvalGateService) bulkPlans(ctx context.Context, codeSet map[string]struct{}) map[string]model.TreatmentPlan {
	codes := keysOf(codeSet)
	plans, err := g.plans.FindByCodes(ctx, codes)
	if err != nil {
		return map[string]model.TreatmentPlan{}
	}
	return plans
}

func (g *approvalGateService) bulkTests(ctx context.Context, codeSet map[string]struct{}) map[string]model.MaterialTest {
	codes := keysOf(codeSet)
	tests, err := g.tests.FindByCodes(ctx, codes)
	if err != nil {
		return map[string]model.MaterialTest{}
	}
	return tests
}

// evaluateFromMaps 使用批量预读的映射复刻 EvaluateLive 的判定，保证列表结论
// 与详情接口逐条计算的结论完全一致。
func (g *approvalGateService) evaluateFromMaps(item *model.StageApproval, plans map[string]model.TreatmentPlan, tests map[string]model.MaterialTest) GateLiveState {
	if !item.HasGateSnapshot() {
		return GateLiveState{Verdict: constants.GateVerdictSnapshotEmpty, Reason: "该待复核审批缺少冻结检测快照，需退回草稿后重新提交"}
	}
	if _, ok := plans[item.GatePlanCode]; !ok {
		return GateLiveState{Verdict: constants.GateVerdictPlanMissing, Reason: fmt.Sprintf("关联处理方案 %s 已不存在", item.GatePlanCode)}
	}
	test, ok := tests[item.GateTestCode]
	if !ok {
		return GateLiveState{Verdict: constants.GateVerdictTestDeleted, Reason: fmt.Sprintf("冻结检测 %s 已删除或不可读", item.GateTestCode)}
	}
	if test.Status != constants.MaterialTestVerified {
		return GateLiveState{Verdict: constants.GateVerdictTestInvalid, Reason: fmt.Sprintf("检测 %s 已改判为 %s，不再是已核验状态", test.Code, test.Status)}
	}
	if test.Version != item.GateTestVersion {
		return GateLiveState{Verdict: constants.GateVerdictVersionChanged, Reason: fmt.Sprintf("检测 %s 已由 v%d 换版为 v%d，需退回重新冻结", test.Code, item.GateTestVersion, test.Version)}
	}
	return GateLiveState{Verdict: constants.GateVerdictReady, Reason: fmt.Sprintf("冻结检测 %s v%d 仍为已核验，门禁放行", test.Code, test.Version)}
}

func keysOf(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	return out
}
