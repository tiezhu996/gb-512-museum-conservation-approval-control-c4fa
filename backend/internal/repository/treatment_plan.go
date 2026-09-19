package repository

import (
	"context"

	"github.com/blueship581/museum-conservation-approval-control/backend/internal/dto"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/model"
	"gorm.io/gorm"
)

// TreatmentPlanRepository owns all persistence operations for 处理方案.
type TreatmentPlanRepository interface {
	List(context.Context, dto.PageQuery) (Page[model.TreatmentPlan], error)
	Get(context.Context, uint) (model.TreatmentPlan, error)
	Create(context.Context, *model.TreatmentPlan) error
	Update(context.Context, uint, uint, *model.TreatmentPlan) error
	Delete(context.Context, uint) error
	CountByStatus(context.Context) (map[string]int64, error)
}

type treatmentPlanRepository struct {
	store *Store[model.TreatmentPlan]
}

func NewTreatmentPlanRepository(db *gorm.DB) TreatmentPlanRepository {
	return &treatmentPlanRepository{store: NewStore[model.TreatmentPlan](db)}
}

func (r *treatmentPlanRepository) List(ctx context.Context, q dto.PageQuery) (Page[model.TreatmentPlan], error) {
	return r.store.List(ctx, q)
}
func (r *treatmentPlanRepository) Get(ctx context.Context, id uint) (model.TreatmentPlan, error) {
	return r.store.Get(ctx, id)
}
func (r *treatmentPlanRepository) Create(ctx context.Context, item *model.TreatmentPlan) error {
	return r.store.Create(ctx, item)
}
func (r *treatmentPlanRepository) Update(ctx context.Context, id, version uint, item *model.TreatmentPlan) error {
	return r.store.Update(ctx, id, version, item)
}
func (r *treatmentPlanRepository) Delete(ctx context.Context, id uint) error {
	return r.store.Delete(ctx, id)
}
func (r *treatmentPlanRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	return r.store.CountByStatus(ctx)
}
