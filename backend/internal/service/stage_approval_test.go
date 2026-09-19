package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/blueship581/museum-conservation-approval-control/backend/internal/config"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/dto"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/model"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newGateTestService(t *testing.T) (StageApprovalService, repository.MaterialTestRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	// A single connection keeps the in-memory database visible to every
	// goroutine and serializes concurrent statements deterministically.
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(&model.StageApproval{}, &model.ApprovalOpinion{}, &model.AuditLog{},
		&model.TreatmentPlan{}, &model.MaterialTest{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	security := NewSecurityService(repository.NewSecurityRepository(db), config.Config{})
	planRepository := repository.NewTreatmentPlanRepository(db)
	testRepository := repository.NewMaterialTestRepository(db)
	svc := NewStageApprovalService(repository.NewStageApprovalRepository(db), planRepository, testRepository, security)
	return svc, testRepository, db
}

func seedPlanAndVerifiedTest(t *testing.T, db *gorm.DB, planCode, testCode string, testVersion uint) {
	t.Helper()
	plan := model.TreatmentPlan{
		BaseModel: model.BaseModel{Code: planCode, Name: "Gate plan", Status: "review", Version: 1},
		Facility:  "Conservation Lab", Owner: "operator", Category: "treatment", RiskLevel: "medium",
		EffectiveAt: time.Now().UTC(), RelatedCode: "REL-GATE",
	}
	if err := db.Create(&plan).Error; err != nil {
		t.Fatalf("create plan: %v", err)
	}
	test := model.MaterialTest{
		BaseModel: model.BaseModel{Code: testCode, Name: "Gate test", Status: "verified", Version: testVersion},
		Facility:  "Conservation Lab", Owner: "operator", Category: "material", RiskLevel: "low",
		EffectiveAt: time.Now().UTC(), RelatedCode: planCode,
	}
	if err := db.Create(&test).Error; err != nil {
		t.Fatalf("create material test: %v", err)
	}
}

func createDraftApproval(t *testing.T, db *gorm.DB, code, relatedCode string) model.StageApproval {
	t.Helper()
	item := model.StageApproval{
		BaseModel: model.BaseModel{Code: code, Name: "Gate approval", Status: "draft", Version: 1},
		Facility:  "Conservation Lab", Owner: "operator", Category: "treatment", RiskLevel: "medium",
		EffectiveAt: time.Now().UTC(), Evidence: "material test attached", RelatedCode: relatedCode,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create approval: %v", err)
	}
	return item
}

func TestStageApprovalAppendsImmutableOpinionVersions(t *testing.T) {
	svc, _, db := newGateTestService(t)
	seedPlanAndVerifiedTest(t, db, "TP-TEST", "MT-TEST", 1)
	item := createDraftApproval(t, db, "SA-TEST", "TP-TEST")

	review, err := svc.Transition(context.Background(), item.ID, dto.TransitionRequest{
		Status: "review", ExpectedVersion: 1, Reason: "材料检测通过，提交阶段复核",
	}, "operator", model.RoleOperator, "request-review")
	if err != nil {
		t.Fatalf("submit review: %v", err)
	}
	if review.Status != "review" || len(review.Opinions) != 1 {
		t.Fatalf("unexpected review state: %+v", review)
	}
	if review.Opinions[0].Version != 2 || review.Opinions[0].Actor != "operator" || review.Opinions[0].RequestID != "request-review" {
		t.Fatalf("review opinion did not preserve version context: %+v", review.Opinions[0])
	}
	if review.GateTestCode != "MT-TEST" || review.GateTestVersion != 1 || review.GateVerdict == "" {
		t.Fatalf("review did not freeze the material test snapshot: %+v", review)
	}

	_, err = svc.Transition(context.Background(), item.ID, dto.TransitionRequest{
		Status: "approved", ExpectedVersion: review.Version, Reason: "operator attempted approval",
	}, "operator", model.RoleOperator, "request-denied")
	if !errors.Is(err, ErrReviewerRequired) {
		t.Fatalf("expected reviewer role error, got %v", err)
	}

	approved, err := svc.Transition(context.Background(), item.ID, dto.TransitionRequest{
		Status: "approved", ExpectedVersion: review.Version, Reason: "检测证据完整，同意进入下一阶段",
	}, "reviewer", model.RoleReviewer, "request-approved")
	if err != nil {
		t.Fatalf("approve stage: %v", err)
	}
	if approved.Status != "approved" || len(approved.Opinions) != 2 {
		t.Fatalf("unexpected approved state: %+v", approved)
	}
	if approved.Opinions[0].Opinion != "材料检测通过，提交阶段复核" || approved.Opinions[1].Version != 3 ||
		approved.Opinions[1].Actor != "reviewer" || approved.Opinions[1].RequestID != "request-approved" {
		t.Fatalf("opinion history was overwritten or incomplete: %+v", approved.Opinions)
	}
	if approved.GateTestCode != "MT-TEST" || approved.GateTestVersion != 1 {
		t.Fatalf("approval lost the frozen snapshot: %+v", approved)
	}

	_, err = svc.Update(context.Background(), item.ID, dto.UpdateStageApproval{ExpectedVersion: approved.Version}, "admin", "request-update")
	if !errors.Is(err, ErrApprovalLocked) {
		t.Fatalf("expected immutable approval error, got %v", err)
	}
}

func TestStageApprovalGateBlocksReviewWithoutPlanOrVerifiedTest(t *testing.T) {
	svc, _, db := newGateTestService(t)

	orphan := createDraftApproval(t, db, "SA-NO-PLAN", "TP-MISSING")
	_, err := svc.Transition(context.Background(), orphan.ID, dto.TransitionRequest{
		Status: "review", ExpectedVersion: 1, Reason: "提交复核但方案不存在",
	}, "operator", model.RoleOperator, "request-no-plan")
	if !errors.Is(err, ErrGatePlanMissing) {
		t.Fatalf("expected missing plan gate error, got %v", err)
	}
	unchanged, getErr := svc.Get(context.Background(), orphan.ID)
	if getErr != nil || unchanged.Status != "draft" || unchanged.Version != 1 || len(unchanged.Opinions) != 0 {
		t.Fatalf("rejected review changed status or opinions: %+v, err=%v", unchanged, getErr)
	}

	seedPlanAndVerifiedTest(t, db, "TP-GATE", "MT-GATE", 1)
	if err := db.Model(&model.MaterialTest{}).Where("code = ?", "MT-GATE").
		Update("status", "running").Error; err != nil {
		t.Fatalf("downgrade material test: %v", err)
	}
	unverified := createDraftApproval(t, db, "SA-NO-TEST", "TP-GATE")
	_, err = svc.Transition(context.Background(), unverified.ID, dto.TransitionRequest{
		Status: "review", ExpectedVersion: 1, Reason: "提交复核但检测未核验",
	}, "operator", model.RoleOperator, "request-no-test")
	if !errors.Is(err, ErrGateTestMissing) {
		t.Fatalf("expected missing verified test gate error, got %v", err)
	}
	unchanged, getErr = svc.Get(context.Background(), unverified.ID)
	if getErr != nil || unchanged.Status != "draft" || unchanged.Version != 1 || len(unchanged.Opinions) != 0 {
		t.Fatalf("rejected review changed status or opinions: %+v, err=%v", unchanged, getErr)
	}
}

func TestStageApprovalGateBlocksApproveWhenTestDrifts(t *testing.T) {
	svc, testRepository, db := newGateTestService(t)
	seedPlanAndVerifiedTest(t, db, "TP-DRIFT", "MT-DRIFT", 1)
	item := createDraftApproval(t, db, "SA-DRIFT", "TP-DRIFT")

	review, err := svc.Transition(context.Background(), item.ID, dto.TransitionRequest{
		Status: "review", ExpectedVersion: 1, Reason: "冻结检测快照进入复核",
	}, "operator", model.RoleOperator, "request-review")
	if err != nil {
		t.Fatalf("submit review: %v", err)
	}
	if review.GateTestCode != "MT-DRIFT" || review.GateTestVersion != 1 {
		t.Fatalf("snapshot was not frozen: %+v", review)
	}

	// 改判：检测从 verified 变为 invalid，批准必须被拦截并给出具体原因。
	frozen, err := testRepository.GetByCode(context.Background(), "MT-DRIFT")
	if err != nil {
		t.Fatalf("load frozen test: %v", err)
	}
	frozen.Status = "invalid"
	frozen.UpdatedAt = time.Now().UTC()
	if err := testRepository.Update(context.Background(), frozen.ID, frozen.Version, &frozen); err != nil {
		t.Fatalf("re-judge frozen test: %v", err)
	}
	_, err = svc.Transition(context.Background(), item.ID, dto.TransitionRequest{
		Status: "approved", ExpectedVersion: review.Version, Reason: "尝试批准已改判的检测",
	}, "reviewer", model.RoleReviewer, "request-blocked-status")
	if !errors.Is(err, ErrGateSnapshotStale) {
		t.Fatalf("expected stale snapshot gate error, got %v", err)
	}
	blocked, getErr := svc.Get(context.Background(), item.ID)
	if getErr != nil || blocked.Status != "review" || blocked.Version != review.Version || len(blocked.Opinions) != 1 {
		t.Fatalf("blocked approval changed review state: %+v, err=%v", blocked, getErr)
	}
	if blocked.GateVerdict == "" || blocked.GateVerdict == review.GateVerdict {
		t.Fatalf("blocked approval did not record the gate verdict: %+v", blocked)
	}

	// 换版：检测恢复 verified 但版本号变化，批准仍须被拦截。
	frozen, err = testRepository.GetByCode(context.Background(), "MT-DRIFT")
	if err != nil {
		t.Fatalf("reload frozen test: %v", err)
	}
	frozen.Status = "verified"
	frozen.Version++
	frozen.UpdatedAt = time.Now().UTC()
	if err := testRepository.Update(context.Background(), frozen.ID, frozen.Version-1, &frozen); err != nil {
		t.Fatalf("re-version frozen test: %v", err)
	}
	_, err = svc.Transition(context.Background(), item.ID, dto.TransitionRequest{
		Status: "approved", ExpectedVersion: review.Version, Reason: "尝试批准已换版的检测",
	}, "reviewer", model.RoleReviewer, "request-blocked-version")
	if !errors.Is(err, ErrGateSnapshotStale) {
		t.Fatalf("expected version drift gate error, got %v", err)
	}
	blocked, getErr = svc.Get(context.Background(), item.ID)
	if getErr != nil || blocked.Status != "review" || len(blocked.Opinions) != 1 {
		t.Fatalf("version drift changed review state: %+v, err=%v", blocked, getErr)
	}
}

func TestStageApprovalApproveCompletesOnlyOnce(t *testing.T) {
	svc, _, db := newGateTestService(t)
	seedPlanAndVerifiedTest(t, db, "TP-ONCE", "MT-ONCE", 1)
	item := createDraftApproval(t, db, "SA-ONCE", "TP-ONCE")

	review, err := svc.Transition(context.Background(), item.ID, dto.TransitionRequest{
		Status: "review", ExpectedVersion: 1, Reason: "冻结检测快照进入复核",
	}, "operator", model.RoleOperator, "request-review")
	if err != nil {
		t.Fatalf("submit review: %v", err)
	}

	// 并发批准：两个请求持有同一版本号，乐观锁保证只有一次完成。
	var wait sync.WaitGroup
	outcomes := make([]error, 2)
	for index := range outcomes {
		wait.Add(1)
		go func(slot int) {
			defer wait.Done()
			_, outcomes[slot] = svc.Transition(context.Background(), item.ID, dto.TransitionRequest{
				Status: "approved", ExpectedVersion: review.Version, Reason: fmt.Sprintf("并发批准请求 %d", slot),
			}, "reviewer", model.RoleReviewer, fmt.Sprintf("request-concurrent-%d", slot))
		}(index)
	}
	wait.Wait()
	succeeded, blocked := 0, 0
	for _, outcome := range outcomes {
		switch {
		case outcome == nil:
			succeeded++
		case errors.Is(outcome, repository.ErrVersionConflict), errors.Is(outcome, ErrInvalidTransition):
			// 乐观锁冲突或状态机拒绝都证明失败方没有重复完成批准。
			blocked++
		default:
			t.Fatalf("unexpected concurrent outcome: %v", outcome)
		}
	}
	if succeeded != 1 || blocked != 1 {
		t.Fatalf("concurrent approvals must complete exactly once, got %d success and %d blocked", succeeded, blocked)
	}

	// 重复批准：状态机拒绝 approved -> approved，意见历史不会追加。
	_, err = svc.Transition(context.Background(), item.ID, dto.TransitionRequest{
		Status: "approved", ExpectedVersion: review.Version + 1, Reason: "重复提交批准",
	}, "reviewer", model.RoleReviewer, "request-duplicate")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected duplicate approval to be rejected, got %v", err)
	}
	final, getErr := svc.Get(context.Background(), item.ID)
	if getErr != nil || final.Status != "approved" || len(final.Opinions) != 2 {
		t.Fatalf("duplicate approval mutated the record: %+v, err=%v", final, getErr)
	}
}
