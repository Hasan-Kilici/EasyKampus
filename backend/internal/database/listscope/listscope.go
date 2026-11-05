// Package listscope, okul/kampüs/ders listelerini role göre sınırlayan WHERE
// parçalarını üretir. Saf (GORM'suz) olduğu için tablo/sütun adları gerçek
// şemaya karşı test edilebilir. Mantık, access_repository.go'daki HasCourseAccess /
// HasCampusAccess / HasSchoolAccess kurallarıyla aynıdır:
//
//   - superadmin: hepsi
//   - admin: personeli olduğu okulların kayıtları
//   - öğretmen: aktif olarak ders verdiği sınıfların okul/kampüs/dersleri
//   - öğrenci: aktif kayıtlı olduğu sınıfların okul/kampüs/dersleri
package listscope

import (
	"lms/backend/pkg/domain/enrollments"
	"lms/backend/pkg/domain/scopeguard"
	"lms/backend/pkg/domain/users"
)

// Filter, bir sorguya eklenecek koşuldur. Unrestricted ise hiçbir koşul eklenmez.
type Filter struct {
	Clause       string
	Args         []any
	Unrestricted bool
}

var unrestricted = Filter{Unrestricted: true}

// denyAll, tanımsız roller için hiçbir satırı döndürmez.
var denyAll = Filter{Clause: "1 = 0"}

const (
	staffSchools = "SELECT school_id FROM staffs WHERE user_id = ? AND deleted_at IS NULL"

	teacherClasses = "SELECT teacher_assignments.class_id FROM teacher_assignments WHERE teacher_assignments.teacher_id = ? AND teacher_assignments.active = ? AND teacher_assignments.deleted_at IS NULL"
	studentClasses = "SELECT enrollments.class_id FROM enrollments WHERE enrollments.student_id = ? AND enrollments.status = ? AND enrollments.deleted_at IS NULL"
)

func teacherArgs(viewer scopeguard.Viewer) []any { return []any{viewer.UserID, true} }
func studentArgs(viewer scopeguard.Viewer) []any {
	return []any{viewer.UserID, string(enrollments.StatusActive)}
}

// Schools, `schools` tablosuna eklenecek koşuldur.
func Schools(viewer *scopeguard.Viewer) Filter {
	if viewer == nil || viewer.Role == users.RoleSuperAdmin {
		return unrestricted
	}
	switch viewer.Role {
	case users.RoleAdmin:
		return Filter{Clause: "id IN (" + staffSchools + ")", Args: []any{viewer.UserID}}
	case users.RoleTeacher:
		return Filter{Clause: "id IN (SELECT school_id FROM classes WHERE deleted_at IS NULL AND id IN (" + teacherClasses + "))", Args: teacherArgs(*viewer)}
	case users.RoleStudent:
		return Filter{Clause: "id IN (SELECT school_id FROM classes WHERE deleted_at IS NULL AND id IN (" + studentClasses + "))", Args: studentArgs(*viewer)}
	default:
		return denyAll
	}
}

// Campuses, `campus` tablosuna eklenecek koşuldur.
func Campuses(viewer *scopeguard.Viewer) Filter {
	if viewer == nil || viewer.Role == users.RoleSuperAdmin {
		return unrestricted
	}
	switch viewer.Role {
	case users.RoleAdmin:
		return Filter{Clause: "school_id IN (" + staffSchools + ")", Args: []any{viewer.UserID}}
	case users.RoleTeacher:
		return Filter{Clause: "id IN (SELECT campus_id FROM classes WHERE deleted_at IS NULL AND id IN (" + teacherClasses + "))", Args: teacherArgs(*viewer)}
	case users.RoleStudent:
		return Filter{Clause: "id IN (SELECT campus_id FROM classes WHERE deleted_at IS NULL AND id IN (" + studentClasses + "))", Args: studentArgs(*viewer)}
	default:
		return denyAll
	}
}

// Courses, `courses` tablosuna eklenecek koşuldur. Öğretmen ve öğrenci yalnızca
// sınıflarına atanmış (aktif) dersleri görür; yönetici okulunun tüm derslerini görür.
func Courses(viewer *scopeguard.Viewer) Filter {
	if viewer == nil || viewer.Role == users.RoleSuperAdmin {
		return unrestricted
	}
	switch viewer.Role {
	case users.RoleAdmin:
		return Filter{Clause: "school_id IN (" + staffSchools + ")", Args: []any{viewer.UserID}}
	case users.RoleTeacher:
		return Filter{Clause: "id IN (SELECT course_id FROM class_courses WHERE active = ? AND deleted_at IS NULL AND class_id IN (" + teacherClasses + "))", Args: append([]any{true}, teacherArgs(*viewer)...)}
	case users.RoleStudent:
		return Filter{Clause: "id IN (SELECT course_id FROM class_courses WHERE active = ? AND deleted_at IS NULL AND class_id IN (" + studentClasses + "))", Args: append([]any{true}, studentArgs(*viewer)...)}
	default:
		return denyAll
	}
}
