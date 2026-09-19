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

type ArtifactService interface {
	List(context.Context, dto.PageQuery) (repository.Page[model.Artifact], error)
	Get(context.Context, uint) (model.Artifact, error)
	Create(context.Context, dto.CreateArtifact, string, string) (model.Artifact, error)
	Update(context.Context, uint, dto.UpdateArtifact, string, string) (model.Artifact, error)
	Transition(context.Context, uint, dto.TransitionRequest, string, string) (model.Artifact, error)
	Delete(context.Context, uint, string, string) error
	StatusCounts(context.Context) (map[string]int64, error)
}

type artifactService struct {
	repository repository.ArtifactRepository
	security   SecurityService
}

func NewArtifactService(repo repository.ArtifactRepository, security SecurityService) ArtifactService {
	return &artifactService{repository: repo, security: security}
}

func (s *artifactService) List(ctx context.Context, query dto.PageQuery) (repository.Page[model.Artifact], error) {
	return s.repository.List(ctx, query)
}

func (s *artifactService) Get(ctx context.Context, id uint) (model.Artifact, error) {
	return s.repository.Get(ctx, id)
}

func (s *artifactService) Create(ctx context.Context, input dto.CreateArtifact, actor, requestID string) (model.Artifact, error) {
	if err := validateArtifactBusinessFields(input.Code, input.Name, input.Facility, input.Owner); err != nil {
		return model.Artifact{}, err
	}
	item := model.Artifact{
		BaseModel: model.BaseModel{
			Code: strings.ToUpper(strings.TrimSpace(input.Code)), Name: strings.TrimSpace(input.Name),
			Status: model.ArtifactInitialStatus, Version: 1, Description: strings.TrimSpace(input.Description),
		},
		Facility: strings.TrimSpace(input.Facility), Owner: strings.TrimSpace(input.Owner),
		Category: strings.TrimSpace(input.Category), RiskLevel: input.RiskLevel,
		MetricValue: input.MetricValue, MetricUnit: strings.TrimSpace(input.MetricUnit),
		EffectiveAt: input.EffectiveAt.UTC(), Evidence: strings.TrimSpace(input.Evidence),
		RelatedCode: strings.ToUpper(strings.TrimSpace(input.RelatedCode)),
	}
	if err := s.repository.Create(ctx, &item); err != nil {
		return model.Artifact{}, fmt.Errorf("create 文物: %w", err)
	}
	_ = s.security.Audit(ctx, actor, requestID, "create", "Artifact", item.ID, "", item.Status, "created 文物")
	return item, nil
}

func (s *artifactService) Update(ctx context.Context, id uint, input dto.UpdateArtifact, actor, requestID string) (model.Artifact, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.Artifact{}, err
	}
	if err := validateArtifactBusinessFields(current.Code, input.Name, input.Facility, input.Owner); err != nil {
		return model.Artifact{}, err
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
		return model.Artifact{}, fmt.Errorf("update 文物: %w", err)
	}
	_ = s.security.Audit(ctx, actor, requestID, "update", "Artifact", id, current.Status, current.Status, "updated business fields")
	return s.repository.Get(ctx, id)
}

func (s *artifactService) Transition(ctx context.Context, id uint, input dto.TransitionRequest, actor, requestID string) (model.Artifact, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.Artifact{}, err
	}
	target := strings.TrimSpace(input.Status)
	if !constants.CanTransition(constants.ArtifactTransitions, current.Status, target) {
		return model.Artifact{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, current.Status, target)
	}
	before := current.Status
	current.Status = target
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, id, input.ExpectedVersion, &current); err != nil {
		return model.Artifact{}, fmt.Errorf("transition 文物: %w", err)
	}
	if err := s.security.Audit(ctx, actor, requestID, "transition", "Artifact", id, before, target, input.Reason); err != nil {
		return model.Artifact{}, fmt.Errorf("persist transition audit: %w", err)
	}
	return s.repository.Get(ctx, id)
}

func (s *artifactService) Delete(ctx context.Context, id uint, actor, requestID string) error {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		return err
	}
	return s.security.Audit(ctx, actor, requestID, "delete", "Artifact", id, current.Status, "deleted", "soft deleted 文物")
}

func (s *artifactService) StatusCounts(ctx context.Context) (map[string]int64, error) {
	return s.repository.CountByStatus(ctx)
}

func validateArtifactBusinessFields(code, name, facility, owner string) error {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(facility) == "" || strings.TrimSpace(owner) == "" {
		return ErrInvalidInput
	}
	return nil
}
