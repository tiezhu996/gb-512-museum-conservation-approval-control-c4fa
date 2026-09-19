package repository

import (
	"context"

	"github.com/blueship581/museum-conservation-approval-control/backend/internal/dto"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/model"
	"gorm.io/gorm"
)

// StageApprovalRepository owns all persistence operations for 阶段审批.
type StageApprovalRepository interface {
	List(context.Context, dto.PageQuery) (Page[model.StageApproval], error)
	Get(context.Context, uint) (model.StageApproval, error)
	Create(context.Context, *model.StageApproval) error
	Update(context.Context, uint, uint, *model.StageApproval) error
	TransitionWithOpinion(context.Context, uint, uint, *model.StageApproval, *model.ApprovalOpinion) error
	Delete(context.Context, uint) error
	CountByStatus(context.Context) (map[string]int64, error)
}

type stageApprovalRepository struct {
	store *Store[model.StageApproval]
	db    *gorm.DB
}

func NewStageApprovalRepository(db *gorm.DB) StageApprovalRepository {
	return &stageApprovalRepository{store: NewStore[model.StageApproval](db), db: db}
}

func (r *stageApprovalRepository) List(ctx context.Context, q dto.PageQuery) (Page[model.StageApproval], error) {
	page, err := r.store.List(ctx, q)
	if err != nil || len(page.Items) == 0 {
		return page, err
	}
	ids := make([]uint, 0, len(page.Items))
	for _, item := range page.Items {
		ids = append(ids, item.ID)
	}
	var opinions []model.ApprovalOpinion
	if err := r.db.WithContext(ctx).Where("stage_approval_id IN ?", ids).
		Order("version ASC").Find(&opinions).Error; err != nil {
		return Page[model.StageApproval]{}, err
	}
	byApproval := make(map[uint][]model.ApprovalOpinion)
	for _, opinion := range opinions {
		byApproval[opinion.StageApprovalID] = append(byApproval[opinion.StageApprovalID], opinion)
	}
	for index := range page.Items {
		page.Items[index].Opinions = byApproval[page.Items[index].ID]
	}
	return page, nil
}
func (r *stageApprovalRepository) Get(ctx context.Context, id uint) (model.StageApproval, error) {
	var item model.StageApproval
	err := r.db.WithContext(ctx).Preload("Opinions", func(db *gorm.DB) *gorm.DB {
		return db.Order("version ASC")
	}).First(&item, id).Error
	return item, err
}
func (r *stageApprovalRepository) Create(ctx context.Context, item *model.StageApproval) error {
	return r.store.Create(ctx, item)
}
func (r *stageApprovalRepository) Update(ctx context.Context, id, version uint, item *model.StageApproval) error {
	return r.store.Update(ctx, id, version, item)
}
func (r *stageApprovalRepository) TransitionWithOpinion(ctx context.Context, id, version uint, item *model.StageApproval, opinion *model.ApprovalOpinion) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.StageApproval{}).Where("id = ? AND version = ?", id, version).
			Select("*").Omit("id", "code", "created_at", "deleted_at", "Opinions").Updates(item)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrVersionConflict
		}
		opinion.StageApprovalID = id
		return tx.Create(opinion).Error
	})
}
func (r *stageApprovalRepository) Delete(ctx context.Context, id uint) error {
	return r.store.Delete(ctx, id)
}
func (r *stageApprovalRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	return r.store.CountByStatus(ctx)
}
