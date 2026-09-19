package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/blueship581/museum-conservation-approval-control/backend/internal/config"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/constants"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/dto"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/model"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newGateTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.Artifact{}, &model.TreatmentPlan{}, &model.MaterialTest{},
		&model.StageApproval{}, &model.ApprovalOpinion{}, &model.AuditLog{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	return db
}

func newGateFixture(t *testing.T) (StageApprovalService, *gorm.DB, repository.TreatmentPlanRepository, repository.MaterialTestRepository, repository.StageApprovalRepository) {
	t.Helper()
	db := newGateTestDB(t)
	approvalRepository := repository.NewStageApprovalRepository(db)
	planRepository := repository.NewTreatmentPlanRepository(db)
	testRepository := repository.NewMaterialTestRepository(db)
	security := NewSecurityService(repository.NewSecurityRepository(db), config.Config{})
	gate := NewApprovalGateService(planRepository, testRepository)
	svc := NewStageApprovalService(approvalRepository, security, gate)
	return svc, db, planRepository, testRepository, approvalRepository
}

func seedPlanAndTest(t *testing.T, db *gorm.DB, planCode, testCode string, testStatus string, testVersion uint, updatedAt time.Time) {
	t.Helper()
	seedPlanOnce(t, db, planCode)
	seedTest(t, db, planCode, testCode, testStatus, testVersion, updatedAt)
}

func seedPlanOnce(t *testing.T, db *gorm.DB, planCode string) {
	t.Helper()
	var count int64
	if err := db.Model(&model.TreatmentPlan{}).Where("code = ?", planCode).Count(&count).Error; err != nil {
		t.Fatalf("count plan: %v", err)
	}
	if count > 0 {
		return
	}
	now := time.Now().UTC()
	plan := model.TreatmentPlan{
		BaseModel: model.BaseModel{Code: planCode, Name: "Plan " + planCode, Status: "review", Version: 1, CreatedAt: now, UpdatedAt: now},
		Facility:  "Lab", Owner: "planner", Category: "treatment", RiskLevel: "medium",
		EffectiveAt: now, RelatedCode: "REL-" + planCode,
	}
	if err := db.Create(&plan).Error; err != nil {
		t.Fatalf("create plan: %v", err)
	}
}

func seedTest(t *testing.T, db *gorm.DB, planCode, testCode, testStatus string, testVersion uint, updatedAt time.Time) {
	t.Helper()
	test := model.MaterialTest{
		BaseModel: model.BaseModel{Code: testCode, Name: "Test " + testCode, Status: testStatus, Version: testVersion,
			CreatedAt: updatedAt.Add(-time.Hour), UpdatedAt: updatedAt},
		Facility: "Lab", Owner: "tester", Category: "material", RiskLevel: "medium",
		EffectiveAt: updatedAt, RelatedCode: planCode,
	}
	if err := db.Create(&test).Error; err != nil {
		t.Fatalf("create material test: %v", err)
	}
}

func createDraftApproval(t *testing.T, db *gorm.DB, code, relatedCode string) model.StageApproval {
	t.Helper()
	now := time.Now().UTC()
	item := model.StageApproval{
		BaseModel: model.BaseModel{Code: code, Name: "Approval " + code, Status: "draft", Version: 1, CreatedAt: now, UpdatedAt: now},
		Facility:  "Conservation Lab", Owner: "operator", Category: "treatment", RiskLevel: "medium",
		EffectiveAt: now, Evidence: "material test attached", RelatedCode: relatedCode,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create approval: %v", err)
	}
	return item
}

// TestStageApprovalAppendsImmutableOpinionVersions 验证原有意见版本链不被门禁破坏。
func TestStageApprovalAppendsImmutableOpinionVersions(t *testing.T) {
	svc, db, _, _, _ := newGateFixture(t)
	seedPlanAndTest(t, db, "TP-TEST", "MT-TEST", "verified", 1, time.Now().UTC().Add(-2*time.Hour))
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
	if review.GateTestCode != "MT-TEST" || review.GateTestVersion != 1 || review.GateVerdict != constants.GateVerdictReady {
		t.Fatalf("review transition did not freeze verified test snapshot: %+v", review)
	}
	if review.Opinions[0].Version != 2 || review.Opinions[0].Actor != "operator" || review.Opinions[0].RequestID != "request-review" {
		t.Fatalf("review opinion did not preserve version context: %+v", review.Opinions[0])
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

	_, err = svc.Update(context.Background(), item.ID, dto.UpdateStageApproval{ExpectedVersion: approved.Version}, "admin", "request-update")
	if !errors.Is(err, ErrApprovalLocked) {
		t.Fatalf("expected immutable approval error, got %v", err)
	}
}

// TestGateRejectsReviewWithoutPlanOrVerifiedTest 验证缺方案或缺有效检测时，
// 提交待复核被拒绝且状态/版本/意见保持不变。
func TestGateRejectsReviewWithoutPlanOrVerifiedTest(t *testing.T) {
	svc, db, _, _, approvalRepository := newGateFixture(t)

	// 场景一：关联方案不存在。
	missingPlan := createDraftApproval(t, db, "SA-NO-PLAN", "TP-GHOST")
	_, err := svc.Transition(context.Background(), missingPlan.ID, dto.TransitionRequest{
		Status: "review", ExpectedVersion: 1, Reason: "尝试提交无方案的复核",
	}, "operator", model.RoleOperator, "req-no-plan")
	if !errors.Is(err, ErrGatePlanMissing) {
		t.Fatalf("expected ErrGatePlanMissing, got %v", err)
	}
	stored, err := approvalRepository.Get(context.Background(), missingPlan.ID)
	if err != nil {
		t.Fatalf("reload approval: %v", err)
	}
	if stored.Status != "draft" || stored.Version != 1 || len(stored.Opinions) != 0 || stored.HasGateSnapshot() {
		t.Fatalf("approval must remain untouched after plan gate rejection: %+v", stored)
	}

	// 场景二：方案存在，但检测未核验。
	seedPlanAndTest(t, db, "TP-RUNNING", "MT-RUNNING", "running", 1, time.Now().UTC())
	running := createDraftApproval(t, db, "SA-RUNNING", "TP-RUNNING")
	_, err = svc.Transition(context.Background(), running.ID, dto.TransitionRequest{
		Status: "review", ExpectedVersion: 1, Reason: "检测还在运行却提交复核",
	}, "operator", model.RoleOperator, "req-no-test")
	if !errors.Is(err, ErrGateTestMissing) {
		t.Fatalf("expected ErrGateTestMissing, got %v", err)
	}
	stored, _ = approvalRepository.Get(context.Background(), running.ID)
	if stored.Status != "draft" || stored.Version != 1 || len(stored.Opinions) != 0 || stored.HasGateSnapshot() {
		t.Fatalf("approval must remain untouched after missing-test rejection: %+v", stored)
	}
}

// TestGateFreezesLatestVerifiedTestByUpdatedAt 验证多个已核验检测时，
// 冻结按更新时间最新的那一条。
func TestGateFreezesLatestVerifiedTestByUpdatedAt(t *testing.T) {
	svc, db, _, _, _ := newGateFixture(t)
	base := time.Now().UTC().Add(-time.Hour)
	seedPlanAndTest(t, db, "TP-LATEST", "MT-OLD", "verified", 1, base)
	seedPlanAndTest(t, db, "TP-LATEST", "MT-NEW", "verified", 3, base.Add(30*time.Minute))
	approval := createDraftApproval(t, db, "SA-LATEST", "TP-LATEST")

	review, err := svc.Transition(context.Background(), approval.ID, dto.TransitionRequest{
		Status: "review", ExpectedVersion: 1, Reason: "提交复核并冻结最新检测",
	}, "operator", model.RoleOperator, "req-freeze")
	if err != nil {
		t.Fatalf("freeze snapshot: %v", err)
	}
	if review.GateTestCode != "MT-NEW" || review.GateTestVersion != 3 {
		t.Fatalf("expected newest verified test MT-NEW v3 frozen, got %s v%d", review.GateTestCode, review.GateTestVersion)
	}
}

// TestGateKeepsReviewWhenTestRejudgedOrVersioned 验证批准时检测改判或换版，
// 审批保持待复核并返回具体原因，且不追加意见、不升版本。
func TestGateKeepsReviewWhenTestRejudgedOrVersioned(t *testing.T) {
	t.Run("test rejudged to invalid", func(t *testing.T) {
		svc, db, _, testRepo, approvalRepo := newGateFixture(t)
		freezeTime := time.Now().UTC().Add(-2 * time.Hour)
		seedPlanAndTest(t, db, "TP-INVALID", "MT-INVALID", "verified", 1, freezeTime)
		approval := createDraftApproval(t, db, "SA-INVALID", "TP-INVALID")
		review := mustTransition(t, svc, approval.ID, 1, "review", "reviewer", model.RoleOperator)

		// 检测被改判：verified -> invalid（版本升到 2）。
		test, _ := testRepo.Get(context.Background(), mustTestID(t, db, "MT-INVALID"))
		test.Status = "invalid"
		test.Version = 2
		if err := testRepo.Update(context.Background(), test.ID, 1, &test); err != nil {
			t.Fatalf("rejudge test: %v", err)
		}

		_, err := svc.Transition(context.Background(), review.ID, dto.TransitionRequest{
			Status: "approved", ExpectedVersion: review.Version, Reason: "批准时检测已改判",
		}, "reviewer", model.RoleReviewer, "req-invalid")
		if !errors.Is(err, ErrGateSnapshotMismatch) {
			t.Fatalf("expected snapshot mismatch, got %v", err)
		}
		stored, _ := approvalRepo.Get(context.Background(), review.ID)
		assertStillReview(t, stored, review.Version)
	})

	t.Run("test version changed", func(t *testing.T) {
		svc, db, _, testRepo, approvalRepo := newGateFixture(t)
		seedPlanAndTest(t, db, "TP-BUMP", "MT-BUMP", "verified", 1, time.Now().UTC().Add(-time.Hour))
		approval := createDraftApproval(t, db, "SA-BUMP", "TP-BUMP")
		review := mustTransition(t, svc, approval.ID, 1, "review", "reviewer", model.RoleOperator)

		// 检测换版但仍为 verified：状态满足，版本漂移，必须拦截。
		test, _ := testRepo.Get(context.Background(), mustTestID(t, db, "MT-BUMP"))
		test.Name = "MT-BUMP renewed"
		test.Version = 2
		if err := testRepo.Update(context.Background(), test.ID, 1, &test); err != nil {
			t.Fatalf("renew test: %v", err)
		}

		_, err := svc.Transition(context.Background(), review.ID, dto.TransitionRequest{
			Status: "approved", ExpectedVersion: review.Version, Reason: "批准时检测已换版",
		}, "admin", model.RoleAdmin, "req-bump")
		if !errors.Is(err, ErrGateSnapshotMismatch) {
			t.Fatalf("expected snapshot mismatch for version drift, got %v", err)
		}
		stored, _ := approvalRepo.Get(context.Background(), review.ID)
		assertStillReview(t, stored, review.Version)

		// 列表实时结论应为 version_changed。
		page, err := svc.List(context.Background(), dto.PageQuery{Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("list approvals: %v", err)
		}
		var found *model.StageApproval
		for index := range page.Items {
			if page.Items[index].ID == review.ID {
				found = &page.Items[index]
			}
		}
		if found == nil || found.GateLiveVerdict != constants.GateVerdictVersionChanged {
			t.Fatalf("expected live verdict %s, got %+v", constants.GateVerdictVersionChanged, found)
		}
	})
}

// TestGateApprovalCompletesOnceUnderDuplicateRequests 验证重复/并发批准只完成一次：
// 两个请求使用相同期望版本，只有一个能提交成功，另一个命中乐观锁冲突。
func TestGateApprovalCompletesOnceUnderDuplicateRequests(t *testing.T) {
	svc, db, _, _, approvalRepo := newGateFixture(t)
	seedPlanAndTest(t, db, "TP-ONCE", "MT-ONCE", "verified", 1, time.Now().UTC().Add(-time.Hour))
	approval := createDraftApproval(t, db, "SA-ONCE", "TP-ONCE")
	review := mustTransition(t, svc, approval.ID, 1, "review", "reviewer", model.RoleOperator)

	first, err := svc.Transition(context.Background(), review.ID, dto.TransitionRequest{
		Status: "approved", ExpectedVersion: review.Version, Reason: "首次批准",
	}, "reviewer", model.RoleReviewer, "req-first")
	if err != nil {
		t.Fatalf("first approval: %v", err)
	}
	if first.Status != "approved" || first.Version != review.Version+1 {
		t.Fatalf("unexpected first approval: %+v", first)
	}

	// 重复请求：旧期望版本必然失败，状态机 approved 无后继迁移，且无第三条意见。
	_, err = svc.Transition(context.Background(), review.ID, dto.TransitionRequest{
		Status: "approved", ExpectedVersion: review.Version, Reason: "重复批准",
	}, "reviewer", model.RoleReviewer, "req-dup")
	if !errors.Is(err, ErrInvalidTransition) && !errors.Is(err, repository.ErrVersionConflict) {
		t.Fatalf("duplicate approval must be rejected by state machine or optimistic lock, got %v", err)
	}
	stored, _ := approvalRepo.Get(context.Background(), review.ID)
	if stored.Status != "approved" || stored.Version != review.Version+1 || len(stored.Opinions) != 2 {
		t.Fatalf("approval completed more than once: %+v", stored)
	}
}

func mustTransition(t *testing.T, svc StageApprovalService, id, expectedVersion uint, target, actor, role string) model.StageApproval {
	t.Helper()
	item, err := svc.Transition(context.Background(), id, dto.TransitionRequest{
		Status: target, ExpectedVersion: expectedVersion, Reason: "transition " + target,
	}, actor, role, "req-"+target)
	if err != nil {
		t.Fatalf("transition to %s: %v", target, err)
	}
	return item
}

func mustTestID(t *testing.T, db *gorm.DB, code string) uint {
	t.Helper()
	var test model.MaterialTest
	if err := db.Where("code = ?", code).First(&test).Error; err != nil {
		t.Fatalf("load test %s: %v", code, err)
	}
	return test.ID
}

func assertStillReview(t *testing.T, stored model.StageApproval, expectedVersion uint) {
	t.Helper()
	if stored.Status != "review" {
		t.Fatalf("status must remain review, got %s", stored.Status)
	}
	if stored.Version != expectedVersion {
		t.Fatalf("version must stay at %d, got %d", expectedVersion, stored.Version)
	}
	if len(stored.Opinions) != 1 {
		t.Fatalf("no opinion may be appended on blocked approval, got %d", len(stored.Opinions))
	}
}
