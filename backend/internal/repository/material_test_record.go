package repository

import (
	"context"

	"github.com/blueship581/museum-conservation-approval-control/backend/internal/dto"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/model"
	"gorm.io/gorm"
)

// MaterialTestRepository owns all persistence operations for 材料检测.
type MaterialTestRepository interface {
	List(context.Context, dto.PageQuery) (Page[model.MaterialTest], error)
	Get(context.Context, uint) (model.MaterialTest, error)
	Create(context.Context, *model.MaterialTest) error
	Update(context.Context, uint, uint, *model.MaterialTest) error
	Delete(context.Context, uint) error
	CountByStatus(context.Context) (map[string]int64, error)
}

type materialTestRepository struct {
	store *Store[model.MaterialTest]
}

func NewMaterialTestRepository(db *gorm.DB) MaterialTestRepository {
	return &materialTestRepository{store: NewStore[model.MaterialTest](db)}
}

func (r *materialTestRepository) List(ctx context.Context, q dto.PageQuery) (Page[model.MaterialTest], error) {
	return r.store.List(ctx, q)
}
func (r *materialTestRepository) Get(ctx context.Context, id uint) (model.MaterialTest, error) {
	return r.store.Get(ctx, id)
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
