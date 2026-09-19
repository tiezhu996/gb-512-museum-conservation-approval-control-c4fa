package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/blueship581/museum-conservation-approval-control/backend/internal/constants"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/model"
	"gorm.io/gorm"
)

// The stage-approval gate freezes a material-test snapshot when an approval
// enters review and re-validates that snapshot before approval completes.
// All gate failures leave status, version and opinion history untouched.

// freezeTestSnapshot resolves the treatment plan referenced by the approval's
// related code and captures the latest verified material test linked to that
// plan. The returned verdict describes what was frozen for later display.
func (s *stageApprovalService) freezeTestSnapshot(ctx context.Context, relatedCode string) (string, uint, string, error) {
	code := strings.ToUpper(strings.TrimSpace(relatedCode))
	if code == "" {
		return "", 0, "", fmt.Errorf("%w: 审批未填写关联编码，无法定位处理方案", ErrGatePlanMissing)
	}
	plan, err := s.plans.FindByCodeOrRelated(ctx, code)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", 0, "", fmt.Errorf("%w: 关联编码 %s 未匹配到处理方案", ErrGatePlanMissing, code)
	}
	if err != nil {
		return "", 0, "", fmt.Errorf("resolve treatment plan for gate: %w", err)
	}
	test, err := s.tests.FindLatestVerified(ctx, planLinkCodes(plan))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", 0, "", fmt.Errorf("%w: 方案 %s 没有已核验的材料检测", ErrGateTestMissing, plan.Code)
	}
	if err != nil {
		return "", 0, "", fmt.Errorf("resolve verified material test for gate: %w", err)
	}
	verdict := fmt.Sprintf("已冻结检测 %s v%d（已核验），批准前须保持一致", test.Code, test.Version)
	return test.Code, test.Version, verdict, nil
}

// verifyFrozenTest re-reads the frozen material test at approval time. Any
// drift — deletion, re-judgement away from verified, or a version bump —
// blocks the transition and the returned verdict records the concrete reason.
func (s *stageApprovalService) verifyFrozenTest(ctx context.Context, approval model.StageApproval) (string, error) {
	if approval.GateTestCode == "" {
		verdict := "门禁拦截：缺少冻结检测快照，请退回草稿重新提交复核"
		return verdict, fmt.Errorf("%w: 审批 %s 未冻结材料检测快照", ErrGateSnapshotStale, approval.Code)
	}
	test, err := s.tests.GetByCode(ctx, approval.GateTestCode)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		verdict := fmt.Sprintf("门禁拦截：冻结检测 %s 已删除或不可读", approval.GateTestCode)
		return verdict, fmt.Errorf("%w: 冻结检测 %s 已删除或不可读", ErrGateSnapshotStale, approval.GateTestCode)
	}
	if err != nil {
		return "", fmt.Errorf("re-read frozen material test: %w", err)
	}
	if test.Version != approval.GateTestVersion {
		verdict := fmt.Sprintf("门禁拦截：检测 %s 已换版 v%d→v%d", test.Code, approval.GateTestVersion, test.Version)
		return verdict, fmt.Errorf("%w: 检测 %s 版本从 v%d 变为 v%d", ErrGateSnapshotStale, test.Code, approval.GateTestVersion, test.Version)
	}
	if test.Status != string(constants.MaterialTestStateVerified) {
		verdict := fmt.Sprintf("门禁拦截：检测 %s 已改判为 %s", test.Code, test.Status)
		return verdict, fmt.Errorf("%w: 检测 %s 状态从 verified 改判为 %s", ErrGateSnapshotStale, test.Code, test.Status)
	}
	return fmt.Sprintf("门禁通过：检测 %s v%d 保持已核验", test.Code, test.Version), nil
}

// planLinkCodes collects the distinct non-empty codes a material test may use
// to reference the plan: its own code and its related code.
func planLinkCodes(plan model.TreatmentPlan) []string {
	codes := make([]string, 0, 2)
	seen := map[string]bool{}
	for _, code := range []string{plan.Code, plan.RelatedCode} {
		code = strings.TrimSpace(code)
		if code != "" && !seen[code] {
			seen[code] = true
			codes = append(codes, code)
		}
	}
	return codes
}
