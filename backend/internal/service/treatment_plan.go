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

type TreatmentPlanService interface {
	List(context.Context, dto.PageQuery) (repository.Page[model.TreatmentPlan], error)
	Get(context.Context, uint) (model.TreatmentPlan, error)
	Create(context.Context, dto.CreateTreatmentPlan, string, string) (model.TreatmentPlan, error)
	Update(context.Context, uint, dto.UpdateTreatmentPlan, string, string) (model.TreatmentPlan, error)
	Transition(context.Context, uint, dto.TransitionRequest, string, string) (model.TreatmentPlan, error)
	Delete(context.Context, uint, string, string) error
	StatusCounts(context.Context) (map[string]int64, error)
}

type treatmentPlanService struct {
	repository repository.TreatmentPlanRepository
	security   SecurityService
}

func NewTreatmentPlanService(repo repository.TreatmentPlanRepository, security SecurityService) TreatmentPlanService {
	return &treatmentPlanService{repository: repo, security: security}
}

func (s *treatmentPlanService) List(ctx context.Context, query dto.PageQuery) (repository.Page[model.TreatmentPlan], error) {
	return s.repository.List(ctx, query)
}

func (s *treatmentPlanService) Get(ctx context.Context, id uint) (model.TreatmentPlan, error) {
	return s.repository.Get(ctx, id)
}

func (s *treatmentPlanService) Create(ctx context.Context, input dto.CreateTreatmentPlan, actor, requestID string) (model.TreatmentPlan, error) {
	if err := validateTreatmentPlanBusinessFields(input.Code, input.Name, input.Facility, input.Owner); err != nil {
		return model.TreatmentPlan{}, err
	}
	item := model.TreatmentPlan{
		BaseModel: model.BaseModel{
			Code: strings.ToUpper(strings.TrimSpace(input.Code)), Name: strings.TrimSpace(input.Name),
			Status: model.TreatmentPlanInitialStatus, Version: 1, Description: strings.TrimSpace(input.Description),
		},
		Facility: strings.TrimSpace(input.Facility), Owner: strings.TrimSpace(input.Owner),
		Category: strings.TrimSpace(input.Category), RiskLevel: input.RiskLevel,
		MetricValue: input.MetricValue, MetricUnit: strings.TrimSpace(input.MetricUnit),
		EffectiveAt: input.EffectiveAt.UTC(), Evidence: strings.TrimSpace(input.Evidence),
		RelatedCode: strings.ToUpper(strings.TrimSpace(input.RelatedCode)),
	}
	if err := s.repository.Create(ctx, &item); err != nil {
		return model.TreatmentPlan{}, fmt.Errorf("create 处理方案: %w", err)
	}
	_ = s.security.Audit(ctx, actor, requestID, "create", "TreatmentPlan", item.ID, "", item.Status, "created 处理方案")
	return item, nil
}

func (s *treatmentPlanService) Update(ctx context.Context, id uint, input dto.UpdateTreatmentPlan, actor, requestID string) (model.TreatmentPlan, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.TreatmentPlan{}, err
	}
	if err := validateTreatmentPlanBusinessFields(current.Code, input.Name, input.Facility, input.Owner); err != nil {
		return model.TreatmentPlan{}, err
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
		return model.TreatmentPlan{}, fmt.Errorf("update 处理方案: %w", err)
	}
	_ = s.security.Audit(ctx, actor, requestID, "update", "TreatmentPlan", id, current.Status, current.Status, "updated business fields")
	return s.repository.Get(ctx, id)
}

func (s *treatmentPlanService) Transition(ctx context.Context, id uint, input dto.TransitionRequest, actor, requestID string) (model.TreatmentPlan, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.TreatmentPlan{}, err
	}
	target := strings.TrimSpace(input.Status)
	if !constants.CanTransition(constants.TreatmentPlanTransitions, current.Status, target) {
		return model.TreatmentPlan{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, current.Status, target)
	}
	before := current.Status
	current.Status = target
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, id, input.ExpectedVersion, &current); err != nil {
		return model.TreatmentPlan{}, fmt.Errorf("transition 处理方案: %w", err)
	}
	if err := s.security.Audit(ctx, actor, requestID, "transition", "TreatmentPlan", id, before, target, input.Reason); err != nil {
		return model.TreatmentPlan{}, fmt.Errorf("persist transition audit: %w", err)
	}
	return s.repository.Get(ctx, id)
}

func (s *treatmentPlanService) Delete(ctx context.Context, id uint, actor, requestID string) error {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		return err
	}
	return s.security.Audit(ctx, actor, requestID, "delete", "TreatmentPlan", id, current.Status, "deleted", "soft deleted 处理方案")
}

func (s *treatmentPlanService) StatusCounts(ctx context.Context) (map[string]int64, error) {
	return s.repository.CountByStatus(ctx)
}

func validateTreatmentPlanBusinessFields(code, name, facility, owner string) error {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(facility) == "" || strings.TrimSpace(owner) == "" {
		return ErrInvalidInput
	}
	return nil
}
