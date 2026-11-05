package database

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"lms/backend/pkg/domain/assignments"
	"lms/backend/pkg/domain/campuses"
	"lms/backend/pkg/domain/classes"
	"lms/backend/pkg/domain/enrollments"
	"lms/backend/pkg/domain/schools"
	"lms/backend/pkg/domain/users"
)

type AccessRepository struct {
	db *gorm.DB
}

func NewAccessRepository(db *gorm.DB) *AccessRepository {
	return &AccessRepository{db: db}
}

func (repository *AccessRepository) HasAccess(ctx context.Context, userID uuid.UUID, classID uuid.UUID) (bool, error) {
	role, err := repository.role(ctx, userID)
	if err != nil {
		return false, err
	}
	switch role {
	case users.RoleSuperAdmin:
		return true, nil
	case users.RoleAdmin:
		schoolIDs, err := repository.staffSchoolIDs(ctx, userID, false)
		if err != nil || len(schoolIDs) == 0 {
			return false, err
		}
		return exists(ctx, repository.db, &classes.Class{}, "id = ? AND school_id IN ?", classID, schoolIDs)
	case users.RoleTeacher:
		return repository.hasTeacherClass(ctx, userID, classID)
	case users.RoleStudent:
		return repository.hasStudentClass(ctx, userID, classID)
	default:
		return false, nil
	}
}

func (repository *AccessRepository) HasClassManagementAccess(ctx context.Context, userID uuid.UUID, classID uuid.UUID) (bool, error) {
	role, err := repository.role(ctx, userID)
	if err != nil {
		return false, err
	}
	if role == users.RoleSuperAdmin {
		return true, nil
	}
	if role != users.RoleAdmin {
		return false, nil
	}
	schoolIDs, err := repository.staffSchoolIDs(ctx, userID, true)
	if err != nil || len(schoolIDs) == 0 {
		return false, err
	}
	return exists(ctx, repository.db, &classes.Class{}, "id = ? AND school_id IN ?", classID, schoolIDs)
}

func (repository *AccessRepository) HasTeacherAccess(ctx context.Context, userID uuid.UUID, classID uuid.UUID) (bool, error) {
	role, err := repository.role(ctx, userID)
	if err != nil {
		return false, err
	}
	switch role {
	case users.RoleSuperAdmin:
		return true, nil
	case users.RoleAdmin:
		schoolIDs, err := repository.staffSchoolIDs(ctx, userID, false)
		if err != nil || len(schoolIDs) == 0 {
			return false, err
		}
		return exists(ctx, repository.db, &classes.Class{}, "id = ? AND school_id IN ?", classID, schoolIDs)
	case users.RoleTeacher:
		return repository.hasTeacherClass(ctx, userID, classID)
	default:
		return false, nil
	}
}

func (repository *AccessRepository) HasSchoolAccess(ctx context.Context, userID uuid.UUID, schoolID uuid.UUID) (bool, error) {
	role, err := repository.role(ctx, userID)
	if err != nil {
		return false, err
	}
	switch role {
	case users.RoleSuperAdmin:
		return true, nil
	case users.RoleAdmin:
		ids, err := repository.staffSchoolIDs(ctx, userID, false)
		if err != nil {
			return false, err
		}
		return containsID(ids, schoolID), nil
	case users.RoleTeacher:
		return repository.schoolInScope(ctx, &classes.Class{}, teacherAssignmentJoin, userID, schoolID)
	case users.RoleStudent:
		return repository.schoolInScope(ctx, &classes.Class{}, enrollmentJoin, userID, schoolID)
	default:
		return false, nil
	}
}

func (repository *AccessRepository) HasSchoolStaffAccess(ctx context.Context, userID uuid.UUID, schoolID uuid.UUID) (bool, error) {
	role, err := repository.role(ctx, userID)
	if err != nil {
		return false, err
	}
	if role == users.RoleSuperAdmin {
		return true, nil
	}
	if role != users.RoleAdmin {
		return false, nil
	}
	return exists(ctx, repository.db, &schools.Staff{}, "school_id = ? AND user_id = ?", schoolID, userID)
}

// HasTeacherSubjectAccess, çağıran personelin görev yaptığı okulda görev alan
// öğretmenin verisine erişmesine izin verir. Yönetici, müdür ve kayıt görevlisi
// okul personel kaydı üzerinden yetkilendirilir.
func (repository *AccessRepository) HasTeacherSubjectAccess(ctx context.Context, userID uuid.UUID, teacherID uuid.UUID) (bool, error) {
	role, err := repository.role(ctx, userID)
	if err != nil {
		return false, err
	}
	if role == users.RoleSuperAdmin {
		return true, nil
	}
	if role == users.RoleStudent || role == users.RoleTeacher {
		return false, nil
	}
	managed := repository.db.WithContext(ctx).Model(&schools.Staff{}).
		Select("school_id").
		Where("user_id = ?", userID)
	return exists(ctx, repository.db, &schools.Staff{},
		"user_id = ? AND school_id IN (?)", teacherID, managed)
}

func (repository *AccessRepository) HasCampusAccess(ctx context.Context, userID uuid.UUID, campusID uuid.UUID) (bool, error) {
	var campus campuses.Campus
	if err := repository.db.WithContext(ctx).Select("id", "school_id").First(&campus, "id = ?", campusID).Error; err != nil {
		return false, translate(err)
	}
	return repository.HasSchoolAccess(ctx, userID, campus.SchoolID)
}

func (repository *AccessRepository) HasCourseAccess(ctx context.Context, userID uuid.UUID, courseID uuid.UUID) (bool, error) {
	role, err := repository.role(ctx, userID)
	if err != nil {
		return false, err
	}
	switch role {
	case users.RoleSuperAdmin:
		return true, nil
	case users.RoleAdmin:
		ids, err := repository.staffSchoolIDs(ctx, userID, false)
		if err != nil || len(ids) == 0 {
			return false, err
		}
		return exists(ctx, repository.db, &assignments.ClassCourse{},
			"course_id = ? AND active = ? AND class_id IN (SELECT id FROM classes WHERE school_id IN ?)", courseID, true, ids)
	case users.RoleTeacher:
		var count int64
		err := repository.db.WithContext(ctx).Model(&classes.Class{}).
			Joins(teacherAssignmentJoin).
			Where("teacher_assignments.teacher_id = ?", userID).
			Where("teacher_assignments.active = ?", true).
			Where("classes.id IN (SELECT class_id FROM class_courses WHERE course_id = ? AND active = ? AND deleted_at IS NULL)", courseID, true).
			Count(&count).Error
		return count > 0, err
	case users.RoleStudent:
		var count int64
		err := repository.db.WithContext(ctx).Model(&enrollments.Enrollment{}).
			Joins("JOIN class_courses ON class_courses.class_id = enrollments.class_id AND class_courses.deleted_at IS NULL").
			Where("enrollments.student_id = ?", userID).
			Where("enrollments.status = ?", enrollments.StatusActive).
			Where("enrollments.deleted_at IS NULL").
			Where("class_courses.course_id = ? AND class_courses.active = ?", courseID, true).
			Count(&count).Error
		return count > 0, err
	default:
		return false, nil
	}
}

func (repository *AccessRepository) HasCourseWriteAccess(ctx context.Context, userID uuid.UUID, courseID uuid.UUID) (bool, error) {
	role, err := repository.role(ctx, userID)
	if err != nil {
		return false, err
	}
	switch role {
	case users.RoleSuperAdmin:
		return true, nil
	case users.RoleAdmin:
		ids, err := repository.staffSchoolIDs(ctx, userID, true)
		if err != nil || len(ids) == 0 {
			return false, err
		}
		return exists(ctx, repository.db, &assignments.ClassCourse{},
			"course_id = ? AND active = ? AND class_id IN (SELECT id FROM classes WHERE school_id IN ?)", courseID, true, ids)
	case users.RoleTeacher:
		var count int64
		err := repository.db.WithContext(ctx).Model(&classes.Class{}).
			Joins(teacherAssignmentJoin).
			Where("teacher_assignments.teacher_id = ?", userID).
			Where("teacher_assignments.active = ?", true).
			Where("classes.id IN (SELECT class_id FROM class_courses WHERE course_id = ? AND active = ? AND deleted_at IS NULL)", courseID, true).
			Count(&count).Error
		return count > 0, err
	default:
		return false, nil
	}
}

func (repository *AccessRepository) HasStudentAccess(ctx context.Context, userID uuid.UUID, studentID uuid.UUID) (bool, error) {
	if userID == studentID {
		return true, nil
	}
	role, err := repository.role(ctx, userID)
	if err != nil {
		return false, err
	}
	switch role {
	case users.RoleSuperAdmin:
		return true, nil
	case users.RoleAdmin:
		ids, err := repository.staffSchoolIDs(ctx, userID, false)
		if err != nil || len(ids) == 0 {
			return false, err
		}
		return exists(ctx, repository.db, &enrollments.Enrollment{},
			"student_id = ? AND status = ? AND deleted_at IS NULL AND class_id IN (SELECT id FROM classes WHERE school_id IN ?)", studentID, enrollments.StatusActive, ids)
	case users.RoleTeacher:
		var count int64
		err := repository.db.WithContext(ctx).Model(&enrollments.Enrollment{}).
			Joins("JOIN teacher_assignments ON teacher_assignments.class_id = enrollments.class_id AND teacher_assignments.deleted_at IS NULL").
			Where("teacher_assignments.teacher_id = ?", userID).
			Where("teacher_assignments.active = ?", true).
			Where("enrollments.student_id = ?", studentID).
			Where("enrollments.status = ?", enrollments.StatusActive).
			Where("enrollments.deleted_at IS NULL").
			Count(&count).Error
		return count > 0, err
	default:
		return false, nil
	}
}

// ScopedClassIDs, çağıran personelin görmesi gereken sınıfların kümesini döner.
// Superadmin için kısıt uygulanmaz; diğer roller için sınıflar okul personeli
// veya öğretmen ataması üzerinden filtrelenir.
func (repository *AccessRepository) ScopedClassIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	role, err := repository.role(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := []uuid.UUID{}
	switch role {
	case users.RoleSuperAdmin:
		return nil, nil
	case users.RoleTeacher:
		err = repository.db.WithContext(ctx).Model(&classes.Class{}).
			Joins(teacherAssignmentJoin).
			Where("teacher_assignments.teacher_id = ?", userID).
			Where("teacher_assignments.active = ?", true).
			Distinct().Pluck("classes.id", &ids).Error
	case users.RoleAdmin:
		schoolIDs, schoolErr := repository.staffSchoolIDs(ctx, userID, false)
		if schoolErr != nil {
			return nil, schoolErr
		}
		if len(schoolIDs) == 0 {
			return ids, nil
		}
		err = repository.db.WithContext(ctx).Model(&classes.Class{}).
			Where("school_id IN ?", schoolIDs).
			Pluck("id", &ids).Error
	case users.RoleStudent:
		// Öğrenci yalnızca aktif kayıtlı olduğu sınıfları görür (ders programı gibi sınıf bazlı okumalar için).
		err = repository.db.WithContext(ctx).Model(&enrollments.Enrollment{}).
			Where("student_id = ?", userID).
			Where("status = ?", enrollments.StatusActive).
			Where("deleted_at IS NULL").
			Distinct().Pluck("class_id", &ids).Error
	default:
		return ids, nil
	}
	if err != nil {
		return nil, translate(err)
	}
	return ids, nil
}

func (repository *AccessRepository) SchoolIDsForStaff(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	role, err := repository.role(ctx, userID)
	if err != nil {
		return nil, err
	}
	if role == users.RoleSuperAdmin {
		var ids []uuid.UUID
		err := repository.db.WithContext(ctx).Model(&schools.School{}).Pluck("id", &ids).Error
		return ids, err
	}
	return repository.staffSchoolIDs(ctx, userID, false)
}

func (repository *AccessRepository) role(ctx context.Context, userID uuid.UUID) (users.Role, error) {
	var user users.User
	if err := repository.db.WithContext(ctx).Select("id", "role").First(&user, "id = ?", userID).Error; err != nil {
		return "", translate(err)
	}
	return user.Role, nil
}

func (repository *AccessRepository) staffSchoolIDs(ctx context.Context, userID uuid.UUID, directorOnly bool) ([]uuid.UUID, error) {
	query := repository.db.WithContext(ctx).Model(&schools.Staff{}).Where("user_id = ?", userID)
	if directorOnly {
		query = query.Where("role = ?", schools.StaffRoleDirector)
	}
	var ids []uuid.UUID
	err := query.Distinct().Pluck("school_id", &ids).Error
	return ids, err
}

func (repository *AccessRepository) hasTeacherClass(ctx context.Context, teacherID uuid.UUID, classID uuid.UUID) (bool, error) {
	if classID == uuid.Nil {
		var count int64
		err := repository.db.WithContext(ctx).Model(&classes.Class{}).
			Joins(teacherAssignmentJoin).
			Where("teacher_assignments.teacher_id = ?", teacherID).
			Where("teacher_assignments.active = ?", true).
			Count(&count).Error
		return count > 0, err
	}
	return exists(ctx, repository.db, &assignments.TeacherAssignment{},
		"teacher_id = ? AND class_id = ? AND active = ?", teacherID, classID, true)
}

func (repository *AccessRepository) hasStudentClass(ctx context.Context, studentID uuid.UUID, classID uuid.UUID) (bool, error) {
	query := repository.db.WithContext(ctx).Model(&enrollments.Enrollment{}).
		Where("student_id = ?", studentID).
		Where("status = ?", enrollments.StatusActive).
		Where("deleted_at IS NULL")
	if classID != uuid.Nil {
		query = query.Where("class_id = ?", classID)
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func (repository *AccessRepository) schoolInScope(ctx context.Context, model any, join string, userID uuid.UUID, schoolID uuid.UUID) (bool, error) {
	query := repository.db.WithContext(ctx).Model(model)
	switch join {
	case teacherAssignmentJoin:
		query = query.Joins(join).
			Where("teacher_assignments.teacher_id = ?", userID).
			Where("teacher_assignments.active = ?", true)
	case enrollmentJoin:
		query = query.Joins(join).
			Where("enrollments.student_id = ?", userID).
			Where("enrollments.status = ?", enrollments.StatusActive)
	}
	var count int64
	err := query.Where("classes.school_id = ?", schoolID).Count(&count).Error
	return count > 0, err
}

func containsID(ids []uuid.UUID, id uuid.UUID) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}

const teacherAssignmentJoin = "JOIN teacher_assignments ON teacher_assignments.class_id = classes.id AND teacher_assignments.deleted_at IS NULL"
const enrollmentJoin = "JOIN enrollments ON enrollments.class_id = classes.id AND enrollments.deleted_at IS NULL"

func teacherScope(teacherID uuid.UUID, classID uuid.UUID, term string) classScope {
	conditions := []string{"teacher_assignments.teacher_id = ?", "teacher_assignments.active = ?"}
	args := []any{teacherID, true}
	if classID != uuid.Nil {
		conditions = append(conditions, "teacher_assignments.class_id = ?")
		args = append(args, classID)
	}
	if trimmed := strings.TrimSpace(term); trimmed != "" {
		conditions = append(conditions, "teacher_assignments.term = ?")
		args = append(args, trimmed)
	}
	return classScope{conditions: conditions, args: args}
}

func studentScope(studentID uuid.UUID, classID uuid.UUID, term string) classScope {
	conditions := []string{"enrollments.student_id = ?", "enrollments.status = ?"}
	args := []any{studentID, enrollments.StatusActive}
	if classID != uuid.Nil {
		conditions = append(conditions, "enrollments.class_id = ?")
		args = append(args, classID)
	}
	if trimmed := strings.TrimSpace(term); trimmed != "" {
		conditions = append(conditions, "enrollments.term = ?")
		args = append(args, trimmed)
	}
	return classScope{conditions: conditions, args: args}
}

type classScope struct {
	conditions []string
	args       []any
}

func (scope classScope) apply(query *gorm.DB) *gorm.DB {
	if len(scope.conditions) == 0 {
		return query
	}
	return query.Where(strings.Join(scope.conditions, " AND "), scope.args...)
}
