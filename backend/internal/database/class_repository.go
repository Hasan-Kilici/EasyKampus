package database

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"lms/backend/pkg/domain/classes"
	"lms/backend/pkg/domain/enrollments"
	"lms/backend/pkg/domain/users"
)

type ClassRepository struct {
	db     *gorm.DB
	access *AccessRepository
}

func NewClassRepository(db *gorm.DB, access *AccessRepository) *ClassRepository {
	return &ClassRepository{db: db, access: access}
}

func (repository *ClassRepository) Create(ctx context.Context, class *classes.Class) error {
	return translate(repository.db.WithContext(ctx).Create(class).Error)
}

func (repository *ClassRepository) GetByID(ctx context.Context, id uuid.UUID) (*classes.Class, error) {
	var class classes.Class
	err := repository.db.WithContext(ctx).First(&class, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &class, nil
}

func (repository *ClassRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	return exists(ctx, repository.db, &classes.Class{}, "id = ?", id)
}

func (repository *ClassRepository) Update(ctx context.Context, class *classes.Class) error {
	return translate(repository.db.WithContext(ctx).Save(class).Error)
}

func (repository *ClassRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return translate(repository.db.WithContext(ctx).Delete(&classes.Class{}, "id = ?", id).Error)
}

func (repository *ClassRepository) List(ctx context.Context, params classes.ListParams, scope classes.Scope) ([]classes.Class, error) {
	offset, limit := window(params.Page, params.Limit, 50)
	var result []classes.Class
	err := repository.applyParams(repository.scoped(ctx, scope), params).
		Offset(offset).Limit(limit).Order("classes.name ASC").Find(&result).Error
	return result, translate(err)
}

func (repository *ClassRepository) Count(ctx context.Context, params classes.ListParams, scope classes.Scope) (int64, error) {
	var count int64
	err := repository.applyParams(repository.scoped(ctx, scope), params).Count(&count).Error
	return count, translate(err)
}

func (repository *ClassRepository) SchoolIDsForStaff(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return repository.access.SchoolIDsForStaff(ctx, userID)
}

func (repository *ClassRepository) CampusIDsForSchool(ctx context.Context, schoolID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := repository.db.WithContext(ctx).Model(&classes.Class{}).
		Where("school_id = ?", schoolID).
		Distinct().
		Pluck("campus_id", &ids).Error
	return ids, translate(err)
}

func (repository *ClassRepository) scoped(ctx context.Context, scope classes.Scope) *gorm.DB {
	query := repository.db.WithContext(ctx).Model(&classes.Class{})
	switch scope.Role {
	case users.RoleSuperAdmin:
	case users.RoleAdmin:
		schoolIDs, err := repository.access.SchoolIDsForStaff(ctx, scope.UserID)
		if err != nil || len(schoolIDs) == 0 {
			return query.Where("1 = 0")
		}
		query = query.Where("classes.school_id IN ?", schoolIDs)
	case users.RoleTeacher:
		query = query.Joins(teacherAssignmentJoin).Where("teacher_assignments.teacher_id = ?", scope.UserID).Where("teacher_assignments.active = ?", true)
	case users.RoleStudent:
		query = query.Joins(enrollmentJoin).Where("enrollments.student_id = ?", scope.UserID).Where("enrollments.status = ?", enrollments.StatusActive)
	default:
		return query.Where("1 = 0")
	}
	return query
}

func (repository *ClassRepository) applyParams(query *gorm.DB, params classes.ListParams) *gorm.DB {
	if params.SchoolID != nil {
		query = query.Where("classes.school_id = ?", *params.SchoolID)
	}
	if params.CampusID != nil {
		query = query.Where("classes.campus_id = ?", *params.CampusID)
	}
	if params.CourseID != nil {
		query = query.Joins("JOIN class_courses ON class_courses.class_id = classes.id AND class_courses.deleted_at IS NULL").
			Where("class_courses.course_id = ?", *params.CourseID)
	}
	if keyword := strings.TrimSpace(params.Query); keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		query = query.Where("LOWER(classes.name) LIKE ? OR LOWER(classes.level) LIKE ?", like, like)
	}
	return query
}
