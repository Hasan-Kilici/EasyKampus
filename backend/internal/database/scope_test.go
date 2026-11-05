package database

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"lms/backend/internal/config"
	"lms/backend/pkg/domain/grades"
)

func TestGradeListEmptyScopeReturnsNoRows(t *testing.T) {
	db, err := Open(config.Config{DatabasePath: filepath.Join(t.TempDir(), "scope.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("database handle: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repository := NewGradeRepository(db)
	ctx := context.Background()

	grade := &grades.Grade{
		StudentID: uuid.New(),
		ClassID:   uuid.New(),
		CourseID:  uuid.New(),
		Term:      "2026-2027 Güz",
		Score:     90,
		Type:      grades.GradeTypeExam,
	}
	if err := repository.Upsert(ctx, grade); err != nil {
		t.Fatalf("insert grade: %v", err)
	}

	unscoped, err := repository.List(ctx, grades.ListParams{})
	if err != nil {
		t.Fatalf("unscoped list: %v", err)
	}
	if len(unscoped) != 1 {
		t.Fatalf("unscoped list: got %d rows, want 1", len(unscoped))
	}

	scoped, err := repository.List(ctx, grades.ListParams{ScopeApplied: true, ClassIDs: nil})
	if err != nil {
		t.Fatalf("scoped list: %v", err)
	}
	if len(scoped) != 0 {
		t.Fatalf("empty scope must hide rows, got %d", len(scoped))
	}
}
