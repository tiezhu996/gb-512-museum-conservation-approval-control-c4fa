package repository

import (
	"context"

	"github.com/blueship581/museum-conservation-approval-control/backend/internal/dto"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/model"
	"gorm.io/gorm"
)

// ArtifactRepository owns all persistence operations for 文物.
type ArtifactRepository interface {
	List(context.Context, dto.PageQuery) (Page[model.Artifact], error)
	Get(context.Context, uint) (model.Artifact, error)
	Create(context.Context, *model.Artifact) error
	Update(context.Context, uint, uint, *model.Artifact) error
	Delete(context.Context, uint) error
	CountByStatus(context.Context) (map[string]int64, error)
}

type artifactRepository struct {
	store *Store[model.Artifact]
}

func NewArtifactRepository(db *gorm.DB) ArtifactRepository {
	return &artifactRepository{store: NewStore[model.Artifact](db)}
}

func (r *artifactRepository) List(ctx context.Context, q dto.PageQuery) (Page[model.Artifact], error) {
	return r.store.List(ctx, q)
}
func (r *artifactRepository) Get(ctx context.Context, id uint) (model.Artifact, error) {
	return r.store.Get(ctx, id)
}
func (r *artifactRepository) Create(ctx context.Context, item *model.Artifact) error {
	return r.store.Create(ctx, item)
}
func (r *artifactRepository) Update(ctx context.Context, id, version uint, item *model.Artifact) error {
	return r.store.Update(ctx, id, version, item)
}
func (r *artifactRepository) Delete(ctx context.Context, id uint) error {
	return r.store.Delete(ctx, id)
}
func (r *artifactRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	return r.store.CountByStatus(ctx)
}
