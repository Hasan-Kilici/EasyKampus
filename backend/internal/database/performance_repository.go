package database

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"lms/backend/pkg/domain/enrollments"
	"lms/backend/pkg/domain/performance"
)

const retainedSnapshots = 10

type PerformanceRepository struct {
	db *gorm.DB
}

func NewPerformanceRepository(db *gorm.DB) *PerformanceRepository {
	return &PerformanceRepository{db: db}
}

func (repository *PerformanceRepository) SaveStudentSnapshot(ctx context.Context, snapshot *performance.Snapshot, courses []performance.SnapshotCourse, topics []performance.SnapshotTopic, problems []performance.Problem, recommendations []performance.Recommendation) error {
	snapshot.SubjectType = performance.SubjectStudent
	return repository.save(ctx, snapshot, courses, topics, problems, recommendations)
}

func (repository *PerformanceRepository) SaveTeacherSnapshot(ctx context.Context, snapshot *performance.Snapshot, courses []performance.SnapshotCourse, topics []performance.SnapshotTopic, problems []performance.Problem, recommendations []performance.Recommendation) error {
	snapshot.SubjectType = performance.SubjectTeacher
	return repository.save(ctx, snapshot, courses, topics, problems, recommendations)
}

func (repository *PerformanceRepository) save(ctx context.Context, snapshot *performance.Snapshot, courses []performance.SnapshotCourse, topics []performance.SnapshotTopic, problems []performance.Problem, recommendations []performance.Recommendation) error {
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(snapshot).Error; err != nil {
			return err
		}
		for index := range courses {
			courses[index].ID = uuid.New()
			courses[index].SnapshotID = snapshot.ID
		}
		if len(courses) > 0 {
			if err := tx.Create(&courses).Error; err != nil {
				return err
			}
		}
		for index := range topics {
			topics[index].ID = uuid.New()
			topics[index].SnapshotID = snapshot.ID
		}
		if len(topics) > 0 {
			if err := tx.Create(&topics).Error; err != nil {
				return err
			}
		}
		for index := range problems {
			problems[index].ID = uuid.New()
			problems[index].SnapshotID = snapshot.ID
		}
		if len(problems) > 0 {
			if err := tx.Create(&problems).Error; err != nil {
				return err
			}
		}
		for index := range recommendations {
			recommendations[index].ID = uuid.New()
			recommendations[index].SnapshotID = snapshot.ID
		}
		if len(recommendations) > 0 {
			if err := tx.Create(&recommendations).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return translate(err)
	}
	return translate(repository.prune(ctx, snapshot.SubjectType, snapshot.SubjectID))
}

func (repository *PerformanceRepository) prune(ctx context.Context, subjectType performance.SubjectType, subjectID uuid.UUID) error {
	return repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var stale []uuid.UUID
		err := tx.Model(&performance.Snapshot{}).
			Where("subject_type = ? AND subject_id = ?", subjectType, subjectID).
			Where("version NOT IN (?)", tx.Model(&performance.Snapshot{}).
				Select("version").
				Where("subject_type = ? AND subject_id = ?", subjectType, subjectID).
				Order("version DESC").
				Limit(retainedSnapshots)).
			Pluck("id", &stale).Error
		if err != nil {
			return err
		}
		if len(stale) == 0 {
			return nil
		}
		if err := tx.Where("snapshot_id IN ?", stale).Delete(&performance.SnapshotCourse{}).Error; err != nil {
			return err
		}
		if err := tx.Where("snapshot_id IN ?", stale).Delete(&performance.SnapshotTopic{}).Error; err != nil {
			return err
		}
		if err := tx.Where("snapshot_id IN ?", stale).Delete(&performance.Problem{}).Error; err != nil {
			return err
		}
		if err := tx.Where("snapshot_id IN ?", stale).Delete(&performance.Recommendation{}).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", stale).Delete(&performance.Snapshot{}).Error
	})
}

func (repository *PerformanceRepository) PreviousSnapshot(ctx context.Context, subjectType performance.SubjectType, subjectID uuid.UUID) (*performance.SnapshotDetail, error) {
	var snapshot performance.Snapshot
	err := repository.db.WithContext(ctx).
		Where("subject_type = ? AND subject_id = ?", subjectType, subjectID).
		Order("version DESC").
		Take(&snapshot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, translate(err)
	}
	var rows []performance.SnapshotCourse
	if err := repository.db.WithContext(ctx).Where("snapshot_id = ?", snapshot.ID).Find(&rows).Error; err != nil {
		return nil, translate(err)
	}
	detail := &performance.SnapshotDetail{OverallScore: snapshot.OverallScore, CourseAverages: make(map[string]float64, len(rows))}
	for _, row := range rows {
		detail.CourseAverages[row.CourseID.String()] = row.Score
	}
	return detail, nil
}

func (repository *PerformanceRepository) LatestStudentScore(ctx context.Context, studentID uuid.UUID) (*float64, error) {
	return repository.latestScore(ctx, performance.SubjectStudent, studentID)
}

func (repository *PerformanceRepository) LatestTeacherScore(ctx context.Context, teacherID uuid.UUID) (*float64, error) {
	return repository.latestScore(ctx, performance.SubjectTeacher, teacherID)
}

func (repository *PerformanceRepository) latestScore(ctx context.Context, subjectType performance.SubjectType, subjectID uuid.UUID) (*float64, error) {
	var snapshot performance.Snapshot
	err := repository.db.WithContext(ctx).
		Where("subject_type = ? AND subject_id = ?", subjectType, subjectID).
		Order("version DESC").
		Take(&snapshot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, translate(err)
	}
	score := snapshot.OverallScore
	return &score, nil
}

func (repository *PerformanceRepository) StudentIDsForClass(ctx context.Context, classID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := repository.db.WithContext(ctx).Model(&enrollments.Enrollment{}).
		Where("class_id = ? AND status = ?", classID, enrollments.StatusActive).
		Distinct().
		Pluck("student_id", &ids).Error
	return ids, translate(err)
}

func (repository *PerformanceRepository) StudentIDsForCourse(ctx context.Context, courseID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := repository.db.WithContext(ctx).Model(&enrollments.Enrollment{}).
		Where("course_id = ? AND status = ?", courseID, enrollments.StatusActive).
		Distinct().
		Pluck("student_id", &ids).Error
	return ids, translate(err)
}
