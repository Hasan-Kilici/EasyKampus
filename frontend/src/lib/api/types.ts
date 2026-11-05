// Swagger (LMS Backend API 1.0) modellerinden elle çıkarılmış tipler.
// Tam otomatik üretim için: npx swagger2openapi swagger.json -o openapi.json && npx openapi-typescript openapi.json -o schema.d.ts
// NOT: interface değil type kullanılır; böylece Rec (Record<string, unknown>) ile uyumlu kalır.

export type Rec = Record<string, unknown>;

export type Role = 'superadmin' | 'admin' | 'teacher' | 'student';
export type ApprovalState = 'draft' | 'pending' | 'approved' | 'rejected';
export type ReviewAction = 'submit' | 'approve' | 'reject';
type Base = { id: string; created_at: string; updated_at: string };

export type ErrorResponse = { data?: unknown; error?: { code: string; message: string } };

// ---------- Auth / Kullanıcı ----------
export type LoginInput = { email: string; password: string };
export type LoginResponse = { token: string; user: { id: string; name: string; surname: string; role: Role } };
export type RegisterInput = { name: string; surname: string; age: number; email: string; password: string };
export type User = Base & { name: string; surname: string; age: number; role: Role };
export type UserInput = { name: string; surname: string; age?: number; email: string; password: string; role: Role };

// ---------- Okul yapısı ----------
export type StaffRole = 'director' | 'registrar';
export type SchoolInput = { code: string; name: string };
export type School = Base & SchoolInput & { campuses?: Campus[] | null; classes?: Class[] | null; staff?: SchoolStaff[] | null };
export type SchoolStaff = Base & { school_id: string; user_id: string; role: StaffRole };
export type StaffInput = { user_id: string; role: StaffRole };
export type StartSchoolInput = { school: SchoolInput; campuses: { code: string; name: string }[] };
export type StartSchoolResult = { school: School; campuses: Campus[] };

export type CampusInput = { school_id: string; code: string; name: string };
export type Campus = Base & CampusInput & { classes?: Class[] | null };

export type ClassInput = { school_id: string; campus_id?: string; name: string; no?: string; level?: string };
export type Class = Base & { school_id: string; campus_id: string; name: string; no: string; level: string };

// ---------- Ders içeriği ----------
export type CourseInput = { school_id: string; code: string; name: string; description?: string; subject_area?: string; active?: boolean; department?: string; department_keywords?: string; homework_hours?: number; project_hours?: number };
export type Course = Base & { school_id: string; code: string; name: string; description: string; subject_area: string; active: boolean; department: string; department_keywords: string; homework_hours: number; project_hours: number; topics?: Topic[] | null };

// EFOR / AKTS: dersin iş yükü (saat) ve AKTS karşılığı.
export type TopicHours = { topic_id: string; name: string; hours: number };
export type Workload = {
	course_id: string; topic_hours: number; homework_hours: number; project_hours: number; total_hours: number;
	hours_per_ects: number; ects: number; ects_exact: number; topics: TopicHours[];
};

// Konu–bölüm ilgisi (TF-IDF + kosinüs), 0–100.
export type RelevanceBand = 'high' | 'medium' | 'low' | 'none';
export type TopicRelevance = { topic_id: string; score: number; band: RelevanceBand; matched_terms: string[] };
export type ContentAnalysis = {
	method: string; ready: boolean; department: string; department_terms: string[]; fit: number; topics: TopicRelevance[];
};

export type TopicInput = { course_id: string; parent_id?: string; code?: string; name: string; weight?: number; active?: boolean; description: string; study_hours?: number };
export type Topic = Base & { course_id: string; parent_id?: string; code: string; name: string; weight: number; active: boolean; description: string; study_hours: number };

export type LessonInput = { course_id: string; title: string; description?: string; markdown_content?: string; duration_minutes?: number; position?: number };
export type Lesson = Base & {
	course_id: string; created_by_id: string; title: string; description: string; markdown_content: string;
	duration_minutes: number; position: number; state: ApprovalState; note?: string; reviewed_at?: string; reviewed_by_id?: string;
};
export type LessonCompletion = Base & { student_id: string; course_id: string; lesson_id: string; completed_at: string };

export type VideoInput = { lesson_id?: string; title: string; url: string; duration?: number; position?: number };
export type Video = Base & {
	lesson_id: string; title: string; url: string; duration: number; position: number; created_by_id: string;
};
export type VideoCompletion = Base & {
	student_id: string; video_id: string; lesson_id: string; course_id: string; completed_at: string;
};
export type PresignInput = { filename: string; content_type?: string };
export type PresignResult = {
	key: string; configured: boolean; upload_url: string; method?: string;
	content_type?: string; expires_in?: number; public_url: string; message?: string;
};
export type PlaybackResult = { url: string; expires_in: number };

export type ReviewInput = { action: ReviewAction; note?: string; reviewer_id?: string };

// ---------- Sınav ----------
export type QuizInput = {
	class_id: string; course_id: string; name: string; description?: string; duration_minutes?: number;
	max_attempts?: number; points?: number; open_at?: string; close_at?: string;
};
export type Quiz = Base & {
	class_id: string; course_id: string; created_by_id: string; name: string; description: string; duration_minutes: number;
	max_attempts: number; points: number; open_at: string; close_at: string; state: ApprovalState;
	note?: string; reviewed_at?: string; reviewed_by_id?: string;
};
export type QuestionInput = { quiz_id: string; topic_id?: string; question: string; points?: number; position?: number };
export type Question = Base & {
	quiz_id: string; topic_id?: string; created_by_id: string; question: string; points: number; position: number;
	state: ApprovalState; note?: string; reviewed_at?: string; reviewed_by_id?: string;
};
export type OptionInput = { question_id: string; content: string; position?: number; answer?: boolean };
export type Option = Base & { question_id: string; content: string; position: number; answer: boolean };

export type AttemptStatus = 'in_progress' | 'completed' | 'abandoned';
export type AnswerInput = { question_id: string; selected_option_id: string };
export type AttemptAnswer = Base & {
	attempt_id: string; question_id: string; selected_option_id: string; topic_id?: string; correct: boolean; points: number; answered_at: string;
};
export type QuizAttempt = Base & {
	quiz_id: string; class_id: string; course_id: string; student_id: string; status: AttemptStatus; score: number; max_score: number;
	duration_seconds: number; started_at: string; completed_at?: string; answers?: AttemptAnswer[];
};
// Swagger GET /attempts/{id} cevabını bu şekilde gösteriyor (muhtemelen hatalı annotation)
export type QuizScope = {
	id: string; classID: string; courseID: string; state: string; durationMinutes: number; maxAttempts: number; openAt: string; closeAt: string;
};

// ---------- Sınav iş akışı ----------
export type ExamIssue = { path: string; message: string };
export type ExamOptionInput = { content: string; answer: boolean };
export type ExamQuestionInput = { topic_id?: string; question: string; points: number; options: ExamOptionInput[] };
export type ExamInput = {
	course_id: string; name: string; description?: string; open_at?: string | null; close_at?: string | null;
	duration_minutes: number; max_attempts: number; questions: ExamQuestionInput[];
};
export type ExamDetail = {
	quiz: Quiz; questions: (Question & { options: Option[] })[]; issues: ExamIssue[]; attempt_count: number; questions_editable: boolean;
};
export type TakeOption = { id: string; content: string; position: number };
export type TakeQuestion = { id: string; question: string; points: number; position: number; options: TakeOption[] };
export type TakeExam = {
	attempt: { id: string; status: AttemptStatus; started_at: string };
	quiz: { id: string; name: string; description: string; duration_minutes: number; max_attempts: number; close_at: string | null };
	deadline: string | null; server_time: string; questions: TakeQuestion[]; answers: Record<string, string>;
};
export type ResultOption = { id: string; content: string; selected: boolean; correct?: boolean };
export type ResultQuestion = {
	question_id: string; question: string; points: number; position: number; answered: boolean; selected_option_id: string | null;
	correct?: boolean; earned_points?: number; options: ResultOption[];
};
export type ExamResult = {
	attempt_id: string; quiz_id: string; quiz_name: string; status: AttemptStatus; score: number; max_score: number; percentage: number;
	started_at: string; completed_at: string | null; duration_seconds: number; answers_revealed: boolean; questions: ResultQuestion[];
};
export type ExamStudentResult = {
	student_id: string; attempts: number; best_score: number; last_score: number; max_score: number; best_percentage: number; last_completed_at: string | null;
};
export type ExamQuestionStat = { question_id: string; question: string; position: number; answered: number; correct: number; rate: number };
export type ExamSummary = {
	quiz_id: string; participants: number; average_percentage: number; highest_percentage: number; lowest_percentage: number;
	students: ExamStudentResult[]; questions: ExamQuestionStat[];
};

// ---------- Not / devamsızlık / kayıt ----------
export type GradeType = 'exam' | 'assignment' | 'project' | 'participation';
export type GradeInput = { class_id: string; course_id: string; student_id: string; quiz_id?: string; type: GradeType; score: number; term?: string; note?: string };
export type GradeUpdate = { score?: number; note?: string };
export type Grade = Base & {
	student_id: string; class_id: string; course_id: string; quiz_id?: string; graded_by: string; type: GradeType; score: number; term: string; note?: string;
};
export type GradeAudit = Base & {
	grade_id: string; actor_id: string; action: string; before_score: number; after_score: number; before_note?: string; after_note?: string;
};

export type AttendanceStatus = 'present' | 'absent' | 'late' | 'excused';
export type AttendanceInput = { class_id: string; course_id: string; student_id: string; date: string; status: AttendanceStatus; note?: string };
export type Attendance = Base & {
	student_id: string; class_id: string; course_id: string; teacher_id: string; date: string; status: AttendanceStatus; note?: string;
};
export type AttendanceDaySummary = {
	date: string; total: number; present: number; absent: number; late: number; excused: number;
};
export type AttendancePeriodSummary = {
	total: number; present: number; absent: number; late: number; excused: number; rate: number; days: AttendanceDaySummary[];
};

export type EnrollmentStatus = 'active' | 'withdrawn' | 'completed';
export type EnrollInput = { student_id: string; course_id?: string; term?: string; note?: string };
export type TransferInput = { student_id: string; course_id: string; new_class_id: string; term?: string; note?: string };
export type Enrollment = Base & {
	student_id: string; class_id: string; course_id: string; term: string; status: EnrollmentStatus; note?: string; enrolled_at: string; finished_at?: string;
};

export type AssignTeacherInput = { teacher_id: string; course_id: string; term?: string };
export type TeacherAssignment = Base & { teacher_id: string; class_id: string; course_id: string; term: string; active: boolean };
export type AssignCourseInput = { course_id: string; term?: string };
export type ClassCourse = Base & { class_id: string; course_id: string; term: string; active: boolean };

export type ScheduleInput = { course_id: string; teacher_id: string; term?: string; day_of_week: number; start_minute: number; end_minute: number };
export type ScheduleEntry = Base & ScheduleInput & { class_id: string; active: boolean };

export type NoteInput = { class_id: string; course_id: string; student_id: string; body: string; visible_to_student?: boolean };
export type Note = Base & { author_id: string; class_id: string; course_id: string; student_id: string; body: string; visible_to_student: boolean };

// ---------- Aktivite ----------
export type EventType =
	| 'lesson_view' | 'lesson_completion' | 'quiz_started' | 'quiz_completed'
	| 'question_answered' | 'assignment_completion' | 'login' | 'course_access';
export type Activity = Base & {
	student_id: string; type: EventType; course_id?: string; class_id?: string; lesson_id?: string; quiz_id?: string;
	duration_seconds: number; metadata?: string; occurred_at: string;
};
export type DailyAggregate = Base & {
	student_id: string; course_id?: string; date: string; events: number; logins: number; course_access: number; lesson_views: number;
	lesson_completions: number; quiz_started: number; quiz_completed: number; question_answered: number; study_minutes: number;
};

// ---------- Performans (Python analiz servisi) ----------
export type PerformanceStatus = 'excellent' | 'good' | 'needs_attention' | 'weak' | 'unknown';
export type Severity = 'info' | 'low' | 'medium' | 'high';
export type MetricResult = { score: number; status: PerformanceStatus; values?: Record<string, number> };
export type TopicResult = { topic_id: string; questions: number; correct: number; score: number; status: PerformanceStatus };
export type CourseResult = {
	course_id: string; score: number; status: PerformanceStatus; activity: number; attendance: number; completion: number;
	class_count: number; student_count: number; quiz_count: number; completed_quiz_count: number; topics?: TopicResult[];
};
export type ProblemOutput = { type: string; severity: Severity; subject: string; topic_id?: string; message: string };
export type RecommendationOutput = { type: string; severity: Severity; subject: string; topic_id?: string; title: string; message: string; action: string; priority: number };
type PerfCommon = {
	version: number; subject_id: string; generated_at: string; overall_score: number; overall_status: PerformanceStatus;
	metrics?: Record<string, number>; activity?: MetricResult; attendance?: MetricResult;
	problems?: ProblemOutput[]; recommendations?: RecommendationOutput[];
};
export type StudentResult = PerfCommon & { completion?: MetricResult; courses?: CourseResult[]; weak_areas?: TopicResult[] };
export type StudentPerformance = { snapshot_id: string; student_id: string; generated_at: string; cached: boolean; result: StudentResult };
export type TeacherClassResult = { class_id: string; name: string; score: number; status: PerformanceStatus; attendance: number; participation: number; student_count: number };
export type TeacherCourseResult = {
	course_id: string; name: string; score: number; status: PerformanceStatus; average_student_score: number; improvement: number; class_count: number; student_count: number;
};
export type StudentOutcomeResult = { average_score: number; improvement: number; participating_rate: number; participating_students: number; status: PerformanceStatus };
export type TeacherResult = PerfCommon & {
	content?: MetricResult; quiz_metrics?: MetricResult; student_outcomes?: StudentOutcomeResult;
	classes?: TeacherClassResult[]; courses?: TeacherCourseResult[];
};
export type TeacherPerformance = { snapshot_id: string; teacher_id: string; generated_at: string; cached: boolean; result: TeacherResult };
