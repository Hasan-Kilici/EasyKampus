package schools

import (
	"context"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/base"
	"lms/backend/pkg/domain/campuses"
	"lms/backend/pkg/domain/classes"
	"lms/backend/pkg/domain/scopeguard"
)

type StaffRole string

const (
	StaffRoleDirector  StaffRole = "director"  // Okulun yönetim kaynaklarına tam erişir.
	StaffRoleRegistrar StaffRole = "registrar" // Okul kayıt ve kampüs işlemlerini yönetir.
)

type School struct {
	base.Model
	Name     string            `gorm:"size:200;not null" json:"name"`
	Code     string            `gorm:"size:32;not null;uniqueIndex" json:"code"`
	Campuses []campuses.Campus `gorm:"foreignKey:SchoolID" json:"campuses"`
	Classes  []classes.Class   `gorm:"foreignKey:SchoolID" json:"classes"`
	Staff    []Staff           `gorm:"foreignKey:SchoolID" json:"staff"`
}

type Staff struct {
	base.Model
	SchoolID uuid.UUID `gorm:"type:char(36);primaryKey" json:"school_id"`
	UserID   uuid.UUID `gorm:"type:char(36);primaryKey;index" json:"user_id"`
	Role     StaffRole `gorm:"size:20;not null;index" json:"role"`
}

type ListParams struct {
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
	Query string `json:"query"`
	// Viewer, listeyi bakan kullanıcının görebildikleriyle sınırlar; nil sınırsızdır (yalnızca iç çağrılar).
	Viewer *scopeguard.Viewer `json:"-"`
}

type Repository interface {
	Create(ctx context.Context, school *School) error
	GetByID(ctx context.Context, id uuid.UUID) (*School, error)
	Update(ctx context.Context, school *School) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params ListParams) ([]School, error)
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
	ListStaff(ctx context.Context, schoolID uuid.UUID) ([]Staff, error)
	AssignStaff(ctx context.Context, staff *Staff) error
	RemoveStaff(ctx context.Context, schoolID uuid.UUID, userID uuid.UUID) error
}
