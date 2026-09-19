package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/blueship581/museum-conservation-approval-control/backend/internal/config"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/dto"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/model"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestStageApprovalAppendsImmutableOpinionVersions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.StageApproval{}, &model.ApprovalOpinion{}, &model.AuditLog{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	approvalRepository := repository.NewStageApprovalRepository(db)
	security := NewSecurityService(repository.NewSecurityRepository(db), config.Config{})
	svc := NewStageApprovalService(approvalRepository, security)
	item := model.StageApproval{
		BaseModel: model.BaseModel{Code: "SA-TEST", Name: "Test approval", Status: "draft", Version: 1},
		Facility:  "Conservation Lab", Owner: "operator", Category: "treatment", RiskLevel: "medium",
		EffectiveAt: time.Now().UTC(), Evidence: "material test attached", RelatedCode: "TP-TEST",
	}
	if err := approvalRepository.Create(context.Background(), &item); err != nil {
		t.Fatalf("create approval: %v", err)
	}

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
