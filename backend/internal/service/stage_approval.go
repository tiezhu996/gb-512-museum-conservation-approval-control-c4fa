package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/blueship581/museum-conservation-approval-control/backend/internal/constants"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/dto"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/model"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/repository"
)

type StageApprovalService interface {
	List(context.Context, dto.PageQuery) (repository.Page[model.StageApproval], error)
	Get(context.Context, uint) (model.StageApproval, error)
	Create(context.Context, dto.CreateStageApproval, string, string) (model.StageApproval, error)
	Update(context.Context, uint, dto.UpdateStageApproval, string, string) (model.StageApproval, error)
	Transition(context.Context, uint, dto.TransitionRequest, string, string, string) (model.StageApproval, error)
	Delete(context.Context, uint, string, string) error
	StatusCounts(context.Context) (map[string]int64, error)
}

type stageApprovalService struct {
	repository repository.StageApprovalRepository
	security   SecurityService
}

func NewStageApprovalService(repo repository.StageApprovalRepository, security SecurityService) StageApprovalService {
	return &stageApprovalService{repository: repo, security: security}
}

func (s *stageApprovalService) List(ctx context.Context, query dto.PageQuery) (repository.Page[model.StageApproval], error) {
	return s.repository.List(ctx, query)
}

func (s *stageApprovalService) Get(ctx context.Context, id uint) (model.StageApproval, error) {
	return s.repository.Get(ctx, id)
}

func (s *stageApprovalService) Create(ctx context.Context, input dto.CreateStageApproval, actor, requestID string) (model.StageApproval, error) {
	if err := validateStageApprovalBusinessFields(input.Code, input.Name, input.Facility, input.Owner); err != nil {
		return model.StageApproval{}, err
	}
	item := model.StageApproval{
		BaseModel: model.BaseModel{
			Code: strings.ToUpper(strings.TrimSpace(input.Code)), Name: strings.TrimSpace(input.Name),
			Status: model.StageApprovalInitialStatus, Version: 1, Description: strings.TrimSpace(input.Description),
		},
		Facility: strings.TrimSpace(input.Facility), Owner: strings.TrimSpace(input.Owner),
		Category: strings.TrimSpace(input.Category), RiskLevel: input.RiskLevel,
		MetricValue: input.MetricValue, MetricUnit: strings.TrimSpace(input.MetricUnit),
		EffectiveAt: input.EffectiveAt.UTC(), Evidence: strings.TrimSpace(input.Evidence),
		RelatedCode: strings.ToUpper(strings.TrimSpace(input.RelatedCode)),
	}
	if err := s.repository.Create(ctx, &item); err != nil {
		return model.StageApproval{}, fmt.Errorf("create 阶段审批: %w", err)
	}
	_ = s.security.Audit(ctx, actor, requestID, "create", "StageApproval", item.ID, "", item.Status, "created 阶段审批")
	return item, nil
}

func (s *stageApprovalService) Update(ctx context.Context, id uint, input dto.UpdateStageApproval, actor, requestID string) (model.StageApproval, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.StageApproval{}, err
	}
	if current.Status != string(constants.ApprovalStateDraft) {
		return model.StageApproval{}, ErrApprovalLocked
	}
	if err := validateStageApprovalBusinessFields(current.Code, input.Name, input.Facility, input.Owner); err != nil {
		return model.StageApproval{}, err
	}
	current.Name = strings.TrimSpace(input.Name)
	current.Description = strings.TrimSpace(input.Description)
	current.Facility = strings.TrimSpace(input.Facility)
	current.Owner = strings.TrimSpace(input.Owner)
	current.Category = strings.TrimSpace(input.Category)
	current.RiskLevel = input.RiskLevel
	current.MetricValue = input.MetricValue
	current.MetricUnit = strings.TrimSpace(input.MetricUnit)
	current.EffectiveAt = input.EffectiveAt.UTC()
	current.Evidence = strings.TrimSpace(input.Evidence)
	current.RelatedCode = strings.ToUpper(strings.TrimSpace(input.RelatedCode))
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, id, input.ExpectedVersion, &current); err != nil {
		return model.StageApproval{}, fmt.Errorf("update 阶段审批: %w", err)
	}
	_ = s.security.Audit(ctx, actor, requestID, "update", "StageApproval", id, current.Status, current.Status, "updated business fields")
	return s.repository.Get(ctx, id)
}

func (s *stageApprovalService) Transition(ctx context.Context, id uint, input dto.TransitionRequest, actor, role, requestID string) (model.StageApproval, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.StageApproval{}, err
	}
	target := strings.TrimSpace(input.Status)
	if !constants.CanTransition(constants.StageApprovalTransitions, current.Status, target) {
		return model.StageApproval{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, current.Status, target)
	}
	if (target == string(constants.ApprovalStateApproved) || target == string(constants.ApprovalStateRejected)) &&
		role != model.RoleReviewer && role != model.RoleAdmin {
		return model.StageApproval{}, ErrReviewerRequired
	}
	before := current.Status
	current.Status = target
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	opinion := &model.ApprovalOpinion{
		Version: input.ExpectedVersion + 1, Status: target, Opinion: strings.TrimSpace(input.Reason),
		Actor: actor, RequestID: requestID, CreatedAt: current.UpdatedAt,
	}
	if err := s.repository.TransitionWithOpinion(ctx, id, input.ExpectedVersion, &current, opinion); err != nil {
		return model.StageApproval{}, fmt.Errorf("transition 阶段审批: %w", err)
	}
	if err := s.security.Audit(ctx, actor, requestID, "transition", "StageApproval", id, before, target, input.Reason); err != nil {
		return model.StageApproval{}, fmt.Errorf("persist transition audit: %w", err)
	}
	return s.repository.Get(ctx, id)
}

func (s *stageApprovalService) Delete(ctx context.Context, id uint, actor, requestID string) error {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		return err
	}
	return s.security.Audit(ctx, actor, requestID, "delete", "StageApproval", id, current.Status, "deleted", "soft deleted 阶段审批")
}

func (s *stageApprovalService) StatusCounts(ctx context.Context) (map[string]int64, error) {
	return s.repository.CountByStatus(ctx)
}

func validateStageApprovalBusinessFields(code, name, facility, owner string) error {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(facility) == "" || strings.TrimSpace(owner) == "" {
		return ErrInvalidInput
	}
	return nil
}
