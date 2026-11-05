package listscope

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/scopeguard"
	"lms/backend/pkg/domain/users"
)

func viewer(role users.Role) *scopeguard.Viewer {
	return &scopeguard.Viewer{UserID: uuid.New(), Role: role}
}

func TestSuperAdminAndNilViewerAreUnrestricted(t *testing.T) {
	for name, build := range map[string]func(*scopeguard.Viewer) Filter{"schools": Schools, "campuses": Campuses, "courses": Courses} {
		if !build(nil).Unrestricted || !build(viewer(users.RoleSuperAdmin)).Unrestricted {
			t.Errorf("%s: nil viewer and superadmin must be unrestricted", name)
		}
	}
}

func TestUnknownRolesSeeNothing(t *testing.T) {
	for name, build := range map[string]func(*scopeguard.Viewer) Filter{"schools": Schools, "campuses": Campuses, "courses": Courses} {
		filter := build(viewer(users.Role("guest")))
		if filter.Unrestricted || filter.Clause != "1 = 0" {
			t.Errorf("%s: unknown role must be denied, got %+v", name, filter)
		}
	}
}

func TestEveryRestrictedFilterBindsAllItsPlaceholders(t *testing.T) {
	for name, build := range map[string]func(*scopeguard.Viewer) Filter{"schools": Schools, "campuses": Campuses, "courses": Courses} {
		for _, role := range []users.Role{users.RoleAdmin, users.RoleTeacher, users.RoleStudent} {
			filter := build(viewer(role))
			if filter.Unrestricted || filter.Clause == "" {
				t.Fatalf("%s/%s must be restricted", name, role)
			}
			if got, want := len(filter.Args), strings.Count(filter.Clause, "?"); got != want {
				t.Errorf("%s/%s: %d args for %d placeholders in %q", name, role, got, want, filter.Clause)
			}
		}
	}
}

func TestStudentAndTeacherFiltersAreBoundToTheViewer(t *testing.T) {
	student := viewer(users.RoleStudent)
	filter := Courses(student)
	if filter.Args[1] != student.UserID {
		t.Fatalf("the student's id must be bound, got %v", filter.Args)
	}
	if !strings.Contains(filter.Clause, "enrollments.status = ?") || filter.Args[2] != "active" {
		t.Fatalf("only active enrollments count: %q %v", filter.Clause, filter.Args)
	}
	teacher := viewer(users.RoleTeacher)
	if args := Courses(teacher).Args; args[1] != teacher.UserID {
		t.Fatalf("the teacher's id must be bound, got %v", args)
	}
}
