package database

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"gorm.io/gorm"

	"lms/backend/pkg/domain/assignments"
)

type AssignmentRepository struct {
	db *gorm.DB
}

func NewAssignmentRepository(db *gorm.DB) *AssignmentRepository {
	return &AssignmentRepository{db: db}
}

func (repository *AssignmentRepository) CreateClassCourse(ctx context.Context, entity *assignments.ClassCourse) error {
	return translate(repository.db.WithContext(ctx).Create(entity).Error)
}

func (repository *AssignmentRepository) GetClassCourse(ctx context.Context, id uuid.UUID) (*assignments.ClassCourse, error) {
	var entity assignments.ClassCourse
	err := repository.db.WithContext(ctx).First(&entity, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &entity, nil
}

func (repository *AssignmentRepository) DeleteClassCourse(ctx context.Context, id uuid.UUID) error {
	return translate(repository.db.WithContext(ctx).Delete(&assignments.ClassCourse{}, "id = ?", id).Error)
}

func (repository *AssignmentRepository) ListClassCourses(ctx context.Context, params assignments.ListParams) ([]assignments.ClassCourse, error) {
	offset, limit := window(params.Page, params.Limit, 50)
	query := repository.db.WithContext(ctx).Model(&assignments.ClassCourse{})
	if params.ClassID != nil {
		query = query.Where("class_id = ?", *params.ClassID)
	}
	if params.CourseID != nil {
		query = query.Where("course_id = ?", *params.CourseID)
	}
	if params.TeacherID != nil {
		query = query.Joins("JOIN teacher_assignments ON teacher_assignments.class_id = class_courses.class_id AND teacher_assignments.deleted_at IS NULL").
			Where("teacher_assignments.teacher_id = ?", *params.TeacherID).
			Where("teacher_assignments.active = ?", true)
	}
	if params.Term != "" {
		query = query.Where("class_courses.term = ?", params.Term)
	}
	var result []assignments.ClassCourse
	err := query.Order("class_id ASC, course_id ASC").Offset(offset).Limit(limit).Find(&result).Error
	return result, translate(err)
}

func (repository *AssignmentRepository) ClassCourseExists(ctx context.Context, classID uuid.UUID, courseID uuid.UUID) (bool, error) {
	return exists(ctx, repository.db, &assignments.ClassCourse{}, "class_id = ? AND course_id = ? AND active = ?", classID, courseID, true)
}

func (repository *AssignmentRepository) CreateTeacherAssignment(ctx context.Context, entity *assignments.TeacherAssignment) error {
	return translate(repository.db.WithContext(ctx).Create(entity).Error)
}

func (repository *AssignmentRepository) GetTeacherAssignment(ctx context.Context, id uuid.UUID) (*assignments.TeacherAssignment, error) {
	var entity assignments.TeacherAssignment
	err := repository.db.WithContext(ctx).First(&entity, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &entity, nil
}

func (repository *AssignmentRepository) DeleteTeacherAssignment(ctx context.Context, id uuid.UUID) error {
	return translate(repository.db.WithContext(ctx).Delete(&assignments.TeacherAssignment{}, "id = ?", id).Error)
}

func (repository *AssignmentRepository) ListTeacherAssignments(ctx context.Context, params assignments.ListParams) ([]assignments.TeacherAssignment, error) {
	offset, limit := window(params.Page, params.Limit, 50)
	query := repository.db.WithContext(ctx).Model(&assignments.TeacherAssignment{})
	if params.ClassID != nil {
		query = query.Where("class_id = ?", *params.ClassID)
	}
	if params.CourseID != nil {
		query = query.Where("course_id = ?", *params.CourseID)
	}
	if params.TeacherID != nil {
		query = query.Where("teacher_id = ?", *params.TeacherID)
	}
	var result []assignments.TeacherAssignment
	err := query.Order("class_id ASC, course_id ASC").Offset(offset).Limit(limit).Find(&result).Error
	return result, translate(err)
}

func (repository *AssignmentRepository) Exists(ctx context.Context, classID uuid.UUID, courseID uuid.UUID, teacherID uuid.UUID) (bool, error) {
	return exists(ctx, repository.db, &assignments.TeacherAssignment{},
		"class_id = ? AND course_id = ? AND teacher_id = ? AND active = ?", classID, courseID, teacherID, true)
}

func (repository *AssignmentRepository) ExistsInScope(ctx context.Context, classID uuid.UUID, courseID uuid.UUID, teacherID uuid.UUID) (bool, error) {
	conditions := []string{"class_id = ?", "teacher_id = ?", "active = ?"}
	args := []any{classID, teacherID, true}
	if courseID != uuid.Nil {
		conditions = append(conditions, "course_id = ?")
		args = append(args, courseID)
	}
	return exists(ctx, repository.db, &assignments.TeacherAssignment{}, strings.Join(conditions, " AND "), args...)
}

func (repository *AssignmentRepository) TeacherIDsForClass(ctx context.Context, classID uuid.UUID, courseID uuid.UUID) ([]uuid.UUID, error) {
	query := repository.db.WithContext(ctx).Model(&assignments.TeacherAssignment{}).
		Where("class_id = ? AND active = ?", classID, true)
	if courseID != uuid.Nil {
		query = query.Where("course_id = ?", courseID)
	}
	var ids []uuid.UUID
	err := query.Distinct().Pluck("teacher_id", &ids).Error
	return ids, translate(err)
}

func (repository *AssignmentRepository) ClassIDsForTeacher(ctx context.Context, teacherID uuid.UUID, courseID uuid.UUID) ([]uuid.UUID, error) {
	query := repository.db.WithContext(ctx).Model(&assignments.TeacherAssignment{}).
		Where("teacher_id = ? AND active = ?", teacherID, true)
	if courseID != uuid.Nil {
		query = query.Where("course_id = ?", courseID)
	}
	var ids []uuid.UUID
	err := query.Distinct().Pluck("class_id", &ids).Error
	return ids, translate(err)
}

func (repository *AssignmentRepository) CourseIDsForTeacher(ctx context.Context, teacherID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := repository.db.WithContext(ctx).Model(&assignments.TeacherAssignment{}).
		Where("teacher_id = ? AND active = ?", teacherID, true).
		Distinct().
		Pluck("course_id", &ids).Error
	return ids, translate(err)
}

func (repository *AssignmentRepository) ClassCourseIDsForTeacher(ctx context.Context, teacherID uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	var rows []assignments.TeacherAssignment
	err := repository.db.WithContext(ctx).
		Where("teacher_id = ? AND active = ?", teacherID, true).
		Find(&rows).Error
	if err != nil {
		return nil, translate(err)
	}
	result := make(map[uuid.UUID][]uuid.UUID, len(rows))
	for _, row := range rows {
		result[row.CourseID] = appendUniqueID(result[row.CourseID], row.ClassID)
	}
	return result, nil
}

func appendUniqueID(ids []uuid.UUID, id uuid.UUID) []uuid.UUID {
	for _, existing := range ids {
		if existing == id {
			return ids
		}
	}
	return append(ids, id)
}
