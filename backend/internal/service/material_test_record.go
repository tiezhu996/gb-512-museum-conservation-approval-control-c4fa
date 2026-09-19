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

type MaterialTestService interface {
	List(context.Context, dto.PageQuery) (repository.Page[model.MaterialTest], error)
	Get(context.Context, uint) (model.MaterialTest, error)
	Create(context.Context, dto.CreateMaterialTest, string, string) (model.MaterialTest, error)
	Update(context.Context, uint, dto.UpdateMaterialTest, string, string) (model.MaterialTest, error)
	Transition(context.Context, uint, dto.TransitionRequest, string, string) (model.MaterialTest, error)
	Delete(context.Context, uint, string, string) error
	StatusCounts(context.Context) (map[string]int64, error)
}

type materialTestService struct {
	repository repository.MaterialTestRepository
	security   SecurityService
}

func NewMaterialTestService(repo repository.MaterialTestRepository, security SecurityService) MaterialTestService {
	return &materialTestService{repository: repo, security: security}
}

func (s *materialTestService) List(ctx context.Context, query dto.PageQuery) (repository.Page[model.MaterialTest], error) {
	return s.repository.List(ctx, query)
}

func (s *materialTestService) Get(ctx context.Context, id uint) (model.MaterialTest, error) {
	return s.repository.Get(ctx, id)
}

func (s *materialTestService) Create(ctx context.Context, input dto.CreateMaterialTest, actor, requestID string) (model.MaterialTest, error) {
	if err := validateMaterialTestBusinessFields(input.Code, input.Name, input.Facility, input.Owner); err != nil {
		return model.MaterialTest{}, err
	}
	item := model.MaterialTest{
		BaseModel: model.BaseModel{
			Code: strings.ToUpper(strings.TrimSpace(input.Code)), Name: strings.TrimSpace(input.Name),
			Status: model.MaterialTestInitialStatus, Version: 1, Description: strings.TrimSpace(input.Description),
		},
		Facility: strings.TrimSpace(input.Facility), Owner: strings.TrimSpace(input.Owner),
		Category: strings.TrimSpace(input.Category), RiskLevel: input.RiskLevel,
		MetricValue: input.MetricValue, MetricUnit: strings.TrimSpace(input.MetricUnit),
		EffectiveAt: input.EffectiveAt.UTC(), Evidence: strings.TrimSpace(input.Evidence),
		RelatedCode: strings.ToUpper(strings.TrimSpace(input.RelatedCode)),
	}
	if err := s.repository.Create(ctx, &item); err != nil {
		return model.MaterialTest{}, fmt.Errorf("create 材料检测: %w", err)
	}
	_ = s.security.Audit(ctx, actor, requestID, "create", "MaterialTest", item.ID, "", item.Status, "created 材料检测")
	return item, nil
}

func (s *materialTestService) Update(ctx context.Context, id uint, input dto.UpdateMaterialTest, actor, requestID string) (model.MaterialTest, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.MaterialTest{}, err
	}
	if err := validateMaterialTestBusinessFields(current.Code, input.Name, input.Facility, input.Owner); err != nil {
		return model.MaterialTest{}, err
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
		return model.MaterialTest{}, fmt.Errorf("update 材料检测: %w", err)
	}
	_ = s.security.Audit(ctx, actor, requestID, "update", "MaterialTest", id, current.Status, current.Status, "updated business fields")
	return s.repository.Get(ctx, id)
}

func (s *materialTestService) Transition(ctx context.Context, id uint, input dto.TransitionRequest, actor, requestID string) (model.MaterialTest, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.MaterialTest{}, err
	}
	target := strings.TrimSpace(input.Status)
	if !constants.CanTransition(constants.MaterialTestTransitions, current.Status, target) {
		return model.MaterialTest{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, current.Status, target)
	}
	before := current.Status
	current.Status = target
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, id, input.ExpectedVersion, &current); err != nil {
		return model.MaterialTest{}, fmt.Errorf("transition 材料检测: %w", err)
	}
	if err := s.security.Audit(ctx, actor, requestID, "transition", "MaterialTest", id, before, target, input.Reason); err != nil {
		return model.MaterialTest{}, fmt.Errorf("persist transition audit: %w", err)
	}
	return s.repository.Get(ctx, id)
}

func (s *materialTestService) Delete(ctx context.Context, id uint, actor, requestID string) error {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		return err
	}
	return s.security.Audit(ctx, actor, requestID, "delete", "MaterialTest", id, current.Status, "deleted", "soft deleted 材料检测")
}

func (s *materialTestService) StatusCounts(ctx context.Context) (map[string]int64, error) {
	return s.repository.CountByStatus(ctx)
}

func validateMaterialTestBusinessFields(code, name, facility, owner string) error {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(facility) == "" || strings.TrimSpace(owner) == "" {
		return ErrInvalidInput
	}
	return nil
}
