package database

import (
	"context"
	"sort"

	"github.com/google/uuid"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"lms/backend/pkg/domain/attendance"
)

type AttendanceRepository struct {
	db *gorm.DB
}

func NewAttendanceRepository(db *gorm.DB) *AttendanceRepository {
	return &AttendanceRepository{db: db}
}

func (repository *AttendanceRepository) Upsert(ctx context.Context, records []attendance.Attendance) error {
	if len(records) == 0 {
		return nil
	}
	entities := make([]*attendance.Attendance, 0, len(records))
	for index := range records {
		entities = append(entities, &records[index])
	}
	return translate(repository.db.WithContext(ctx).
		Clauses(liveRowConflict(
			conflictColumns(records[0]),
			clause.AssignmentColumns([]string{"status", "note", "teacher_id", "class_id", "updated_at", "deleted_at"}),
		)).
		Create(&entities).Error)
}

func (repository *AttendanceRepository) GetByID(ctx context.Context, id uuid.UUID) (*attendance.Attendance, error) {
	var record attendance.Attendance
	err := repository.db.WithContext(ctx).First(&record, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &record, nil
}

func (repository *AttendanceRepository) Update(ctx context.Context, record *attendance.Attendance) error {
	return translate(repository.db.WithContext(ctx).Save(record).Error)
}

func (repository *AttendanceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return translate(repository.db.WithContext(ctx).Delete(&attendance.Attendance{}, "id = ?", id).Error)
}

func (repository *AttendanceRepository) List(ctx context.Context, params attendance.ListParams) ([]attendance.Attendance, error) {
	if params.Limit < 1 || params.Limit > 1000 {
		params.Limit = 200
	}
	query, empty := repository.filtered(ctx, params)
	if empty {
		return []attendance.Attendance{}, nil
	}
	var result []attendance.Attendance
	err := query.Order("date DESC, student_id ASC").Limit(params.Limit).Find(&result).Error
	return result, translate(err)
}

func (repository *AttendanceRepository) Summarize(ctx context.Context, params attendance.ListParams) (attendance.PeriodSummary, error) {
	result := attendance.PeriodSummary{Days: []attendance.DaySummary{}}
	query, empty := repository.filtered(ctx, params)
	if empty {
		return result, nil
	}
	var rows []struct {
		Date   string
		Status attendance.Status
		Amount int
	}
	err := query.Select("date, status, COUNT(*) AS amount").Group("date, status").Find(&rows).Error
	if err != nil {
		return result, translate(err)
	}
	byDate := make(map[string]*attendance.DaySummary, len(rows))
	for _, row := range rows {
		day, ok := byDate[row.Date]
		if !ok {
			day = &attendance.DaySummary{Date: row.Date}
			byDate[row.Date] = day
		}
		day.Total += row.Amount
		result.Total += row.Amount
		switch row.Status {
		case attendance.StatusPresent:
			day.Present += row.Amount
			result.Present += row.Amount
		case attendance.StatusAbsent:
			day.Absent += row.Amount
			result.Absent += row.Amount
		case attendance.StatusLate:
			day.Late += row.Amount
			result.Late += row.Amount
		case attendance.StatusExcused:
			day.Excused += row.Amount
			result.Excused += row.Amount
		}
	}
	result.Rate = result.AttendedRate()
	for _, day := range byDate {
		result.Days = append(result.Days, *day)
	}
	sort.Slice(result.Days, func(i, j int) bool { return result.Days[i].Date > result.Days[j].Date })
	return result, nil
}

func (repository *AttendanceRepository) filtered(ctx context.Context, params attendance.ListParams) (*gorm.DB, bool) {
	if params.ScopeApplied && len(params.ClassIDs) == 0 {
		return nil, true
	}
	query := repository.db.WithContext(ctx).Model(&attendance.Attendance{})
	if params.ClassID != nil {
		query = query.Where("class_id = ?", *params.ClassID)
	}
	if len(params.ClassIDs) > 0 {
		query = query.Where("class_id IN ?", params.ClassIDs)
	}
	if params.CourseID != nil {
		query = query.Where("course_id = ?", *params.CourseID)
	}
	if params.StudentID != nil {
		query = query.Where("student_id = ?", *params.StudentID)
	}
	if params.TeacherID != nil {
		query = query.Where("teacher_id = ?", *params.TeacherID)
	}
	if params.From != "" {
		query = query.Where("date >= ?", params.From)
	}
	if params.To != "" {
		query = query.Where("date <= ?", params.To)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	return query, false
}

func (repository *AttendanceRepository) StudentSummary(ctx context.Context, studentID uuid.UUID, courseID *uuid.UUID) (attendance.Summary, error) {
	var summary attendance.Summary
	query := repository.db.WithContext(ctx).Model(&attendance.Attendance{}).
		Select("status, COUNT(*) AS amount").
		Where("student_id = ?", studentID)
	if courseID != nil {
		query = query.Where("course_id = ?", *courseID)
	}
	var rows []struct {
		Status attendance.Status
		Amount int
	}
	if err := query.Group("status").Find(&rows).Error; err != nil {
		return summary, translate(err)
	}
	for _, row := range rows {
		summary.Total += row.Amount
		switch row.Status {
		case attendance.StatusPresent:
			summary.Present = row.Amount
		case attendance.StatusAbsent:
			summary.Absent = row.Amount
		case attendance.StatusLate:
			summary.Late = row.Amount
		case attendance.StatusExcused:
			summary.Excused = row.Amount
		}
	}
	summary.Rate = summary.AttendedRate()
	return summary, nil
}

func (repository *AttendanceRepository) CourseSummary(ctx context.Context, courseID uuid.UUID) (attendance.Summary, error) {
	var summary attendance.Summary
	type row struct {
		Status attendance.Status
		Amount int
	}
	var rows []row
	err := repository.db.WithContext(ctx).Model(&attendance.Attendance{}).
		Select("status, COUNT(*) AS amount").
		Where("course_id = ?", courseID).
		Group("status").
		Find(&rows).Error
	if err != nil {
		return summary, translate(err)
	}
	for _, item := range rows {
		summary.Total += item.Amount
		switch item.Status {
		case attendance.StatusPresent:
			summary.Present = item.Amount
		case attendance.StatusAbsent:
			summary.Absent = item.Amount
		case attendance.StatusLate:
			summary.Late = item.Amount
		case attendance.StatusExcused:
			summary.Excused = item.Amount
		}
	}
	summary.Rate = summary.AttendedRate()
	return summary, translate(err)
}

func (repository *AttendanceRepository) CountByTeacher(ctx context.Context, teacherID uuid.UUID, classIDs []uuid.UUID) (int64, error) {
	if len(classIDs) == 0 {
		return 0, nil
	}
	var count int64
	err := repository.db.WithContext(ctx).Model(&attendance.Attendance{}).
		Where("teacher_id = ? AND class_id IN ?", teacherID, classIDs).
		Count(&count).Error
	return count, translate(err)
}

func conflictColumns(_ attendance.Attendance) []string {
	return []string{"student_id", "course_id", "date"}
}
