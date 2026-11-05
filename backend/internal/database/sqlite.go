package database

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"lms/backend/internal/config"
	"lms/backend/pkg/domain/activities"
	"lms/backend/pkg/domain/assignments"
	"lms/backend/pkg/domain/attempts"
	"lms/backend/pkg/domain/attendance"
	"lms/backend/pkg/domain/campuses"
	"lms/backend/pkg/domain/classes"
	"lms/backend/pkg/domain/courses"
	"lms/backend/pkg/domain/enrollments"
	"lms/backend/pkg/domain/grades"
	"lms/backend/pkg/domain/lessons"
	"lms/backend/pkg/domain/notes"
	"lms/backend/pkg/domain/options"
	"lms/backend/pkg/domain/performance"
	"lms/backend/pkg/domain/questions"
	"lms/backend/pkg/domain/quizes"
	"lms/backend/pkg/domain/schedules"
	"lms/backend/pkg/domain/schools"
	"lms/backend/pkg/domain/sessions"
	"lms/backend/pkg/domain/terms"
	"lms/backend/pkg/domain/topics"
	"lms/backend/pkg/domain/users"
	"lms/backend/pkg/domain/video"
)

const sqliteParams = "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(1)&_pragma=cache_size(-16000)&_pragma=temp_store(MEMORY)"

// sqliteDSN appends connection parameters so every pooled connection gets the
// WAL journal mode, a busy timeout and foreign keys. Executing the PRAGMAs once
// only configures whichever pooled connection happened to run them; the rest
// would fall back to the rollback journal with no lock wait (SQLITE_BUSY) and
// could stall requests for hundreds of milliseconds.
func sqliteDSN(path string) string {
	if path == ":memory:" {
		return path
	}
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return path + separator + sqliteParams
}

func Open(cfg config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(sqliteDSN(cfg.DatabasePath)), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(4)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	sqlDB.SetConnMaxLifetime(0)
	if err := configureSQLite(db); err != nil {
		return nil, err
	}

	return db, nil
}

func configureSQLite(db *gorm.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
	}
	for _, pragma := range pragmas {
		if err := db.Exec(pragma).Error; err != nil {
			return err
		}
	}
	return nil
}

func Migrate(db *gorm.DB) error {
	models := []any{
		&users.User{},
		&sessions.Session{},
		&schools.School{},
		&schools.Staff{},
		&campuses.Campus{},
		&courses.Course{},
		&topics.Topic{},
		&classes.Class{},
		&assignments.ClassCourse{},
		&assignments.TeacherAssignment{},
		&enrollments.Enrollment{},
		&schedules.ScheduleEntry{},
		&lessons.Lesson{},
		&lessons.LessonCompletion{},
		&quizes.Quiz{},
		&questions.Question{},
		&options.Option{},
		&attendance.Attendance{},
		&grades.Grade{},
		&grades.Audit{},
		&attempts.QuizAttempt{},
		&attempts.AttemptAnswer{},
		&notes.Note{},
		&activities.Activity{},
		&activities.DailyAggregate{},
		&performance.Snapshot{},
		&performance.SnapshotCourse{},
		&performance.SnapshotTopic{},
		&performance.Problem{},
		&performance.Recommendation{},
		&terms.Term{},
		&video.Video{},
		&video.VideoCompletion{},
	}
	if err := db.AutoMigrate(models...); err != nil {
		return err
	}
	// AutoMigrate aynı adlı bir indeks varsa onu yeniden oluşturmaz; eski/yanlış
	// tanımlı UNIQUE indeksleri burada düzeltilir (bkz. index_repair.go).
	if err := RepairUniqueIndexes(db); err != nil {
		return err
	}
	return CreateIndexes(db)
}

func CreateIndexes(db *gorm.DB) error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_users_role_created ON users(role, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_classes_school_created ON classes(school_id, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_sessions_active_user ON sessions(user_id, revoked_at, expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_class_courses_class_course ON class_courses(class_id, course_id, term)",
		"CREATE INDEX IF NOT EXISTS idx_teacher_assignments_teacher ON teacher_assignments(teacher_id, class_id, course_id)",
		"CREATE INDEX IF NOT EXISTS idx_enrollments_student_status ON enrollments(student_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_enrollments_class_status ON enrollments(class_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_schedule_entries_class_term ON schedule_entries(class_id, term, day_of_week)",
		"CREATE INDEX IF NOT EXISTS idx_lessons_course_position ON lessons(course_id, position)",
		"CREATE INDEX IF NOT EXISTS idx_quizzes_class_course ON quizzes(class_id, course_id)",
		"CREATE INDEX IF NOT EXISTS idx_questions_quiz_position ON questions(quiz_id, position)",
		"CREATE INDEX IF NOT EXISTS idx_options_question_position ON options(question_id, position)",
		"CREATE INDEX IF NOT EXISTS idx_attendance_class_date ON attendances(class_id, date)",
		"CREATE INDEX IF NOT EXISTS idx_attendance_student_course ON attendances(student_id, course_id, date)",
		"CREATE INDEX IF NOT EXISTS idx_grades_class_course ON grades(class_id, course_id, term)",
		"CREATE INDEX IF NOT EXISTS idx_grades_student_course ON grades(student_id, course_id, term)",
		"CREATE INDEX IF NOT EXISTS idx_attempts_student_quiz ON quiz_attempts(student_id, quiz_id)",
		"CREATE INDEX IF NOT EXISTS idx_attempts_class_status ON quiz_attempts(class_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_notes_class_course ON notes(class_id, course_id)",
		"CREATE INDEX IF NOT EXISTS idx_notes_student ON notes(student_id)",
		"CREATE INDEX IF NOT EXISTS idx_activities_student_occurred ON activities(student_id, occurred_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_activity_daily_student_date ON daily_aggregates(student_id, date)",
		"CREATE INDEX IF NOT EXISTS idx_snapshot_courses_snapshot ON snapshot_courses(snapshot_id)",
		"CREATE INDEX IF NOT EXISTS idx_snapshot_topics_snapshot ON snapshot_topics(snapshot_id)",
		"CREATE INDEX IF NOT EXISTS idx_problems_snapshot ON problems(snapshot_id)",
		"CREATE INDEX IF NOT EXISTS idx_recommendations_snapshot ON recommendations(snapshot_id)",
	}
	for _, statement := range indexes {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func Close(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return closeSQLDB(ctx, sqlDB)
}

func closeSQLDB(ctx context.Context, db *sql.DB) error {
	closed := make(chan error, 1)
	go func() { closed <- db.Close() }()

	select {
	case err := <-closed:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
