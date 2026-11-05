package scopeguard

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/users"
)

func TestStudentSubjectForcesTheStudentsOwnID(t *testing.T) {
	self, other := uuid.New(), uuid.New()

	got, err := StudentSubject(users.RoleStudent, self, nil)
	if err != nil || got == nil || *got != self {
		t.Fatalf("no filter requested: want self, got %v %v", got, err)
	}
	got, err = StudentSubject(users.RoleStudent, self, &self)
	if err != nil || *got != self {
		t.Fatalf("own id requested: want self, got %v %v", got, err)
	}
	if _, err = StudentSubject(users.RoleStudent, self, &other); !errors.Is(err, core.ErrForbidden) {
		t.Fatalf("another student's id must be forbidden, got %v", err)
	}
}

func TestStudentSubjectLeavesStaffRequestsAlone(t *testing.T) {
	self, other := uuid.New(), uuid.New()
	for _, role := range []users.Role{users.RoleTeacher, users.RoleAdmin, users.RoleSuperAdmin} {
		if got, err := StudentSubject(role, self, nil); err != nil || got != nil {
			t.Errorf("%s: no filter should stay nil, got %v %v", role, got, err)
		}
		if got, err := StudentSubject(role, self, &other); err != nil || *got != other {
			t.Errorf("%s: requested filter should pass through, got %v %v", role, got, err)
		}
	}
}

func TestStudentSubjectDoesNotAliasTheCallersPointer(t *testing.T) {
	self := uuid.New()
	got, _ := StudentSubject(users.RoleStudent, self, nil)
	*got = uuid.New()
	if self == *got {
		t.Fatal("the returned pointer must not alias the caller's value")
	}
}

func TestSharedNotesOnly(t *testing.T) {
	if !SharedNotesOnly(users.RoleStudent, false) {
		t.Fatal("students must always get shared notes only, even without the query flag")
	}
	if SharedNotesOnly(users.RoleTeacher, false) {
		t.Fatal("teachers see all notes unless they ask for shared ones")
	}
	if !SharedNotesOnly(users.RoleTeacher, true) {
		t.Fatal("an explicit shared filter is honoured")
	}
}

func TestMayReadNote(t *testing.T) {
	if MayReadNote(users.RoleStudent, false) {
		t.Fatal("a private teacher note must not be readable by the student")
	}
	if !MayReadNote(users.RoleStudent, true) || !MayReadNote(users.RoleTeacher, false) {
		t.Fatal("shared notes are readable by students; staff read everything")
	}
}
