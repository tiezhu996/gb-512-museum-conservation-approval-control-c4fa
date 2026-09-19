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
	FindByCode(context.Context, string) (model.TreatmentPlan, error)
	FindByCodes(context.Context, []string) (map[string]model.TreatmentPlan, error)
	Create(context.Context, *model.TreatmentPlan) error
	Update(context.Context, uint, uint, *model.TreatmentPlan) error
	Delete(context.Context, uint) error
	CountByStatus(context.Context) (map[string]int64, error)
}

type treatmentPlanRepository struct {
	store *Store[model.TreatmentPlan]
	db    *gorm.DB
}

func NewTreatmentPlanRepository(db *gorm.DB) TreatmentPlanRepository {
	return &treatmentPlanRepository{store: NewStore[model.TreatmentPlan](db), db: db}
}

func (r *treatmentPlanRepository) List(ctx context.Context, q dto.PageQuery) (Page[model.TreatmentPlan], error) {
	return r.store.List(ctx, q)
}
func (r *treatmentPlanRepository) Get(ctx context.Context, id uint) (model.TreatmentPlan, error) {
	return r.store.Get(ctx, id)
}
func (r *treatmentPlanRepository) FindByCode(ctx context.Context, code string) (model.TreatmentPlan, error) {
	var item model.TreatmentPlan
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&item).Error
	return item, err
}

// FindByCodes 批量解析审批的关联编码，返回以方案编码为键的映射，供门禁批量预读。
func (r *treatmentPlanRepository) FindByCodes(ctx context.Context, codes []string) (map[string]model.TreatmentPlan, error) {
	items, err := r.store.FindByCodes(ctx, codes)
	if err != nil {
		return nil, err
	}
	byCode := make(map[string]model.TreatmentPlan, len(items))
	for _, item := range items {
		byCode[item.Code] = item
	}
	return byCode, nil
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
