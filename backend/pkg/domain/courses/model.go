package courses

import (
	"context"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/base"
	"lms/backend/pkg/domain/scopeguard"
	"lms/backend/pkg/domain/topics"
)

type Course struct {
	base.Model
	SchoolID    uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_courses_school_code,priority:1;index" json:"school_id"`
	Name        string    `gorm:"size:150;not null;index" json:"name"`
	Code        string    `gorm:"size:32;not null;uniqueIndex:idx_courses_school_code,priority:2" json:"code"`
	SubjectArea string    `gorm:"size:100;index" json:"subject_area"`
	Description string    `gorm:"type:text" json:"description"`
	Active      bool      `gorm:"not null;default:true;index" json:"active"`
	// Department (bölüm), konuların bölümle ilgisini ölçmek için kullanılır
	// (örn. "Bilgisayar Programcılığı"). DepartmentKeywords virgülle ayrılmış
	// ek anahtar kelimelerdir; sözlükte olmayan terimleri buraya yazabilirsiniz.
	Department         string `gorm:"size:150;index" json:"department"`
	DepartmentKeywords string `gorm:"type:text" json:"department_keywords"`
	// HomeworkHours ve ProjectHours (EFOR): ders boyunca ödeve/projeye ayrılacak
	// toplam süre (saat). Konu saatleriyle birlikte AKTS'yi belirler.
	HomeworkHours float64        `gorm:"not null;default:0" json:"homework_hours"`
	ProjectHours  float64        `gorm:"not null;default:0" json:"project_hours"`
	Topics        []topics.Topic `gorm:"foreignKey:CourseID" json:"topics"`
}

type ListParams struct {
	Page        int        `json:"page"`
	Limit       int        `json:"limit"`
	SchoolID    *uuid.UUID `json:"school_id"`
	SubjectArea string     `json:"subject_area"`
	ActiveOnly  bool       `json:"active_only"`
	Query       string     `json:"query"`
	// Viewer, listeyi bakan kullanıcının görebildikleriyle sınırlar; nil sınırsızdır (yalnızca iç çağrılar).
	Viewer *scopeguard.Viewer `json:"-"`
}

type Repository interface {
	Create(ctx context.Context, course *Course) error
	GetByID(ctx context.Context, id uuid.UUID) (*Course, error)
	Update(ctx context.Context, course *Course) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params ListParams) ([]Course, error)
}
