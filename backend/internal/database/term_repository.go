package database

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"lms/backend/pkg/domain/terms"
)

type TermRepository struct{ db *gorm.DB }

func NewTermRepository(db *gorm.DB) *TermRepository { return &TermRepository{db: db} }

func (r *TermRepository) Create(ctx context.Context, term *terms.Term) error {
	return translate(r.db.WithContext(ctx).Create(term).Error)
}

func (r *TermRepository) GetByID(ctx context.Context, id uuid.UUID) (*terms.Term, error) {
	var term terms.Term
	if err := r.db.WithContext(ctx).First(&term, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}
	return &term, nil
}

func (r *TermRepository) List(ctx context.Context, schoolID uuid.UUID) ([]terms.Term, error) {
	var result []terms.Term
	err := r.db.WithContext(ctx).Where("school_id = ?", schoolID).Order("starts_at DESC").Find(&result).Error
	return result, translate(err)
}

func (r *TermRepository) HasActive(ctx context.Context, schoolID uuid.UUID) (bool, error) {
	return exists(ctx, r.db, &terms.Term{}, "school_id = ? AND status = ?", schoolID, terms.StatusActive)
}

func (r *TermRepository) Update(ctx context.Context, term *terms.Term) error {
	return translate(r.db.WithContext(ctx).Save(term).Error)
}
