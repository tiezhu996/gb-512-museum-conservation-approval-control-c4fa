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
	FindByCode(context.Context, string) (model.MaterialTest, error)
	FindByCodes(context.Context, []string) (map[string]model.MaterialTest, error)
	FindVerifiedByRelatedCodes(context.Context, []string) (map[string]model.MaterialTest, error)
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
func (r *materialTestRepository) FindByCode(ctx context.Context, code string) (model.MaterialTest, error) {
	var item model.MaterialTest
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&item).Error
	return item, err
}

// FindByCodes 批量加载冻结检测的现状，键是检测编码。
func (r *materialTestRepository) FindByCodes(ctx context.Context, codes []string) (map[string]model.MaterialTest, error) {
	items, err := r.store.FindByCodes(ctx, codes)
	if err != nil {
		return nil, err
	}
	byCode := make(map[string]model.MaterialTest, len(items))
	for _, item := range items {
		byCode[item.Code] = item
	}
	return byCode, nil
}

// FindVerifiedByRelatedCodes 返回每个关联编码（即处理方案编码）下
// “按更新时间最新的已核验材料检测”。updated_at 相同时以 id 最大者为准，
// 保证冻结时刻的选择在 MySQL 与 SQLite 上结果一致、可重复。
func (r *materialTestRepository) FindVerifiedByRelatedCodes(ctx context.Context, relatedCodes []string) (map[string]model.MaterialTest, error) {
	latest := make(map[string]model.MaterialTest)
	if len(relatedCodes) == 0 {
		return latest, nil
	}
	var items []model.MaterialTest
	err := r.db.WithContext(ctx).
		Where("related_code IN ? AND status = ?", relatedCodes, "verified").
		Order("related_code ASC, updated_at DESC, id DESC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		// 列表已按更新时间倒序，首次出现的即该关联编码下最新的已核验检测。
		if _, exists := latest[item.RelatedCode]; !exists {
			latest[item.RelatedCode] = item
		}
	}
	return latest, nil
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
