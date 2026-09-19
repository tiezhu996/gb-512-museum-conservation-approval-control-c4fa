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
	gate       ApprovalGateService
}

func NewStageApprovalService(repo repository.StageApprovalRepository, security SecurityService, gate ApprovalGateService) StageApprovalService {
	return &stageApprovalService{repository: repo, security: security, gate: gate}
}

func (s *stageApprovalService) List(ctx context.Context, query dto.PageQuery) (repository.Page[model.StageApproval], error) {
	page, err := s.repository.List(ctx, query)
	if err != nil {
		return page, err
	}
	page.Items = s.gate.Hydrate(ctx, page.Items)
	return page, nil
}

func (s *stageApprovalService) Get(ctx context.Context, id uint) (model.StageApproval, error) {
	item, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.StageApproval{}, err
	}
	live := s.gate.EvaluateLive(ctx, item)
	item.GateLiveVerdict = live.Verdict
	item.GateLiveReason = live.Reason
	return item, nil
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
	frozenAt := time.Now().UTC()
	// 检测快照门禁：进入待复核时冻结“按更新时间最新的已核验材料检测”；
	// 缺方案或无有效检测时直接拒绝，状态、版本与意见保持不变（不写库）。
	if target == string(constants.ApprovalStateReview) {
		snapshot, err := s.gate.FreezeForReview(ctx, &current, frozenAt)
		if err != nil {
			return model.StageApproval{}, err
		}
		applyGateSnapshot(&current, snapshot)
	}
	// 批准时必须仍能读到冻结检测、版本未变且状态仍为已核验；
	// 检测改判/换版/删除都会保持 review 并返回具体原因（不写库、不追加意见）。
	if target == string(constants.ApprovalStateApproved) {
		if err := s.gate.VerifyForApprove(ctx, &current); err != nil {
			return model.StageApproval{}, err
		}
	}
	before := current.Status
	current.Status = target
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = frozenAt
	opinion := &model.ApprovalOpinion{
		Version: input.ExpectedVersion + 1, Status: target, Opinion: strings.TrimSpace(input.Reason),
		Actor: actor, RequestID: requestID, CreatedAt: frozenAt,
	}
	if err := s.repository.TransitionWithOpinion(ctx, id, input.ExpectedVersion, &current, opinion); err != nil {
		return model.StageApproval{}, fmt.Errorf("transition 阶段审批: %w", err)
	}
	auditDetail := input.Reason
	if target == string(constants.ApprovalStateReview) && current.HasGateSnapshot() {
		auditDetail = fmt.Sprintf("%s | 门禁冻结检测 %s v%d（状态 %s）", input.Reason, current.GateTestCode, current.GateTestVersion, current.GateTestStatus)
	}
	if err := s.security.Audit(ctx, actor, requestID, "transition", "StageApproval", id, before, target, auditDetail); err != nil {
		return model.StageApproval{}, fmt.Errorf("persist transition audit: %w", err)
	}
	return s.Get(ctx, id)
}

// applyGateSnapshot 把冻结快照落到审批聚合的门禁字段，随乐观锁更新一并持久化。
func applyGateSnapshot(approval *model.StageApproval, snapshot model.GateSnapshot) {
	approval.GatePlanCode = snapshot.PlanCode
	approval.GateTestCode = snapshot.TestCode
	approval.GateTestName = snapshot.TestName
	approval.GateTestVersion = snapshot.TestVersion
	approval.GateTestStatus = snapshot.TestStatus
	approval.GateTestUpdated = snapshot.TestUpdated
	approval.GateFrozenAt = snapshot.FrozenAt
	approval.GateVerdict = constants.GateVerdictReady
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
