package repository

import (
	"context"

	"github.com/blueship581/museum-conservation-approval-control/backend/internal/constants"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/dto"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/model"
	"gorm.io/gorm"
)

// MaterialTestRepository owns all persistence operations for 材料检测.
type MaterialTestRepository interface {
	List(context.Context, dto.PageQuery) (Page[model.MaterialTest], error)
	Get(context.Context, uint) (model.MaterialTest, error)
	GetByCode(context.Context, string) (model.MaterialTest, error)
	FindLatestVerified(context.Context, []string) (model.MaterialTest, error)
	Create(context.Context, *model.MaterialTest) error
	Update(context.Context, uint, uint, *model.MaterialTest) error
	Delete(context.Context, uint) error
	CountByStatus(context.Context) (map[string]int64, error)
}

type materialTestRepository struct {
	store *Store[model.MaterialTest]
	db    *gorm.DB
}

func NewMaterialTestRepository(db *gorm.DB) MaterialTestRepository {
	return &materialTestRepository{store: NewStore[model.MaterialTest](db), db: db}
}

func (r *materialTestRepository) List(ctx context.Context, q dto.PageQuery) (Page[model.MaterialTest], error) {
	return r.store.List(ctx, q)
}
func (r *materialTestRepository) Get(ctx context.Context, id uint) (model.MaterialTest, error) {
	return r.store.Get(ctx, id)
}
func (r *materialTestRepository) GetByCode(ctx context.Context, code string) (model.MaterialTest, error) {
	var item model.MaterialTest
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&item).Error
	return item, err
}

// FindLatestVerified returns the most recently updated verified test linked to
// any of the given plan codes; the approval gate freezes exactly this record.
func (r *materialTestRepository) FindLatestVerified(ctx context.Context, relatedCodes []string) (model.MaterialTest, error) {
	var item model.MaterialTest
	err := r.db.WithContext(ctx).
		Where("status = ?", string(constants.MaterialTestStateVerified)).
		Where("related_code IN ?", relatedCodes).
		Order("updated_at DESC, id DESC").
		First(&item).Error
	return item, err
}
func (r *materialTestRepository) Create(ctx context.Context, item *model.MaterialTest) error {
	return r.store.Create(ctx, item)
}
func (r *materialTestRepository) Update(ctx context.Context, id, version uint, item *model.MaterialTest) error {
	return r.store.Update(ctx, id, version, item)
}
func (r *materialTestRepository) Delete(ctx context.Context, id uint) error {
	return r.store.Delete(ctx, id)
}
func (r *materialTestRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	return r.store.CountByStatus(ctx)
}
