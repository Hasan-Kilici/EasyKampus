package performance

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"

	"lms/backend/pkg/core"
)

const retainedSnapshots = 10

type StudentResponse struct {
	StudentID   uuid.UUID      `json:"student_id"`
	SnapshotID  uuid.UUID      `json:"snapshot_id"`
	Cached      bool           `json:"cached"`
	GeneratedAt time.Time      `json:"generated_at"`
	Result      *StudentResult `json:"result"`
}

type TeacherResponse struct {
	TeacherID   uuid.UUID      `json:"teacher_id"`
	SnapshotID  uuid.UUID      `json:"snapshot_id"`
	Cached      bool           `json:"cached"`
	GeneratedAt time.Time      `json:"generated_at"`
	Result      *TeacherResult `json:"result"`
}

type Service interface {
	StudentPerformance(ctx context.Context, studentID uuid.UUID) (*StudentResponse, error)
	TeacherPerformance(ctx context.Context, teacherID uuid.UUID) (*TeacherResponse, error)
	InvalidateStudent(ctx context.Context, studentID uuid.UUID) error
	InvalidateTeacher(ctx context.Context, teacherID uuid.UUID) error
	InvalidateClass(ctx context.Context, classID uuid.UUID) error
	InvalidateCourse(ctx context.Context, courseID uuid.UUID) error
}

type service struct {
	repository Repository
	builder    InputBuilder
	engine     Engine
	cache      core.Cache
	flight     singleflight.Group
}

func NewService(repository Repository, builder InputBuilder, engine Engine, caches ...core.Cache) Service {
	var cache core.Cache
	if len(caches) > 0 {
		cache = caches[0]
	}
	return &service{repository: repository, builder: builder, engine: engine, cache: cache}
}

func (service *service) StudentPerformance(ctx context.Context, studentID uuid.UUID) (*StudentResponse, error) {
	if studentID == uuid.Nil {
		return nil, core.ErrValidation
	}
	key := studentCacheKey(studentID)
	if response := service.cachedStudent(ctx, key, studentID); response != nil {
		return response, nil
	}

	value, err, _ := service.flight.Do(key, func() (any, error) {
		if response := service.cachedStudent(ctx, key, studentID); response != nil {
			return response, nil
		}
		return service.computeStudent(ctx, studentID)
	})
	if err != nil {
		return nil, err
	}
	response, ok := value.(*StudentResponse)
	if !ok {
		return nil, core.ErrUnavailable
	}
	return response, nil
}

func (service *service) TeacherPerformance(ctx context.Context, teacherID uuid.UUID) (*TeacherResponse, error) {
	if teacherID == uuid.Nil {
		return nil, core.ErrValidation
	}
	key := teacherCacheKey(teacherID)
	if response := service.cachedTeacher(ctx, key, teacherID); response != nil {
		return response, nil
	}

	value, err, _ := service.flight.Do(key, func() (any, error) {
		if response := service.cachedTeacher(ctx, key, teacherID); response != nil {
			return response, nil
		}
		return service.computeTeacher(ctx, teacherID)
	})
	if err != nil {
		return nil, err
	}
	response, ok := value.(*TeacherResponse)
	if !ok {
		return nil, core.ErrUnavailable
	}
	return response, nil
}

func (service *service) InvalidateStudent(ctx context.Context, studentID uuid.UUID) error {
	return service.deletePrefix(ctx, studentCacheKey(studentID))
}

func (service *service) InvalidateTeacher(ctx context.Context, teacherID uuid.UUID) error {
	return service.deletePrefix(ctx, teacherCacheKey(teacherID))
}

func (service *service) InvalidateClass(ctx context.Context, classID uuid.UUID) error {
	studentIDs, err := service.repository.StudentIDsForClass(ctx, classID)
	if err != nil {
		return err
	}
	for _, studentID := range studentIDs {
		if err := service.InvalidateStudent(ctx, studentID); err != nil {
			return err
		}
	}
	return nil
}

func (service *service) InvalidateCourse(ctx context.Context, courseID uuid.UUID) error {
	studentIDs, err := service.repository.StudentIDsForCourse(ctx, courseID)
	if err != nil {
		return err
	}
	for _, studentID := range studentIDs {
		if err := service.InvalidateStudent(ctx, studentID); err != nil {
			return err
		}
	}
	return nil
}

func (service *service) computeStudent(ctx context.Context, studentID uuid.UUID) (*StudentResponse, error) {
	input, err := service.builder.BuildStudentInput(ctx, studentID)
	if err != nil {
		return nil, err
	}
	previous, err := service.repository.LatestStudentScore(ctx, studentID)
	if err != nil {
		return nil, err
	}
	input.PreviousScore = previous

	result, err := service.engine.AnalyzeStudent(ctx, *input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, core.ErrUnavailable
	}
	annotateRelevance(input, result)

	snapshot, courses, topics, problems, recommendations := buildStudentSnapshot(studentID, result)
	if err := service.repository.SaveStudentSnapshot(ctx, snapshot, courses, topics, problems, recommendations); err != nil {
		return nil, err
	}
	if err := service.store(ctx, studentCacheKey(studentID), &StudentResponse{
		StudentID:   studentID,
		SnapshotID:  snapshot.ID,
		GeneratedAt: snapshot.GeneratedAt,
		Result:      result,
	}); err != nil {
		return nil, err
	}
	return &StudentResponse{StudentID: studentID, SnapshotID: snapshot.ID, GeneratedAt: snapshot.GeneratedAt, Result: result}, nil
}

func (service *service) computeTeacher(ctx context.Context, teacherID uuid.UUID) (*TeacherResponse, error) {
	input, err := service.builder.BuildTeacherInput(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	previous, err := service.repository.LatestTeacherScore(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	input.PreviousScore = previous

	result, err := service.engine.AnalyzeTeacher(ctx, *input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, core.ErrUnavailable
	}

	snapshot, courses, topics, problems, recommendations := buildTeacherSnapshot(teacherID, result)
	if err := service.repository.SaveTeacherSnapshot(ctx, snapshot, courses, topics, problems, recommendations); err != nil {
		return nil, err
	}
	if err := service.store(ctx, teacherCacheKey(teacherID), &TeacherResponse{
		TeacherID:   teacherID,
		SnapshotID:  snapshot.ID,
		GeneratedAt: snapshot.GeneratedAt,
		Result:      result,
	}); err != nil {
		return nil, err
	}
	return &TeacherResponse{TeacherID: teacherID, SnapshotID: snapshot.ID, GeneratedAt: snapshot.GeneratedAt, Result: result}, nil
}

func (service *service) cachedStudent(ctx context.Context, key string, studentID uuid.UUID) *StudentResponse {
	if service.cache == nil {
		return nil
	}
	value, ok := service.cache.Get(ctx, key)
	if !ok {
		return nil
	}
	response, ok := value.(*StudentResponse)
	if !ok {
		return nil
	}
	cached := *response
	cached.Cached = true
	return &cached
}

func (service *service) cachedTeacher(ctx context.Context, key string, teacherID uuid.UUID) *TeacherResponse {
	if service.cache == nil {
		return nil
	}
	value, ok := service.cache.Get(ctx, key)
	if !ok {
		return nil
	}
	response, ok := value.(*TeacherResponse)
	if !ok {
		return nil
	}
	cached := *response
	cached.Cached = true
	return &cached
}

func (service *service) store(ctx context.Context, key string, response any) error {
	if service.cache == nil {
		return nil
	}
	return service.cache.Set(ctx, key, response)
}

func (service *service) deletePrefix(ctx context.Context, prefix string) error {
	if service.cache == nil {
		return nil
	}
	return service.cache.DeletePrefix(ctx, prefix)
}

func buildStudentSnapshot(studentID uuid.UUID, result *StudentResult) (*Snapshot, []SnapshotCourse, []SnapshotTopic, []Problem, []Recommendation) {
	snapshot := &Snapshot{
		SubjectType:   SubjectStudent,
		SubjectID:     studentID,
		Version:       result.Version,
		Engine:        "python-performance-service",
		OverallScore:  result.OverallScore,
		OverallStatus: result.OverallStatus,
		GeneratedAt:   result.GeneratedAt,
		Metadata:      encodeMetrics(result.Metrics),
	}
	courses := make([]SnapshotCourse, 0, len(result.Courses))
	topics := make([]SnapshotTopic, 0, len(result.WeakAreas))
	for _, course := range result.Courses {
		courses = append(courses, SnapshotCourse{
			SnapshotID:     snapshot.ID,
			CourseID:       parseOrNil(course.CourseID),
			Score:          course.Score,
			Status:         course.Status,
			ClassCount:     course.ClassCount,
			StudentCount:   course.StudentCount,
			AttendanceRate: course.Attendance,
			ActivityScore:  course.Activity,
			CompletionRate: course.Completion,
			QuizCount:      course.QuizCount,
			CompletedQuiz:  course.CompletedQuiz,
		})
		for _, topic := range course.Topics {
			topics = append(topics, SnapshotTopic{
				SnapshotID: snapshot.ID,
				CourseID:   parseOrNil(course.CourseID),
				TopicID:    parseOrNil(topic.TopicID),
				Score:      topic.Score,
				Status:     topic.Status,
				Correct:    topic.Correct,
				Questions:  topic.Questions,
			})
		}
	}
	problems := make([]Problem, 0, len(result.Problems))
	for _, problem := range result.Problems {
		problems = append(problems, Problem{
			SnapshotID:  snapshot.ID,
			Type:        problem.Type,
			Severity:    problem.Severity,
			Subject:     problem.Subject,
			TopicID:     parseOptionalUUID(problem.TopicID),
			Message:     problem.Message,
			GeneratedAt: result.GeneratedAt,
		})
	}
	recommendations := make([]Recommendation, 0, len(result.Recommendations))
	for _, recommendation := range result.Recommendations {
		recommendations = append(recommendations, Recommendation{
			SnapshotID:  snapshot.ID,
			Type:        recommendation.Type,
			Severity:    recommendation.Severity,
			Subject:     recommendation.Subject,
			TopicID:     parseOptionalUUID(recommendation.TopicID),
			Title:       recommendation.Title,
			Message:     recommendation.Message,
			Action:      recommendation.Action,
			Priority:    recommendation.Priority,
			GeneratedAt: result.GeneratedAt,
		})
	}
	return snapshot, courses, topics, problems, recommendations
}

func buildTeacherSnapshot(teacherID uuid.UUID, result *TeacherResult) (*Snapshot, []SnapshotCourse, []SnapshotTopic, []Problem, []Recommendation) {
	snapshot := &Snapshot{
		SubjectType:   SubjectTeacher,
		SubjectID:     teacherID,
		Version:       result.Version,
		Engine:        "python-performance-service",
		OverallScore:  result.OverallScore,
		OverallStatus: result.OverallStatus,
		GeneratedAt:   result.GeneratedAt,
		Metadata:      encodeMetrics(result.Metrics),
	}
	courses := make([]SnapshotCourse, 0, len(result.Courses))
	for _, course := range result.Courses {
		courses = append(courses, SnapshotCourse{
			SnapshotID:   snapshot.ID,
			CourseID:     parseOrNil(course.CourseID),
			Score:        course.Score,
			Status:       course.Status,
			ClassCount:   course.ClassCount,
			StudentCount: course.StudentCount,
		})
	}
	problems := make([]Problem, 0, len(result.Problems))
	for _, problem := range result.Problems {
		problems = append(problems, Problem{
			SnapshotID:  snapshot.ID,
			Type:        problem.Type,
			Severity:    problem.Severity,
			Subject:     problem.Subject,
			Message:     problem.Message,
			GeneratedAt: result.GeneratedAt,
		})
	}
	recommendations := make([]Recommendation, 0, len(result.Recommendations))
	for _, recommendation := range result.Recommendations {
		recommendations = append(recommendations, Recommendation{
			SnapshotID:  snapshot.ID,
			Type:        recommendation.Type,
			Severity:    recommendation.Severity,
			Subject:     recommendation.Subject,
			Title:       recommendation.Title,
			Message:     recommendation.Message,
			Action:      recommendation.Action,
			Priority:    recommendation.Priority,
			GeneratedAt: result.GeneratedAt,
		})
	}
	return snapshot, courses, nil, problems, recommendations
}

func encodeMetrics(metrics map[string]float64) string {
	if len(metrics) == 0 {
		return ""
	}
	encoded, err := json.Marshal(metrics)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func studentCacheKey(studentID uuid.UUID) string { return "performance:student:" + studentID.String() }

func teacherCacheKey(teacherID uuid.UUID) string { return "performance:teacher:" + teacherID.String() }

func parseOrNil(value string) uuid.UUID {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil
	}
	return parsed
}

func parseOptionalUUID(value *string) *uuid.UUID {
	if value == nil {
		return nil
	}
	parsed, err := uuid.Parse(*value)
	if err != nil {
		return nil
	}
	return &parsed
}
