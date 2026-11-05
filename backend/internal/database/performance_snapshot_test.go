package database

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"lms/backend/internal/config"
	"lms/backend/pkg/domain/classes"
	"lms/backend/pkg/domain/performance"
)

func openMigratedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := Open(config.Config{DatabasePath: filepath.Join(t.TempDir(), "snapshots.db")})
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
	return db
}

func newTestSnapshot() *performance.Snapshot {
	return &performance.Snapshot{
		SubjectID:     uuid.New(),
		Version:       1,
		OverallScore:  75,
		OverallStatus: performance.StatusGood,
		GeneratedAt:   time.Now().UTC(),
	}
}

func twoCoursesAndTopics() ([]performance.SnapshotCourse, []performance.SnapshotTopic) {
	courseA, courseB := uuid.New(), uuid.New()
	courses := []performance.SnapshotCourse{
		{CourseID: courseA, Score: 50, Status: performance.StatusNeedsAttention},
		{CourseID: courseB, Score: 100, Status: performance.StatusExcellent},
	}
	topics := []performance.SnapshotTopic{
		{CourseID: courseA, TopicID: uuid.New(), Score: 40, Status: performance.StatusWeak},
		{CourseID: courseA, TopicID: uuid.New(), Score: 60, Status: performance.StatusNeedsAttention},
	}
	return courses, topics
}

func TestSaveSnapshotAllowsMultipleCoursesAndTopics(t *testing.T) {
	db := openMigratedTestDB(t)
	repository := NewPerformanceRepository(db)
	snapshot := newTestSnapshot()
	courses, topics := twoCoursesAndTopics()

	if err := repository.SaveStudentSnapshot(context.Background(), snapshot, courses, topics, nil, nil); err != nil {
		t.Fatalf("save snapshot with two courses and two topics: %v", err)
	}
	var courseRows, topicRows int64
	db.Model(&performance.SnapshotCourse{}).Where("snapshot_id = ?", snapshot.ID).Count(&courseRows)
	db.Model(&performance.SnapshotTopic{}).Where("snapshot_id = ?", snapshot.ID).Count(&topicRows)
	if courseRows != 2 || topicRows != 2 {
		t.Fatalf("stored %d courses and %d topics, want 2 and 2", courseRows, topicRows)
	}
}

func TestSaveSnapshotStillRejectsTheSameCourseTwice(t *testing.T) {
	db := openMigratedTestDB(t)
	repository := NewPerformanceRepository(db)
	course := uuid.New()
	courses := []performance.SnapshotCourse{
		{CourseID: course, Score: 10, Status: performance.StatusWeak},
		{CourseID: course, Score: 20, Status: performance.StatusWeak},
	}
	if err := repository.SaveStudentSnapshot(context.Background(), newTestSnapshot(), courses, nil, nil, nil); err == nil {
		t.Fatal("expected a duplicate (snapshot, course) to be rejected")
	}
}

// Eski veritabanı dosyalarında yanlış (tek kolonlu) indeksler kalır; Migrate bunları onarmalı.
func TestMigrateRepairsLegacySingleColumnIndexes(t *testing.T) {
	db := openMigratedTestDB(t)
	legacy := []string{
		"DROP INDEX idx_snapshot_courses",
		"CREATE UNIQUE INDEX idx_snapshot_courses ON snapshot_courses(snapshot_id) WHERE deleted_at IS NULL",
		"DROP INDEX idx_snapshot_topics",
		"CREATE UNIQUE INDEX idx_snapshot_topics ON snapshot_topics(snapshot_id) WHERE deleted_at IS NULL",
		"DROP INDEX idx_classes_school_level_no",
		"CREATE UNIQUE INDEX idx_classes_school_level_no ON classes(school_id, no)",
	}
	for _, statement := range legacy {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}

	// Onarımdan önce hata yeniden üretilir.
	repository := NewPerformanceRepository(db)
	courses, topics := twoCoursesAndTopics()
	if err := repository.SaveStudentSnapshot(context.Background(), newTestSnapshot(), courses, topics, nil, nil); err == nil {
		t.Fatal("legacy index should reject a second course for one snapshot")
	}

	// Migrate iki kez çalışır (idempotent) ve indeksleri düzeltir.
	for run := 0; run < 2; run++ {
		if err := Migrate(db); err != nil {
			t.Fatalf("migrate run %d: %v", run, err)
		}
	}
	courses, topics = twoCoursesAndTopics()
	if err := repository.SaveStudentSnapshot(context.Background(), newTestSnapshot(), courses, topics, nil, nil); err != nil {
		t.Fatalf("after repair: %v", err)
	}
}

func TestClassesAllowSameLetterOnDifferentLevels(t *testing.T) {
	db := openMigratedTestDB(t)
	school, campus := uuid.New(), uuid.New()
	create := func(level, no string) error {
		return db.Create(&classes.Class{SchoolID: school, CampusID: campus, Name: level + "-" + no, Level: level, No: no}).Error
	}
	if err := create("1", "A"); err != nil {
		t.Fatalf("1-A: %v", err)
	}
	if err := create("2", "A"); err != nil {
		t.Fatalf("2-A must be allowed in the same school: %v", err)
	}
	if err := create("1", "A"); err == nil {
		t.Fatal("a second 1-A in the same school must be rejected")
	}
}
