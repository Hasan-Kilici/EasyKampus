package topics

import (
	"context"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/base"
)

type Topic struct {
	base.Model
	CourseID uuid.UUID  `gorm:"type:char(36);not null;uniqueIndex:idx_topics_course_code,priority:1;index" json:"course_id"`
	ParentID *uuid.UUID `gorm:"type:char(36);index" json:"parent_id"`
	Name     string     `gorm:"size:150;not null" json:"name"`
	Code     string     `gorm:"size:32;not null;uniqueIndex:idx_topics_course_code,priority:2" json:"code"`
	Weight   float64    `gorm:"not null;default:1" json:"weight"`
	Active   bool       `gorm:"not null;default:true;index" json:"active"`
	// Description, konunun ne anlattığıdır. Bölümle ilgi puanı (TF-IDF) bu metinden
	// ve konu adından hesaplanır; bu yüzden zorunludur.
	Description string `gorm:"type:text" json:"description"`
	// StudyHours (EFOR): bu konunun işlenmesi/çalışılması için ayrılan süre (saat).
	StudyHours float64 `gorm:"not null;default:0" json:"study_hours"`
}

type ListParams struct {
	Page       int       `json:"page"`
	Limit      int       `json:"limit"`
	CourseID   uuid.UUID `json:"course_id"`
	ActiveOnly bool      `json:"active_only"`
}

type Repository interface {
	Create(ctx context.Context, topic *Topic) error
	GetByID(ctx context.Context, id uuid.UUID) (*Topic, error)
	Update(ctx context.Context, topic *Topic) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByCourse(ctx context.Context, courseID uuid.UUID, activeOnly bool) ([]Topic, error)
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}
