package database

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"lms/backend/pkg/domain/activities"
	"lms/backend/pkg/domain/assignments"
	"lms/backend/pkg/domain/attempts"
	"lms/backend/pkg/domain/attendance"
	"lms/backend/pkg/domain/classes"
	"lms/backend/pkg/domain/courses"
	"lms/backend/pkg/domain/enrollments"
	"lms/backend/pkg/domain/lessons"
	"lms/backend/pkg/domain/performance"
	"lms/backend/pkg/domain/questions"
	"lms/backend/pkg/domain/quizes"
	"lms/backend/pkg/domain/topics"
	"lms/backend/pkg/domain/users"
)

type PerformanceInputBuilder struct {
	db *gorm.DB
}

func NewPerformanceInputBuilder(db *gorm.DB) *PerformanceInputBuilder {
	return &PerformanceInputBuilder{db: db}
}

func (builder *PerformanceInputBuilder) BuildStudentInput(ctx context.Context, studentID uuid.UUID) (*performance.StudentInput, error) {
	student, err := builder.studentRef(ctx, studentID)
	if err != nil {
		return nil, err
	}
	classIDs, err := builder.classIDsForStudent(ctx, studentID)
	if err != nil {
		return nil, err
	}
	classRefs, courseIDs, err := builder.classRefs(ctx, classIDs)
	if err != nil {
		return nil, err
	}
	courseRefs, err := builder.courseRefs(ctx, courseIDs)
	if err != nil {
		return nil, err
	}
	quizResults, err := builder.quizResults(ctx, studentID)
	if err != nil {
		return nil, err
	}
	attendanceInput, err := builder.attendanceInput(ctx, studentID)
	if err != nil {
		return nil, err
	}
	activityInput, err := builder.activityInput(ctx, studentID)
	if err != nil {
		return nil, err
	}
	lessonInput, err := builder.lessonInput(ctx, studentID, courseIDs)
	if err != nil {
		return nil, err
	}
	previous, err := builder.previousScore(ctx, performance.SubjectStudent, studentID)
	if err != nil {
		return nil, err
	}
	return &performance.StudentInput{
		Version:       performance.ContractVersion,
		Student:       student,
		Classes:       classRefs,
		Courses:       courseRefs,
		QuizResults:   quizResults,
		Attendance:    attendanceInput,
		Activity:      activityInput,
		Lessons:       lessonInput,
		PreviousScore: previous,
		GeneratedAt:   time.Now().UTC(),
	}, nil
}

func (builder *PerformanceInputBuilder) BuildTeacherInput(ctx context.Context, teacherID uuid.UUID) (*performance.TeacherInput, error) {
	teacher, err := builder.teacherRef(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	classRefs, err := builder.teacherClasses(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	byClass, pairs, err := builder.teacherScope(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	courseIDs := make([]uuid.UUID, 0, len(pairs))
	seen := make(map[uuid.UUID]struct{}, len(pairs))
	for _, pair := range pairs {
		if _, ok := seen[pair.CourseID]; ok {
			continue
		}
		seen[pair.CourseID] = struct{}{}
		courseIDs = append(courseIDs, pair.CourseID)
	}
	previous, err := builder.previousSnapshot(ctx, performance.SubjectTeacher, teacherID)
	if err != nil {
		return nil, err
	}
	courseRefs, err := builder.teacherCourses(ctx, courseIDs, previous)
	if err != nil {
		return nil, err
	}
	content, err := builder.teacherContent(ctx, teacherID, courseIDs, pairs)
	if err != nil {
		return nil, err
	}
	quizRefs, err := builder.teacherQuizzes(ctx, pairs, byClass)
	if err != nil {
		return nil, err
	}
	studentRefs, err := builder.teacherStudents(ctx, classRefs)
	if err != nil {
		return nil, err
	}
	attendanceInput, err := builder.teacherAttendance(ctx, classRefs)
	if err != nil {
		return nil, err
	}
	activityInput, err := builder.teacherActivity(ctx, teacherID, courseIDs, pairs)
	if err != nil {
		return nil, err
	}
	previousScore := previousScore(previous)
	return &performance.TeacherInput{
		Version:       performance.ContractVersion,
		Teacher:       teacher,
		Classes:       classRefs,
		Courses:       courseRefs,
		Content:       content,
		Quizzes:       quizRefs,
		Students:      studentRefs,
		Attendance:    attendanceInput,
		Activity:      activityInput,
		PreviousScore: previousScore,
		GeneratedAt:   time.Now().UTC(),
	}, nil
}

func (builder *PerformanceInputBuilder) previousScore(ctx context.Context, subjectType performance.SubjectType, subjectID uuid.UUID) (*float64, error) {
	return NewPerformanceRepository(builder.db).latestScore(ctx, subjectType, subjectID)
}

func (builder *PerformanceInputBuilder) previousSnapshot(ctx context.Context, subjectType performance.SubjectType, subjectID uuid.UUID) (*performance.SnapshotDetail, error) {
	detail, err := NewPerformanceRepository(builder.db).PreviousSnapshot(ctx, subjectType, subjectID)
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func (builder *PerformanceInputBuilder) studentRef(ctx context.Context, studentID uuid.UUID) (performance.StudentRef, error) {
	var user users.User
	if err := builder.db.WithContext(ctx).First(&user, "id = ?", studentID).Error; err != nil {
		return performance.StudentRef{}, translate(err)
	}
	classIDs, err := builder.classIDsForStudent(ctx, studentID)
	if err != nil {
		return performance.StudentRef{}, err
	}
	ids := make([]string, 0, len(classIDs))
	for _, classID := range classIDs {
		ids = append(ids, classID.String())
	}
	return performance.StudentRef{ID: user.ID.String(), Name: fullName(user), ClassIDs: ids}, nil
}

func (builder *PerformanceInputBuilder) teacherRef(ctx context.Context, teacherID uuid.UUID) (performance.TeacherRef, error) {
	var user users.User
	if err := builder.db.WithContext(ctx).First(&user, "id = ?", teacherID).Error; err != nil {
		return performance.TeacherRef{}, translate(err)
	}
	classIDs, err := NewAssignmentRepository(builder.db).ClassIDsForTeacher(ctx, teacherID, uuid.Nil)
	if err != nil {
		return performance.TeacherRef{}, err
	}
	schoolIDs, err := builder.schoolIDsForClasses(ctx, classIDs)
	if err != nil {
		return performance.TeacherRef{}, err
	}
	ids := make([]string, 0, len(schoolIDs))
	for _, schoolID := range schoolIDs {
		ids = append(ids, schoolID.String())
	}
	return performance.TeacherRef{ID: user.ID.String(), Name: fullName(user), SchoolIDs: ids}, nil
}

func (builder *PerformanceInputBuilder) schoolIDsForClasses(ctx context.Context, classIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(classIDs) == 0 {
		return nil, nil
	}
	var ids []uuid.UUID
	err := builder.db.WithContext(ctx).Model(&classes.Class{}).
		Where("id IN ?", classIDs).
		Distinct().
		Pluck("school_id", &ids).Error
	return ids, translate(err)
}

func (builder *PerformanceInputBuilder) classIDsForStudent(ctx context.Context, studentID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := builder.db.WithContext(ctx).Model(&enrollments.Enrollment{}).
		Where("student_id = ? AND status = ?", studentID, enrollments.StatusActive).
		Distinct().
		Pluck("class_id", &ids).Error
	return ids, translate(err)
}

func (builder *PerformanceInputBuilder) classRefs(ctx context.Context, classIDs []uuid.UUID) ([]performance.ClassRef, []uuid.UUID, error) {
	if len(classIDs) == 0 {
		return []performance.ClassRef{}, nil, nil
	}
	var rows []classes.Class
	if err := builder.db.WithContext(ctx).Where("id IN ?", classIDs).Find(&rows).Error; err != nil {
		return nil, nil, translate(err)
	}
	var links []assignments.ClassCourse
	if err := builder.db.WithContext(ctx).Where("class_id IN ? AND active = ?", classIDs, true).Find(&links).Error; err != nil {
		return nil, nil, translate(err)
	}
	coursesByClass := make(map[uuid.UUID][]string, len(classIDs))
	uniqueCourses := make(map[uuid.UUID]struct{}, len(links))
	for _, link := range links {
		coursesByClass[link.ClassID] = append(coursesByClass[link.ClassID], link.CourseID.String())
		uniqueCourses[link.CourseID] = struct{}{}
	}
	courseIDs := make([]uuid.UUID, 0, len(uniqueCourses))
	for courseID := range uniqueCourses {
		courseIDs = append(courseIDs, courseID)
	}
	result := make([]performance.ClassRef, 0, len(rows))
	for _, row := range rows {
		ids := coursesByClass[row.ID]
		if ids == nil {
			ids = []string{}
		}
		result = append(result, performance.ClassRef{
			ID:        row.ID.String(),
			Name:      row.Name,
			Level:     row.Level,
			SchoolID:  row.SchoolID.String(),
			CourseIDs: ids,
		})
	}
	return result, courseIDs, nil
}

func (builder *PerformanceInputBuilder) courseRefs(ctx context.Context, courseIDs []uuid.UUID) ([]performance.CourseRef, error) {
	if len(courseIDs) == 0 {
		return []performance.CourseRef{}, nil
	}
	var rows []courses.Course
	if err := builder.db.WithContext(ctx).Where("id IN ?", courseIDs).Find(&rows).Error; err != nil {
		return nil, translate(err)
	}
	var links []assignments.ClassCourse
	if err := builder.db.WithContext(ctx).Where("course_id IN ? AND active = ?", courseIDs, true).Find(&links).Error; err != nil {
		return nil, translate(err)
	}
	classesByCourse := make(map[uuid.UUID][]string, len(courseIDs))
	for _, link := range links {
		classesByCourse[link.CourseID] = append(classesByCourse[link.CourseID], link.ClassID.String())
	}
	var topicRows []topics.Topic
	if err := builder.db.WithContext(ctx).
		Where("course_id IN ? AND active = ?", courseIDs, true).
		Order("weight DESC").
		Find(&topicRows).Error; err != nil {
		return nil, translate(err)
	}
	topicsByCourse := make(map[uuid.UUID][]performance.TopicRef, len(courseIDs))
	for _, topic := range topicRows {
		var parent *string
		if topic.ParentID != nil {
			value := topic.ParentID.String()
			parent = &value
		}
		topicsByCourse[topic.CourseID] = append(topicsByCourse[topic.CourseID], performance.TopicRef{
			ID:       topic.ID.String(),
			Name:     topic.Name,
			Code:     topic.Code,
			ParentID: parent,
			Weight:   topic.Weight,

			Description: topic.Description,
		})
	}
	result := make([]performance.CourseRef, 0, len(rows))
	for _, row := range rows {
		ids := classesByCourse[row.ID]
		if ids == nil {
			ids = []string{}
		}
		courseTopics := topicsByCourse[row.ID]
		if courseTopics == nil {
			courseTopics = []performance.TopicRef{}
		}
		result = append(result, performance.CourseRef{
			ID:          row.ID.String(),
			Name:        row.Name,
			SubjectArea: row.SubjectArea,
			ClassIDs:    ids,
			Topics:      courseTopics,

			Department:         row.Department,
			DepartmentKeywords: row.DepartmentKeywords,
		})
	}
	return result, nil
}

func (builder *PerformanceInputBuilder) quizResults(ctx context.Context, studentID uuid.UUID) ([]performance.QuizResult, error) {
	attemptsList, err := NewAttemptRepository(builder.db).CompletedAttempts(ctx, studentID)
	if err != nil {
		return nil, err
	}
	if len(attemptsList) == 0 {
		return []performance.QuizResult{}, nil
	}
	quizIDs := make([]uuid.UUID, 0, len(attemptsList))
	for _, attempt := range attemptsList {
		quizIDs = append(quizIDs, attempt.QuizID)
	}
	var quizzes []quizes.Quiz
	if err := builder.db.WithContext(ctx).Where("id IN ?", quizIDs).Find(&quizzes).Error; err != nil {
		return nil, translate(err)
	}
	names := make(map[uuid.UUID]string, len(quizzes))
	for _, quiz := range quizzes {
		names[quiz.ID] = quiz.Name
	}
	result := make([]performance.QuizResult, 0, len(attemptsList))
	for _, attempt := range attemptsList {
		completedAt := attempt.StartedAt
		if attempt.CompletedAt != nil {
			completedAt = *attempt.CompletedAt
		}
		answers := make([]performance.AnswerInput, 0, len(attempt.Answers))
		for _, answer := range attempt.Answers {
			var topicID *string
			if answer.TopicID != nil {
				value := answer.TopicID.String()
				topicID = &value
			}
			answers = append(answers, performance.AnswerInput{
				QuestionID: answer.QuestionID.String(),
				TopicID:    topicID,
				Correct:    answer.Correct,
				Points:     answer.Points,
			})
		}
		result = append(result, performance.QuizResult{
			AttemptID:       attempt.ID.String(),
			QuizID:          attempt.QuizID.String(),
			CourseID:        attempt.CourseID.String(),
			ClassID:         attempt.ClassID.String(),
			Name:            names[attempt.QuizID],
			Score:           attempt.Score,
			MaxScore:        attempt.MaxScore,
			StartedAt:       attempt.StartedAt,
			CompletedAt:     completedAt,
			DurationSeconds: attempt.DurationSeconds,
			Answers:         answers,
		})
	}
	return result, nil
}

func (builder *PerformanceInputBuilder) attendanceInput(ctx context.Context, studentID uuid.UUID) (performance.AttendanceInput, error) {
	summary, err := NewAttendanceRepository(builder.db).StudentSummary(ctx, studentID, nil)
	if err != nil {
		return performance.AttendanceInput{}, err
	}
	perCourse, err := builder.attendancePerCourse(ctx, studentID)
	if err != nil {
		return performance.AttendanceInput{}, err
	}
	return performance.AttendanceInput{
		Total:     summary.Total,
		Present:   summary.Present,
		Absent:    summary.Absent,
		Late:      summary.Late,
		Excused:   summary.Excused,
		Rate:      summary.Rate,
		PerCourse: perCourse,
	}, nil
}

func (builder *PerformanceInputBuilder) attendancePerCourse(ctx context.Context, studentID uuid.UUID) ([]performance.AttendanceCourse, error) {
	var rows []attendanceStatusRow
	err := builder.db.WithContext(ctx).Model(&attendance.Attendance{}).
		Select("course_id, status, COUNT(*) AS amount").
		Where("student_id = ?", studentID).
		Group("course_id, status").
		Find(&rows).Error
	if err != nil {
		return nil, translate(err)
	}
	totals := make(map[uuid.UUID]*performance.AttendanceCourse, len(rows))
	order := make([]uuid.UUID, 0, len(rows))
	for _, item := range rows {
		entry, ok := totals[item.CourseID]
		if !ok {
			entry = &performance.AttendanceCourse{CourseID: item.CourseID.String()}
			totals[item.CourseID] = entry
			order = append(order, item.CourseID)
		}
		entry.Total += item.Amount
		switch item.Status {
		case attendance.StatusPresent:
			entry.Present = item.Amount
		case attendance.StatusAbsent:
			entry.Absent = item.Amount
		case attendance.StatusLate:
			entry.Late = item.Amount
		case attendance.StatusExcused:
			entry.Excused = item.Amount
		}
	}
	result := make([]performance.AttendanceCourse, 0, len(order))
	for _, courseID := range order {
		entry := totals[courseID]
		entry.Rate = attendance.Summary{
			Total:   entry.Total,
			Present: entry.Present,
			Absent:  entry.Absent,
			Late:    entry.Late,
			Excused: entry.Excused,
		}.AttendedRate()
		result = append(result, *entry)
	}
	return result, nil
}

func (builder *PerformanceInputBuilder) activityInput(ctx context.Context, studentID uuid.UUID) (performance.ActivityInput, error) {
	series, err := NewActivityRepository(builder.db).DailySeries(ctx, studentID, nil)
	if err != nil {
		return performance.ActivityInput{}, err
	}
	input := performance.ActivityInput{ActiveDays: len(series)}
	daily := make([]performance.DailyActivity, 0, len(series))
	for _, day := range series {
		input.TotalEvents += day.Events
		input.LessonViews += day.LessonViews
		input.LessonCompletions += day.LessonCompletions
		input.QuizStarted += day.QuizStarted
		input.QuizCompleted += day.QuizCompleted
		input.StudyMinutes += day.StudyMinutes
		input.Logins += day.Logins
		daily = append(daily, performance.DailyActivity{
			Date:             day.Date,
			Events:           day.Events,
			StudyMinutes:     day.StudyMinutes,
			LessonViews:      day.LessonViews,
			LessonCompletion: day.LessonCompletions,
			QuizCompletions:  day.QuizCompleted,
		})
	}
	input.Daily = daily
	return input, nil
}

func (builder *PerformanceInputBuilder) lessonInput(ctx context.Context, studentID uuid.UUID, courseIDs []uuid.UUID) (performance.LessonInput, error) {
	repository := NewLessonRepository(builder.db)
	lessonIDs, err := repository.PublishedLessonIDs(ctx, courseIDs)
	if err != nil {
		return performance.LessonInput{}, err
	}
	completions, err := repository.ListCompletions(ctx, studentID, nil)
	if err != nil {
		return performance.LessonInput{}, err
	}
	published := make(map[uuid.UUID]struct{}, len(lessonIDs))
	for _, lessonID := range lessonIDs {
		published[lessonID] = struct{}{}
	}
	perCourseTotal, err := builder.lessonCountsByCourse(ctx, lessonIDs)
	if err != nil {
		return performance.LessonInput{}, err
	}
	perCourseComplete := make(map[uuid.UUID]int, len(courseIDs))
	for _, completion := range completions {
		if _, ok := published[completion.LessonID]; !ok {
			continue
		}
		perCourseComplete[completion.CourseID]++
	}
	input := performance.LessonInput{PerCourse: make([]performance.CourseCompletion, 0, len(courseIDs))}
	for _, courseID := range courseIDs {
		total := perCourseTotal[courseID]
		done := perCourseComplete[courseID]
		if done > total {
			done = total
		}
		input.Total += total
		input.Completed += done
		input.PerCourse = append(input.PerCourse, performance.CourseCompletion{
			CourseID: courseID.String(),
			Total:    total,
			Complete: done,
			Rate:     percent(done, total),
		})
	}
	input.Rate = percent(input.Completed, input.Total)
	return input, nil
}

func (builder *PerformanceInputBuilder) lessonCountsByCourse(ctx context.Context, lessonIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	if len(lessonIDs) == 0 {
		return map[uuid.UUID]int{}, nil
	}
	type row struct {
		CourseID uuid.UUID
		Amount   int64
	}
	var rows []row
	err := builder.db.WithContext(ctx).Model(&lessons.Lesson{}).
		Select("course_id, COUNT(*) AS amount").
		Where("id IN ?", lessonIDs).
		Group("course_id").
		Find(&rows).Error
	if err != nil {
		return nil, translate(err)
	}
	result := make(map[uuid.UUID]int, len(rows))
	for _, item := range rows {
		result[item.CourseID] = int(item.Amount)
	}
	return result, nil
}

type teacherScopePair struct {
	ClassID  uuid.UUID
	CourseID uuid.UUID
}

func (builder *PerformanceInputBuilder) teacherScope(ctx context.Context, teacherID uuid.UUID) (map[uuid.UUID][]uuid.UUID, []teacherScopePair, error) {
	byCourse, err := NewAssignmentRepository(builder.db).ClassCourseIDsForTeacher(ctx, teacherID)
	if err != nil {
		return nil, nil, err
	}
	byClass := make(map[uuid.UUID][]uuid.UUID, len(byCourse))
	pairs := make([]teacherScopePair, 0, len(byCourse))
	for courseID, classIDs := range byCourse {
		for _, classID := range classIDs {
			byClass[classID] = appendUniqueID(byClass[classID], courseID)
			pairs = append(pairs, teacherScopePair{ClassID: classID, CourseID: courseID})
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].ClassID == pairs[j].ClassID {
			return pairs[i].CourseID.String() < pairs[j].CourseID.String()
		}
		return pairs[i].ClassID.String() < pairs[j].ClassID.String()
	})
	return byClass, pairs, nil
}

func (builder *PerformanceInputBuilder) teacherClasses(ctx context.Context, teacherID uuid.UUID) ([]performance.TeacherClass, error) {
	byClass, _, err := builder.teacherScope(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	if len(byClass) == 0 {
		return []performance.TeacherClass{}, nil
	}
	classIDs := make([]uuid.UUID, 0, len(byClass))
	for classID := range byClass {
		classIDs = append(classIDs, classID)
	}
	var rows []classes.Class
	if err := builder.db.WithContext(ctx).Where("id IN ?", classIDs).Find(&rows).Error; err != nil {
		return nil, translate(err)
	}
	var enrolled []enrollments.Enrollment
	if err := builder.db.WithContext(ctx).Where("class_id IN ? AND status = ?", classIDs, enrollments.StatusActive).Find(&enrolled).Error; err != nil {
		return nil, translate(err)
	}
	studentsByClass := make(map[uuid.UUID]map[uuid.UUID]struct{}, len(classIDs))
	for _, enrollment := range enrolled {
		if studentsByClass[enrollment.ClassID] == nil {
			studentsByClass[enrollment.ClassID] = make(map[uuid.UUID]struct{})
		}
		studentsByClass[enrollment.ClassID][enrollment.StudentID] = struct{}{}
	}
	lessonCounts, err := builder.lessonCountsByScope(ctx, byClass)
	if err != nil {
		return nil, err
	}
	quizCounts, err := builder.quizCountsByScope(ctx, byClass)
	if err != nil {
		return nil, err
	}
	result := make([]performance.TeacherClass, 0, len(rows))
	for _, row := range rows {
		courseIDs := make([]string, 0, len(byClass[row.ID]))
		for _, courseID := range byClass[row.ID] {
			courseIDs = append(courseIDs, courseID.String())
		}
		result = append(result, performance.TeacherClass{
			ClassID:      row.ID.String(),
			Name:         row.Name,
			SchoolID:     row.SchoolID.String(),
			StudentCount: len(studentsByClass[row.ID]),
			CourseIDs:    courseIDs,
			LessonCount:  lessonCounts[row.ID],
			QuizCount:    quizCounts[row.ID],
		})
	}
	return result, nil
}

func (builder *PerformanceInputBuilder) lessonCountsByScope(ctx context.Context, byClass map[uuid.UUID][]uuid.UUID) (map[uuid.UUID]int, error) {
	type row struct {
		ClassID uuid.UUID
		Amount  int64
	}
	result := make(map[uuid.UUID]int, len(byClass))
	for classID, courseIDs := range byClass {
		if len(courseIDs) == 0 {
			continue
		}
		var item row
		err := builder.db.WithContext(ctx).Model(&lessons.Lesson{}).
			Select("COUNT(*) AS amount").
			Where("course_id IN ?", courseIDs).
			Scan(&item).Error
		if err != nil {
			return nil, translate(err)
		}
		result[classID] = int(item.Amount)
	}
	return result, nil
}

func (builder *PerformanceInputBuilder) quizCountsByScope(ctx context.Context, byClass map[uuid.UUID][]uuid.UUID) (map[uuid.UUID]int, error) {
	type row struct {
		Amount int64
	}
	result := make(map[uuid.UUID]int, len(byClass))
	for classID, courseIDs := range byClass {
		if len(courseIDs) == 0 {
			continue
		}
		var item row
		err := builder.db.WithContext(ctx).Model(&quizes.Quiz{}).
			Select("COUNT(*) AS amount").
			Where("class_id = ? AND course_id IN ?", classID, courseIDs).
			Scan(&item).Error
		if err != nil {
			return nil, translate(err)
		}
		result[classID] = int(item.Amount)
	}
	return result, nil
}

func (builder *PerformanceInputBuilder) teacherCourses(ctx context.Context, courseIDs []uuid.UUID, previous *performance.SnapshotDetail) ([]performance.TeacherCourse, error) {
	if len(courseIDs) == 0 {
		return []performance.TeacherCourse{}, nil
	}
	var rows []courses.Course
	if err := builder.db.WithContext(ctx).Where("id IN ?", courseIDs).Find(&rows).Error; err != nil {
		return nil, translate(err)
	}
	var links []assignments.ClassCourse
	if err := builder.db.WithContext(ctx).Where("course_id IN ? AND active = ?", courseIDs, true).Find(&links).Error; err != nil {
		return nil, translate(err)
	}
	classesByCourse := make(map[uuid.UUID][]string, len(courseIDs))
	for _, link := range links {
		classesByCourse[link.CourseID] = append(classesByCourse[link.CourseID], link.ClassID.String())
	}
	averages, err := builder.averageScoresByCourse(ctx, courseIDs)
	if err != nil {
		return nil, err
	}
	studentsByCourse, err := builder.studentCountsByCourse(ctx, courseIDs)
	if err != nil {
		return nil, err
	}
	result := make([]performance.TeacherCourse, 0, len(rows))
	for _, row := range rows {
		ids := classesByCourse[row.ID]
		if ids == nil {
			ids = []string{}
		}
		entry := performance.TeacherCourse{
			CourseID:            row.ID.String(),
			Name:                row.Name,
			SubjectArea:         row.SubjectArea,
			ClassIDs:            ids,
			StudentCount:        studentsByCourse[row.ID],
			AverageStudentScore: averages[row.ID],
		}
		if previous != nil {
			if value, ok := previous.CourseAverages[row.ID.String()]; ok {
				entry.PreviousAverage = value
			}
		}
		result = append(result, entry)
	}
	return result, nil
}

func (builder *PerformanceInputBuilder) averageScoresByCourse(ctx context.Context, courseIDs []uuid.UUID) (map[uuid.UUID]float64, error) {
	var rows []attemptAverageRow
	err := builder.db.WithContext(ctx).Model(&attempts.QuizAttempt{}).
		Select("course_id AS scope_id, AVG(CASE WHEN max_score > 0 THEN score * 100.0 / max_score ELSE 0 END) AS average").
		Where("course_id IN ? AND status = ?", courseIDs, attempts.StatusCompleted).
		Group("course_id").
		Find(&rows).Error
	if err != nil {
		return nil, translate(err)
	}
	result := make(map[uuid.UUID]float64, len(rows))
	for _, item := range rows {
		result[item.ScopeID] = roundPercent(item.Average)
	}
	return result, nil
}

func (builder *PerformanceInputBuilder) studentCountsByCourse(ctx context.Context, courseIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	type row struct {
		CourseID uuid.UUID
		Amount   int64
	}
	var rows []row
	err := builder.db.WithContext(ctx).Model(&enrollments.Enrollment{}).
		Select("course_id, COUNT(DISTINCT student_id) AS amount").
		Where("course_id IN ? AND status = ?", courseIDs, enrollments.StatusActive).
		Group("course_id").
		Find(&rows).Error
	if err != nil {
		return nil, translate(err)
	}
	result := make(map[uuid.UUID]int, len(rows))
	for _, item := range rows {
		result[item.CourseID] = int(item.Amount)
	}
	return result, nil
}

func (builder *PerformanceInputBuilder) teacherContent(ctx context.Context, teacherID uuid.UUID, courseIDs []uuid.UUID, pairs []teacherScopePair) (performance.TeacherContent, error) {
	lessonRepository := NewLessonRepository(builder.db)
	quizRepository := NewQuizRepository(builder.db)
	questionRepository := NewQuestionRepository(builder.db)

	lessonStates, err := lessonRepository.CountByCreatorAndState(ctx, teacherID)
	if err != nil {
		return performance.TeacherContent{}, err
	}
	quizStates, err := quizRepository.CountByCreatorAndState(ctx, teacherID)
	if err != nil {
		return performance.TeacherContent{}, err
	}
	questionCount, err := questionRepository.CountByCreator(ctx, teacherID)
	if err != nil {
		return performance.TeacherContent{}, err
	}
	lessonsInScope, err := lessonRepository.CountInScope(ctx, courseIDs)
	if err != nil {
		return performance.TeacherContent{}, err
	}
	quizzesInScope, err := builder.quizCountInScope(ctx, pairs)
	if err != nil {
		return performance.TeacherContent{}, err
	}
	return performance.TeacherContent{
		LessonsCreated:   sumCounts(lessonStates),
		LessonsApproved:  int(lessonStates["approved"]),
		LessonsRejected:  int(lessonStates["rejected"]),
		LessonsPending:   int(lessonStates["pending"]),
		QuizzesCreated:   sumCounts(quizStates),
		QuizzesApproved:  int(quizStates["approved"]),
		QuizzesRejected:  int(quizStates["rejected"]),
		QuizzesPending:   int(quizStates["pending"]),
		QuestionsCreated: int(questionCount),
		LessonsInScope:   int(lessonsInScope),
		QuizzesInScope:   quizzesInScope,
	}, nil
}

func (builder *PerformanceInputBuilder) quizCountInScope(ctx context.Context, pairs []teacherScopePair) (int, error) {
	if len(pairs) == 0 {
		return 0, nil
	}
	var count int64
	condition, args := teacherScopeArgs("quizzes", pairs)
	err := builder.db.WithContext(ctx).Model(&quizes.Quiz{}).
		Where(condition, args...).
		Count(&count).Error
	return int(count), translate(err)
}

func (builder *PerformanceInputBuilder) questionCountInScope(ctx context.Context, pairs []teacherScopePair) (int, error) {
	if len(pairs) == 0 {
		return 0, nil
	}
	condition, args := teacherScopeArgs("quizzes", pairs)
	var count int64
	err := builder.db.WithContext(ctx).Model(&questions.Question{}).
		Where("questions.quiz_id IN (SELECT quizzes.id FROM quizzes WHERE "+condition+")", args...).
		Count(&count).Error
	return int(count), translate(err)
}

func (builder *PerformanceInputBuilder) teacherQuizzes(ctx context.Context, pairs []teacherScopePair, byClass map[uuid.UUID][]uuid.UUID) ([]performance.TeacherQuiz, error) {
	if len(pairs) == 0 {
		return []performance.TeacherQuiz{}, nil
	}
	var quizzes []quizes.Quiz
	condition, args := teacherScopeArgs("quizzes", pairs)
	if err := builder.db.WithContext(ctx).Where(condition, args...).Find(&quizzes).Error; err != nil {
		return nil, translate(err)
	}
	if len(quizzes) == 0 {
		return []performance.TeacherQuiz{}, nil
	}
	quizIDs := make([]uuid.UUID, 0, len(quizzes))
	for _, quiz := range quizzes {
		quizIDs = append(quizIDs, quiz.ID)
	}
	var rows []quizStatRow
	err := builder.db.WithContext(ctx).Model(&attempts.QuizAttempt{}).
		Select("quiz_id AS scope_id, COUNT(*) AS attempts, AVG(CASE WHEN max_score > 0 THEN score * 100.0 / max_score ELSE 0 END) AS average_score, AVG(CASE WHEN max_score > 0 AND score * 100.0 / max_score >= 50 THEN 1.0 ELSE 0.0 END) AS pass_rate").
		Where("quiz_id IN ? AND status = ?", quizIDs, attempts.StatusCompleted).
		Group("quiz_id").
		Find(&rows).Error
	if err != nil {
		return nil, translate(err)
	}
	stats := make(map[uuid.UUID]quizStatRow, len(rows))
	for _, item := range rows {
		stats[item.ScopeID] = item
	}
	studentCounts, err := builder.studentCountsByClass(ctx, mapKeys(byClass))
	if err != nil {
		return nil, err
	}
	result := make([]performance.TeacherQuiz, 0, len(quizzes))
	for _, quiz := range quizzes {
		stat := stats[quiz.ID]
		result = append(result, performance.TeacherQuiz{
			QuizID:          quiz.ID.String(),
			Name:            quiz.Name,
			CourseID:        quiz.CourseID.String(),
			ClassID:         quiz.ClassID.String(),
			Attempts:        stat.Attempts,
			StudentCount:    studentCounts[quiz.ClassID],
			AverageScore:    roundPercent(stat.AverageScore),
			PassRate:        roundPercent(stat.PassRate * 100),
			DurationMinutes: quiz.DurationMinutes,
		})
	}
	return result, nil
}

func (builder *PerformanceInputBuilder) teacherStudents(ctx context.Context, classRefs []performance.TeacherClass) ([]performance.TeacherStudent, error) {
	classIDs := classIDsFromClassRefs(classRefs)
	if len(classIDs) == 0 {
		return []performance.TeacherStudent{}, nil
	}
	var enrolled []enrollments.Enrollment
	if err := builder.db.WithContext(ctx).Where("class_id IN ? AND status = ?", classIDs, enrollments.StatusActive).Find(&enrolled).Error; err != nil {
		return nil, translate(err)
	}
	if len(enrolled) == 0 {
		return []performance.TeacherStudent{}, nil
	}
	studentIDs := make([]uuid.UUID, 0, len(enrolled))
	seen := make(map[uuid.UUID]struct{}, len(enrolled))
	for _, enrollment := range enrolled {
		if _, ok := seen[enrollment.StudentID]; ok {
			continue
		}
		seen[enrollment.StudentID] = struct{}{}
		studentIDs = append(studentIDs, enrollment.StudentID)
	}
	var scoreRows []attemptAverageRow
	if err := builder.db.WithContext(ctx).Model(&attempts.QuizAttempt{}).
		Select("student_id AS scope_id, AVG(CASE WHEN max_score > 0 THEN score * 100.0 / max_score ELSE 0 END) AS average").
		Where("student_id IN ? AND status = ?", studentIDs, attempts.StatusCompleted).
		Group("student_id").
		Find(&scoreRows).Error; err != nil {
		return nil, translate(err)
	}
	scores := make(map[uuid.UUID]float64, len(scoreRows))
	for _, item := range scoreRows {
		scores[item.ScopeID] = roundPercent(item.Average)
	}
	rates, err := builder.attendanceRatesForStudents(ctx, studentIDs)
	if err != nil {
		return nil, err
	}
	events, err := builder.eventCountsForStudents(ctx, studentIDs)
	if err != nil {
		return nil, err
	}
	previous, err := builder.previousStudentScores(ctx, studentIDs)
	if err != nil {
		return nil, err
	}
	result := make([]performance.TeacherStudent, 0, len(enrolled))
	for _, enrollment := range enrolled {
		entry := performance.TeacherStudent{
			StudentID:      enrollment.StudentID.String(),
			ClassID:        enrollment.ClassID.String(),
			Score:          scores[enrollment.StudentID],
			AttendanceRate: rates[enrollment.StudentID],
			ActivityEvents: events[enrollment.StudentID],
		}
		if value, ok := previous[enrollment.StudentID]; ok {
			entry.PreviousScore = &value
		}
		result = append(result, entry)
	}
	return result, nil
}

func (builder *PerformanceInputBuilder) previousStudentScores(ctx context.Context, studentIDs []uuid.UUID) (map[uuid.UUID]float64, error) {
	if len(studentIDs) == 0 {
		return nil, nil
	}
	var rows []studentSnapshotRow
	err := builder.db.WithContext(ctx).Model(&performance.Snapshot{}).
		Where("subject_type = ? AND subject_id IN ?", performance.SubjectStudent, studentIDs).
		Where("version = (SELECT MAX(latest.version) FROM snapshots AS latest WHERE latest.subject_type = snapshots.subject_type AND latest.subject_id = snapshots.subject_id AND latest.deleted_at IS NULL)").
		Find(&rows).Error
	if err != nil {
		return nil, translate(err)
	}
	result := make(map[uuid.UUID]float64, len(rows))
	for _, row := range rows {
		result[row.SubjectID] = roundPercent(row.OverallScore)
	}
	return result, nil
}

func (builder *PerformanceInputBuilder) attendanceRatesForStudents(ctx context.Context, studentIDs []uuid.UUID) (map[uuid.UUID]float64, error) {
	var rows []attendanceStatusRow
	err := builder.db.WithContext(ctx).Model(&attendance.Attendance{}).
		Select("student_id AS scope_id, status, COUNT(*) AS amount").
		Where("student_id IN ?", studentIDs).
		Group("student_id, status").
		Find(&rows).Error
	if err != nil {
		return nil, translate(err)
	}
	summaries := make(map[uuid.UUID]attendance.Summary, len(studentIDs))
	for _, item := range rows {
		summary := summaries[item.ScopeID]
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
		summaries[item.ScopeID] = summary
	}
	rates := make(map[uuid.UUID]float64, len(summaries))
	for studentID, summary := range summaries {
		rates[studentID] = summary.AttendedRate()
	}
	return rates, nil
}

func (builder *PerformanceInputBuilder) eventCountsForStudents(ctx context.Context, studentIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	type row struct {
		ScopeID uuid.UUID
		Amount  int64
	}
	var rows []row
	err := builder.db.WithContext(ctx).Model(&activities.DailyAggregate{}).
		Select("student_id AS scope_id, SUM(events) AS amount").
		Where("student_id IN ?", studentIDs).
		Group("student_id").
		Find(&rows).Error
	if err != nil {
		return nil, translate(err)
	}
	result := make(map[uuid.UUID]int, len(rows))
	for _, item := range rows {
		result[item.ScopeID] = int(item.Amount)
	}
	return result, nil
}

func (builder *PerformanceInputBuilder) teacherAttendance(ctx context.Context, classRefs []performance.TeacherClass) (performance.TeacherAttendance, error) {
	classIDs := classIDsFromClassRefs(classRefs)
	input := performance.TeacherAttendance{AssignedClasses: len(classIDs)}
	if len(classIDs) == 0 {
		return input, nil
	}
	var rows []attendanceStatusRow
	err := builder.db.WithContext(ctx).Model(&attendance.Attendance{}).
		Select("class_id AS scope_id, status, COUNT(*) AS amount").
		Where("class_id IN ?", classIDs).
		Group("class_id, status").
		Find(&rows).Error
	if err != nil {
		return performance.TeacherAttendance{}, err
	}
	totals := make(map[uuid.UUID]int, len(classIDs))
	for _, item := range rows {
		totals[item.ScopeID] += item.Amount
		input.Records += item.Amount
		switch item.Status {
		case attendance.StatusPresent:
			input.Present += item.Amount
		case attendance.StatusAbsent:
			input.Absent += item.Amount
		case attendance.StatusLate:
			input.Late += item.Amount
		case attendance.StatusExcused:
			input.Excused += item.Amount
		}
	}
	input.ClassesWithRecord = len(totals)
	input.Rate = percent(input.ClassesWithRecord, input.AssignedClasses)
	return input, nil
}

func (builder *PerformanceInputBuilder) teacherActivity(ctx context.Context, teacherID uuid.UUID, courseIDs []uuid.UUID, pairs []teacherScopePair) (performance.TeacherActivity, error) {
	input := performance.TeacherActivity{}
	classIDs := classIDsFromPairs(pairs)
	lessonsInScope, err := NewLessonRepository(builder.db).CountInScope(ctx, courseIDs)
	if err != nil {
		return performance.TeacherActivity{}, err
	}
	quizzesInScope, err := builder.quizCountInScope(ctx, pairs)
	if err != nil {
		return performance.TeacherActivity{}, err
	}
	questionsInScope, err := builder.questionCountInScope(ctx, pairs)
	if err != nil {
		return performance.TeacherActivity{}, err
	}
	input.ContentEvents = int(lessonsInScope) + quizzesInScope + int(questionsInScope)
	input.ContentUpdates = input.ContentEvents

	attendanceTaken, err := NewAttendanceRepository(builder.db).CountByTeacher(ctx, teacherID, classIDs)
	if err != nil {
		return performance.TeacherActivity{}, err
	}
	input.AttendanceTaken = int(attendanceTaken)

	notesCreated, err := NewNoteRepository(builder.db).CountByTeacher(ctx, teacherID)
	if err != nil {
		return performance.TeacherActivity{}, err
	}
	input.NotesCreated = int(notesCreated)

	activeDays, err := builder.activeContentDays(ctx, teacherID)
	if err != nil {
		return performance.TeacherActivity{}, err
	}
	input.ActiveDays = activeDays
	return input, nil
}

func (builder *PerformanceInputBuilder) activeContentDays(ctx context.Context, teacherID uuid.UUID) (int, error) {
	var days []string
	query := builder.db.WithContext(ctx).Raw(
		"SELECT DATE(created_at) AS day FROM lessons WHERE created_by_id = ? AND deleted_at IS NULL UNION SELECT DATE(created_at) AS day FROM quizzes WHERE created_by_id = ? AND deleted_at IS NULL",
		teacherID, teacherID).
		Scan(&days)
	if query.Error != nil {
		return 0, translate(query.Error)
	}
	return len(days), nil
}

type attendanceStatusRow struct {
	ScopeID  uuid.UUID `gorm:"column:scope_id"`
	CourseID uuid.UUID `gorm:"column:course_id"`
	Status   attendance.Status
	Amount   int
}

type attemptAverageRow struct {
	ScopeID uuid.UUID `gorm:"column:scope_id"`
	Average float64
}

type studentSnapshotRow struct {
	SubjectID    uuid.UUID
	OverallScore float64
}

type quizStatRow struct {
	ScopeID      uuid.UUID `gorm:"column:scope_id"`
	Attempts     int
	AverageScore float64
	PassRate     float64
}

func teacherScopeArgs(table string, pairs []teacherScopePair) (string, []any) {
	conditions := make([]string, 0, len(pairs))
	args := make([]any, 0, len(pairs)*2)
	for _, pair := range pairs {
		conditions = append(conditions, table+".class_id = ? AND "+table+".course_id = ?")
		args = append(args, pair.ClassID, pair.CourseID)
	}
	if len(conditions) == 0 {
		return "1 = 0", nil
	}
	return "(" + strings.Join(conditions, " OR ") + ")", args
}

func mapKeys(values map[uuid.UUID][]uuid.UUID) []uuid.UUID {
	keys := make([]uuid.UUID, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func (builder *PerformanceInputBuilder) studentCountsByClass(ctx context.Context, classIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	result := make(map[uuid.UUID]int, len(classIDs))
	if len(classIDs) == 0 {
		return result, nil
	}
	var rows []enrollments.Enrollment
	if err := builder.db.WithContext(ctx).
		Where("class_id IN ? AND status = ?", classIDs, enrollments.StatusActive).
		Find(&rows).Error; err != nil {
		return nil, translate(err)
	}
	for _, enrollment := range rows {
		result[enrollment.ClassID]++
	}
	return result, nil
}

func classIDsFromPairs(pairs []teacherScopePair) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(pairs))
	seen := make(map[uuid.UUID]struct{}, len(pairs))
	for _, pair := range pairs {
		if _, ok := seen[pair.ClassID]; ok {
			continue
		}
		seen[pair.ClassID] = struct{}{}
		ids = append(ids, pair.ClassID)
	}
	return ids
}

func classIDsFromClassRefs(classRefs []performance.TeacherClass) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(classRefs))
	for _, classRef := range classRefs {
		parsed, err := uuid.Parse(classRef.ClassID)
		if err != nil {
			continue
		}
		ids = append(ids, parsed)
	}
	return ids
}

func previousScore(detail *performance.SnapshotDetail) *float64 {
	if detail == nil {
		return nil
	}
	score := detail.OverallScore
	return &score
}

func sumCounts(counts map[string]int64) int {
	total := 0
	for _, amount := range counts {
		total += int(amount)
	}
	return total
}

func percent(done int, total int) float64 {
	if total <= 0 {
		return 0
	}
	return roundPercent(float64(done) / float64(total) * 100)
}

func roundPercent(value float64) float64 {
	return float64(int64(value*100+0.5)) / 100
}

func fullName(user users.User) string {
	return user.Name + " " + user.Surname
}
