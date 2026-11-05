package performance

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type StudentRef struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	ClassIDs []string `json:"class_ids"`
}

type ClassRef struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Level     string   `json:"level"`
	SchoolID  string   `json:"school_id"`
	CourseIDs []string `json:"course_ids"`
}

type CourseRef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SubjectArea string `json:"subject_area"`
	// Department ve DepartmentKeywords, konu–bölüm ilgisi (TF-IDF) için kullanılır.
	Department         string     `json:"department"`
	DepartmentKeywords string     `json:"department_keywords"`
	ClassIDs           []string   `json:"class_ids"`
	Topics             []TopicRef `json:"topics"`
}

type TopicRef struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Code        string  `json:"code"`
	ParentID    *string `json:"parent_id"`
	Weight      float64 `json:"weight"`
	Description string  `json:"description"`
}

type QuizResult struct {
	AttemptID       string        `json:"attempt_id"`
	QuizID          string        `json:"quiz_id"`
	CourseID        string        `json:"course_id"`
	ClassID         string        `json:"class_id"`
	Name            string        `json:"name"`
	Score           float64       `json:"score"`
	MaxScore        float64       `json:"max_score"`
	StartedAt       time.Time     `json:"started_at"`
	CompletedAt     time.Time     `json:"completed_at"`
	DurationSeconds int           `json:"duration_seconds"`
	Answers         []AnswerInput `json:"answers"`
}

type AnswerInput struct {
	QuestionID string  `json:"question_id"`
	TopicID    *string `json:"topic_id"`
	Correct    bool    `json:"correct"`
	Points     float64 `json:"points"`
}

type AttendanceCourse struct {
	CourseID string  `json:"course_id"`
	Total    int     `json:"total"`
	Present  int     `json:"present"`
	Absent   int     `json:"absent"`
	Late     int     `json:"late"`
	Excused  int     `json:"excused"`
	Rate     float64 `json:"rate"`
}

type AttendanceInput struct {
	Total     int                `json:"total"`
	Present   int                `json:"present"`
	Absent    int                `json:"absent"`
	Late      int                `json:"late"`
	Excused   int                `json:"excused"`
	Rate      float64            `json:"rate"`
	PerCourse []AttendanceCourse `json:"per_course"`
}

type DailyActivity struct {
	Date             string `json:"date"`
	Events           int    `json:"events"`
	StudyMinutes     int    `json:"study_minutes"`
	LessonViews      int    `json:"lesson_views"`
	LessonCompletion int    `json:"lesson_completions"`
	QuizCompletions  int    `json:"quiz_completions"`
}

type ActivityInput struct {
	TotalEvents       int             `json:"total_events"`
	ActiveDays        int             `json:"active_days"`
	LessonViews       int             `json:"lesson_views"`
	LessonCompletions int             `json:"lesson_completions"`
	QuizStarted       int             `json:"quiz_started"`
	QuizCompleted     int             `json:"quiz_completed"`
	StudyMinutes      int             `json:"study_minutes"`
	Logins            int             `json:"logins"`
	Daily             []DailyActivity `json:"daily"`
}

type CourseCompletion struct {
	CourseID string  `json:"course_id"`
	Total    int     `json:"total"`
	Complete int     `json:"completed"`
	Rate     float64 `json:"completion_rate"`
}

type LessonInput struct {
	Total     int                `json:"total"`
	Completed int                `json:"completed"`
	Rate      float64            `json:"completion_rate"`
	PerCourse []CourseCompletion `json:"per_course"`
}

type StudentInput struct {
	Version       int             `json:"version"`
	Student       StudentRef      `json:"student"`
	Classes       []ClassRef      `json:"classes"`
	Courses       []CourseRef     `json:"courses"`
	QuizResults   []QuizResult    `json:"quiz_results"`
	Attendance    AttendanceInput `json:"attendance"`
	Activity      ActivityInput   `json:"activity"`
	Lessons       LessonInput     `json:"lessons"`
	PreviousScore *float64        `json:"previous_score"`
	GeneratedAt   time.Time       `json:"generated_at"`
}

type TeacherRef struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	SchoolIDs []string `json:"school_ids"`
}

type TeacherClass struct {
	ClassID      string   `json:"class_id"`
	Name         string   `json:"name"`
	SchoolID     string   `json:"school_id"`
	StudentCount int      `json:"student_count"`
	CourseIDs    []string `json:"course_ids"`
	LessonCount  int      `json:"lesson_count"`
	QuizCount    int      `json:"quiz_count"`
}

type TeacherCourse struct {
	CourseID            string   `json:"course_id"`
	Name                string   `json:"name"`
	SubjectArea         string   `json:"subject_area"`
	ClassIDs            []string `json:"class_ids"`
	StudentCount        int      `json:"student_count"`
	AverageStudentScore float64  `json:"average_student_score"`
	PreviousAverage     float64  `json:"previous_average_student_score"`
}

type TeacherContent struct {
	LessonsCreated   int `json:"lessons_created"`
	LessonsApproved  int `json:"lessons_approved"`
	LessonsRejected  int `json:"lessons_rejected"`
	LessonsPending   int `json:"lessons_pending"`
	QuizzesCreated   int `json:"quizzes_created"`
	QuizzesApproved  int `json:"quizzes_approved"`
	QuizzesRejected  int `json:"quizzes_rejected"`
	QuizzesPending   int `json:"quizzes_pending"`
	QuestionsCreated int `json:"questions_created"`
	LessonsInScope   int `json:"lessons_in_scope"`
	QuizzesInScope   int `json:"quizzes_in_scope"`
}

type TeacherQuiz struct {
	QuizID          string  `json:"quiz_id"`
	Name            string  `json:"name"`
	CourseID        string  `json:"course_id"`
	ClassID         string  `json:"class_id"`
	Attempts        int     `json:"attempts"`
	StudentCount    int     `json:"student_count"`
	AverageScore    float64 `json:"average_score"`
	PassRate        float64 `json:"pass_rate"`
	DurationMinutes int     `json:"duration_minutes"`
}

type TeacherStudent struct {
	StudentID      string   `json:"student_id"`
	ClassID        string   `json:"class_id"`
	Score          float64  `json:"score"`
	PreviousScore  *float64 `json:"previous_score"`
	AttendanceRate float64  `json:"attendance_rate"`
	ActivityEvents int      `json:"activity_events"`
}

type TeacherAttendance struct {
	Records           int     `json:"records"`
	Present           int     `json:"present"`
	Absent            int     `json:"absent"`
	Late              int     `json:"late"`
	Excused           int     `json:"excused"`
	Rate              float64 `json:"rate"`
	ClassesWithRecord int     `json:"classes_with_records"`
	AssignedClasses   int     `json:"assigned_classes"`
}

type TeacherActivity struct {
	ContentEvents   int `json:"content_events"`
	ContentUpdates  int `json:"content_updates"`
	ActiveDays      int `json:"active_days"`
	AttendanceTaken int `json:"attendance_taken"`
	NotesCreated    int `json:"notes_created"`
}

type TeacherInput struct {
	Version       int               `json:"version"`
	Teacher       TeacherRef        `json:"teacher"`
	Classes       []TeacherClass    `json:"classes"`
	Courses       []TeacherCourse   `json:"courses"`
	Content       TeacherContent    `json:"content"`
	Quizzes       []TeacherQuiz     `json:"quizzes"`
	Students      []TeacherStudent  `json:"students"`
	Attendance    TeacherAttendance `json:"attendance"`
	Activity      TeacherActivity   `json:"activity"`
	PreviousScore *float64          `json:"previous_score"`
	GeneratedAt   time.Time         `json:"generated_at"`
}

type TopicResult struct {
	TopicID   string  `json:"topic_id"`
	Score     float64 `json:"score"`
	Status    Status  `json:"status"`
	Correct   int     `json:"correct"`
	Questions int     `json:"questions"`
	// Relevance (0–100): konunun dersin bölümüyle TF-IDF/kosinüs ilgisi. Python
	// servisi üretmez; backend sonuçtan sonra ekler. Bölüm bilgisi yoksa boştur.
	Relevance *float64 `json:"relevance,omitempty"`
}

type CourseResult struct {
	CourseID      string        `json:"course_id"`
	Score         float64       `json:"score"`
	Status        Status        `json:"status"`
	ClassCount    int           `json:"class_count"`
	StudentCount  int           `json:"student_count"`
	Attendance    float64       `json:"attendance"`
	Activity      float64       `json:"activity"`
	Completion    float64       `json:"completion"`
	QuizCount     int           `json:"quiz_count"`
	CompletedQuiz int           `json:"completed_quiz_count"`
	Topics        []TopicResult `json:"topics"`
	// DepartmentFit (0–100): dersin konularının bölümle ağırlıklı ortalama ilgisi.
	DepartmentFit *float64 `json:"department_fit,omitempty"`
}

type MetricResult struct {
	Score  float64            `json:"score"`
	Status Status             `json:"status"`
	Values map[string]float64 `json:"values"`
}

type ProblemOutput struct {
	Type     string   `json:"type"`
	Severity Severity `json:"severity"`
	Subject  string   `json:"subject"`
	TopicID  *string  `json:"topic_id"`
	Message  string   `json:"message"`
}

type RecommendationOutput struct {
	Type     string   `json:"type"`
	Severity Severity `json:"severity"`
	Subject  string   `json:"subject"`
	TopicID  *string  `json:"topic_id"`
	Title    string   `json:"title"`
	Message  string   `json:"message"`
	Action   string   `json:"action"`
	Priority int      `json:"priority"`
}

type StudentResult struct {
	Version         int                    `json:"version"`
	SubjectID       string                 `json:"subject_id"`
	OverallScore    float64                `json:"overall_score"`
	OverallStatus   Status                 `json:"overall_status"`
	Courses         []CourseResult         `json:"courses"`
	Attendance      MetricResult           `json:"attendance"`
	Activity        MetricResult           `json:"activity"`
	Completion      MetricResult           `json:"completion"`
	WeakAreas       []TopicResult          `json:"weak_areas"`
	Problems        []ProblemOutput        `json:"problems"`
	Recommendations []RecommendationOutput `json:"recommendations"`
	Metrics         map[string]float64     `json:"metrics"`
	GeneratedAt     time.Time              `json:"generated_at"`
}

type TeacherCourseResult struct {
	CourseID       string  `json:"course_id"`
	Name           string  `json:"name"`
	Score          float64 `json:"score"`
	Status         Status  `json:"status"`
	ClassCount     int     `json:"class_count"`
	StudentCount   int     `json:"student_count"`
	AverageStudent float64 `json:"average_student_score"`
	Improvement    float64 `json:"improvement"`
}

type TeacherClassResult struct {
	ClassID       string  `json:"class_id"`
	Name          string  `json:"name"`
	Score         float64 `json:"score"`
	Status        Status  `json:"status"`
	StudentCount  int     `json:"student_count"`
	Attendance    float64 `json:"attendance"`
	Participation float64 `json:"participation"`
}

type StudentOutcomeResult struct {
	AverageScore       float64 `json:"average_score"`
	Improvement        float64 `json:"improvement"`
	ParticipatingRate  float64 `json:"participating_rate"`
	ParticipatingCount int     `json:"participating_students"`
	Status             Status  `json:"status"`
}

type TeacherResult struct {
	Version         int                    `json:"version"`
	SubjectID       string                 `json:"subject_id"`
	OverallScore    float64                `json:"overall_score"`
	OverallStatus   Status                 `json:"overall_status"`
	Courses         []TeacherCourseResult  `json:"courses"`
	Classes         []TeacherClassResult   `json:"classes"`
	StudentOutcomes StudentOutcomeResult   `json:"student_outcomes"`
	Activity        MetricResult           `json:"activity"`
	Attendance      MetricResult           `json:"attendance"`
	Content         MetricResult           `json:"content"`
	QuizMetrics     MetricResult           `json:"quiz_metrics"`
	Problems        []ProblemOutput        `json:"problems"`
	Recommendations []RecommendationOutput `json:"recommendations"`
	Metrics         map[string]float64     `json:"metrics"`
	GeneratedAt     time.Time              `json:"generated_at"`
}

type Engine interface {
	AnalyzeStudent(ctx context.Context, input StudentInput) (*StudentResult, error)
	AnalyzeTeacher(ctx context.Context, input TeacherInput) (*TeacherResult, error)
	Ping(ctx context.Context) error
}

type InputBuilder interface {
	BuildStudentInput(ctx context.Context, studentID uuid.UUID) (*StudentInput, error)
	BuildTeacherInput(ctx context.Context, teacherID uuid.UUID) (*TeacherInput, error)
}

type SnapshotDetail struct {
	OverallScore   float64
	CourseAverages map[string]float64
}

type Repository interface {
	SaveStudentSnapshot(ctx context.Context, snapshot *Snapshot, courses []SnapshotCourse, topics []SnapshotTopic, problems []Problem, recommendations []Recommendation) error
	SaveTeacherSnapshot(ctx context.Context, snapshot *Snapshot, courses []SnapshotCourse, topics []SnapshotTopic, problems []Problem, recommendations []Recommendation) error
	PreviousSnapshot(ctx context.Context, subjectType SubjectType, subjectID uuid.UUID) (*SnapshotDetail, error)
	LatestStudentScore(ctx context.Context, studentID uuid.UUID) (*float64, error)
	LatestTeacherScore(ctx context.Context, teacherID uuid.UUID) (*float64, error)
	StudentIDsForClass(ctx context.Context, classID uuid.UUID) ([]uuid.UUID, error)
	StudentIDsForCourse(ctx context.Context, courseID uuid.UUID) ([]uuid.UUID, error)
}
