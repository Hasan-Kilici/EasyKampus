package campuses

import (
	"context"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/base"
	"lms/backend/pkg/domain/classes"
	"lms/backend/pkg/domain/scopeguard"
)

type Campus struct {
	base.Model
	SchoolID uuid.UUID       `gorm:"type:char(36);not null;uniqueIndex:idx_campuses_school_code,priority:1;index" json:"school_id"`
	Name     string          `gorm:"size:200;not null" json:"name"`
	Code     string          `gorm:"size:32;not null;uniqueIndex:idx_campuses_school_code,priority:2" json:"code"`
	Classes  []classes.Class `gorm:"foreignKey:CampusID" json:"classes"`
}

type ListParams struct {
	Page     int        `json:"page"`
	Limit    int        `json:"limit"`
	SchoolID *uuid.UUID `json:"school_id"`
	Query    string     `json:"query"`
	// Viewer, listeyi bakan kullanıcının görebildikleriyle sınırlar; nil sınırsızdır (yalnızca iç çağrılar).
	Viewer *scopeguard.Viewer `json:"-"`
}

type Repository interface {
	Create(ctx context.Context, campus *Campus) error
	GetByID(ctx context.Context, id uuid.UUID) (*Campus, error)
	Update(ctx context.Context, campus *Campus) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params ListParams) ([]Campus, error)
	SchoolExists(ctx context.Context, schoolID uuid.UUID) (bool, error)
}
