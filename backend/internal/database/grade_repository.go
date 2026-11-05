package database

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"lms/backend/pkg/domain/grades"
)

type GradeRepository struct {
	db *gorm.DB
}

func NewGradeRepository(db *gorm.DB) *GradeRepository {
	return &GradeRepository{db: db}
}

func (repository *GradeRepository) Upsert(ctx context.Context, grade *grades.Grade) error {
	return translate(repository.db.WithContext(ctx).
		Clauses(liveRowConflict(
			[]string{"student_id", "class_id", "course_id", "term"},
			clause.AssignmentColumns([]string{"quiz_id", "score", "type", "graded_by", "note", "updated_at", "deleted_at"}),
		)).
		Create(grade).Error)
}

func (repository *GradeRepository) BulkUpsert(ctx context.Context, items []*grades.Grade) error {
	return repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, grade := range items {
			if err := translate(tx.Clauses(liveRowConflict(
				[]string{"student_id", "class_id", "course_id", "term"},
				clause.AssignmentColumns([]string{"quiz_id", "score", "type", "graded_by", "note", "updated_at", "deleted_at"}),
			)).Create(grade).Error); err != nil {
				return err
			}
		}
		return nil
	})
}

func (repository *GradeRepository) BulkUpsertWithAudits(ctx context.Context, items []*grades.Grade, audits []*grades.Audit) error {
	return repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for index, grade := range items {
			if err := translate(tx.Clauses(liveRowConflict(
				[]string{"student_id", "class_id", "course_id", "term"},
				clause.AssignmentColumns([]string{"quiz_id", "score", "type", "graded_by", "note", "updated_at", "deleted_at"}),
			)).Create(grade).Error); err != nil {
				return err
			}
			var existing grades.Grade
			if err := tx.Select("id").
				Where("student_id = ? AND class_id = ? AND course_id = ? AND term = ? AND deleted_at IS NULL", grade.StudentID, grade.ClassID, grade.CourseID, grade.Term).
				First(&existing).Error; err == nil && existing.ID != uuid.Nil {
				grade.ID = existing.ID
				if index < len(audits) && audits[index] != nil {
					audits[index].GradeID = existing.ID
				}
			}
		}
		if len(audits) > 0 {
			if err := translate(tx.Create(audits).Error); err != nil {
				return err
			}
		}
		return nil
	})
}

func (repository *GradeRepository) CreateAudits(ctx context.Context, audits []*grades.Audit) error {
	return translate(repository.db.WithContext(ctx).Create(audits).Error)
}

func (repository *GradeRepository) ListAudits(ctx context.Context, gradeID uuid.UUID) ([]grades.Audit, error) {
	var result []grades.Audit
	err := repository.db.WithContext(ctx).Where("grade_id = ?", gradeID).Order("created_at ASC").Find(&result).Error
	return result, translate(err)
}

func (repository *GradeRepository) GetByID(ctx context.Context, id uuid.UUID) (*grades.Grade, error) {
	var grade grades.Grade
	err := repository.db.WithContext(ctx).First(&grade, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &grade, nil
}

func (repository *GradeRepository) Update(ctx context.Context, grade *grades.Grade) error {
	return translate(repository.db.WithContext(ctx).Save(grade).Error)
}

func (repository *GradeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return translate(repository.db.WithContext(ctx).Delete(&grades.Grade{}, "id = ?", id).Error)
}

func (repository *GradeRepository) List(ctx context.Context, params grades.ListParams) ([]grades.Grade, error) {
	if params.Limit < 1 || params.Limit > 500 {
		params.Limit = 100
	}
	query := repository.db.WithContext(ctx).Model(&grades.Grade{})
	if params.ScopeApplied && len(params.ClassIDs) == 0 {
		return []grades.Grade{}, nil
	}
	if params.StudentID != nil {
		query = query.Where("student_id = ?", *params.StudentID)
	}
	if params.ClassID != nil {
		query = query.Where("class_id = ?", *params.ClassID)
	}
	if len(params.ClassIDs) > 0 {
		query = query.Where("class_id IN ?", params.ClassIDs)
	}
	if params.CourseID != nil {
		query = query.Where("course_id = ?", *params.CourseID)
	}
	if params.QuizID != nil {
		query = query.Where("quiz_id = ?", *params.QuizID)
	}
	if term := strings.TrimSpace(params.Term); term != "" {
		query = query.Where("term = ?", term)
	}
	if params.Type != "" {
		query = query.Where("type = ?", params.Type)
	}
	var result []grades.Grade
	err := query.Order("created_at DESC").Limit(params.Limit).Find(&result).Error
	return result, translate(err)
}

func (repository *GradeRepository) ListByStudentAndCourse(ctx context.Context, studentID uuid.UUID, courseID uuid.UUID, term string) ([]grades.Grade, error) {
	var result []grades.Grade
	query := repository.db.WithContext(ctx).
		Where("student_id = ? AND course_id = ?", studentID, courseID)
	if strings.TrimSpace(term) != "" {
		query = query.Where("term = ?", term)
	}
	err := query.Order("created_at DESC").Find(&result).Error
	return result, translate(err)
}
