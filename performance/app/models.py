"""Go backend ile paylaşılan sözleşmenin Python karşılığı.

Backend'deki `pkg/domain/performance/contract.go` ile alan adları ve tip
adresleme birebir eşleşmelidir; sözleşme sürümü `ContractVersion = 1`.
"""

from __future__ import annotations

from datetime import datetime
from typing import Dict, List, Optional

from pydantic import BaseModel, Field, field_validator, model_validator

CONTRACT_VERSION = 1

STATUS_EXCELLENT = "excellent"
STATUS_GOOD = "good"
STATUS_NEEDS_ATTENTION = "needs_attention"
STATUS_WEAK = "weak"
STATUS_UNKNOWN = "unknown"

SEVERITY_INFO = "info"
SEVERITY_LOW = "low"
SEVERITY_MEDIUM = "medium"
SEVERITY_HIGH = "high"

STATUSES = (
    STATUS_EXCELLENT,
    STATUS_GOOD,
    STATUS_NEEDS_ATTENTION,
    STATUS_WEAK,
    STATUS_UNKNOWN,
)


class StudentRef(BaseModel):
    id: str = Field(min_length=1, max_length=128)
    name: str = ""
    class_ids: List[str] = Field(default_factory=list)


class ClassRef(BaseModel):
    id: str = Field(min_length=1, max_length=128)
    name: str = ""
    level: str = ""
    school_id: str = ""
    course_ids: List[str] = Field(default_factory=list)


class TopicRef(BaseModel):
    id: str = Field(min_length=1, max_length=128)
    name: str = ""
    code: str = ""
    parent_id: Optional[str] = None
    weight: float = 0.0
    # Konunun ne anlattığı; konu–bölüm ilgisini (TF-IDF) Go backend hesaplar,
    # alan yalnızca sözleşme birebir eşleşsin diye burada da tanımlıdır.
    description: str = ""


class CourseRef(BaseModel):
    id: str = Field(min_length=1, max_length=128)
    name: str = ""
    subject_area: str = ""
    department: str = ""
    department_keywords: str = ""
    class_ids: List[str] = Field(default_factory=list)
    topics: List[TopicRef] = Field(default_factory=list)


class AnswerInput(BaseModel):
    question_id: str = Field(min_length=1, max_length=128)
    topic_id: Optional[str] = None
    correct: bool = False
    points: float = Field(default=0.0, ge=0.0, allow_inf_nan=False)


class QuizResult(BaseModel):
    attempt_id: str = Field(min_length=1, max_length=128)
    quiz_id: str = Field(min_length=1, max_length=128)
    course_id: str = Field(min_length=1, max_length=128)
    class_id: str = Field(min_length=1, max_length=128)
    name: str = ""
    score: float = Field(default=0.0, ge=0.0, allow_inf_nan=False)
    max_score: float = Field(default=0.0, ge=0.0, allow_inf_nan=False)
    started_at: Optional[datetime] = None
    completed_at: Optional[datetime] = None
    duration_seconds: int = Field(default=0, ge=0)
    answers: List[AnswerInput] = Field(default_factory=list, max_length=1000)


class AttendanceCourse(BaseModel):
    course_id: str
    total: int = 0
    present: int = 0
    absent: int = 0
    late: int = 0
    excused: int = 0
    rate: float = 0.0


class AttendanceInput(BaseModel):
    total: int = Field(default=0, ge=0)
    present: int = Field(default=0, ge=0)
    absent: int = Field(default=0, ge=0)
    late: int = Field(default=0, ge=0)
    excused: int = Field(default=0, ge=0)
    rate: float = Field(default=0.0, ge=0.0, le=100.0, allow_inf_nan=False)
    per_course: List[AttendanceCourse] = Field(default_factory=list, max_length=500)

    @model_validator(mode="after")
    def validate_counts(self) -> "AttendanceInput":
        if self.total and self.present + self.absent + self.late + self.excused > self.total:
            raise ValueError("attendance counts cannot exceed total")
        return self


class DailyActivity(BaseModel):
    date: str
    events: int = 0
    study_minutes: int = 0
    lesson_views: int = 0
    lesson_completions: int = 0
    quiz_completions: int = 0


class ActivityInput(BaseModel):
    total_events: int = Field(default=0, ge=0)
    active_days: int = Field(default=0, ge=0)
    lesson_views: int = Field(default=0, ge=0)
    lesson_completions: int = Field(default=0, ge=0)
    quiz_started: int = Field(default=0, ge=0)
    quiz_completed: int = Field(default=0, ge=0)
    study_minutes: int = Field(default=0, ge=0)
    logins: int = Field(default=0, ge=0)
    daily: List[DailyActivity] = Field(default_factory=list, max_length=366)


class CourseCompletion(BaseModel):
    course_id: str
    total: int = 0
    completed: int = 0
    completion_rate: float = 0.0


class LessonInput(BaseModel):
    total: int = Field(default=0, ge=0)
    completed: int = Field(default=0, ge=0)
    completion_rate: float = Field(default=0.0, ge=0.0, le=100.0, allow_inf_nan=False)
    per_course: List[CourseCompletion] = Field(default_factory=list, max_length=500)

    @model_validator(mode="after")
    def validate_counts(self) -> "LessonInput":
        if self.completed > self.total:
            raise ValueError("completed lessons cannot exceed total")
        return self


class StudentInput(BaseModel):
    version: int = CONTRACT_VERSION
    student: StudentRef
    classes: List[ClassRef] = Field(default_factory=list, max_length=200)
    courses: List[CourseRef] = Field(default_factory=list, max_length=200)
    quiz_results: List[QuizResult] = Field(default_factory=list, max_length=1000)
    attendance: AttendanceInput = Field(default_factory=AttendanceInput)
    activity: ActivityInput = Field(default_factory=ActivityInput)
    lessons: LessonInput = Field(default_factory=LessonInput)
    previous_score: Optional[float] = None
    generated_at: Optional[datetime] = None

    @field_validator("version")
    @classmethod
    def validate_version(cls, value: int) -> int:
        if value != CONTRACT_VERSION:
            raise ValueError(f"unsupported contract version: {value}")
        return value


class TeacherRef(BaseModel):
    id: str = Field(min_length=1, max_length=128)
    name: str = ""
    school_ids: List[str] = Field(default_factory=list)


class TeacherClass(BaseModel):
    class_id: str
    name: str = ""
    school_id: str = ""
    student_count: int = 0
    course_ids: List[str] = Field(default_factory=list)
    lesson_count: int = 0
    quiz_count: int = 0


class TeacherCourse(BaseModel):
    course_id: str
    name: str = ""
    subject_area: str = ""
    class_ids: List[str] = Field(default_factory=list)
    student_count: int = 0
    average_student_score: float = 0.0
    previous_average_student_score: float = 0.0


class TeacherContent(BaseModel):
    lessons_created: int = 0
    lessons_approved: int = 0
    lessons_rejected: int = 0
    lessons_pending: int = 0
    quizzes_created: int = 0
    quizzes_approved: int = 0
    quizzes_rejected: int = 0
    quizzes_pending: int = 0
    questions_created: int = 0
    lessons_in_scope: int = 0
    quizzes_in_scope: int = 0


class TeacherQuiz(BaseModel):
    quiz_id: str
    name: str = ""
    course_id: str = ""
    class_id: str = ""
    attempts: int = 0
    student_count: int = 0
    average_score: float = 0.0
    pass_rate: float = 0.0
    duration_minutes: int = 0


class TeacherStudent(BaseModel):
    student_id: str
    class_id: str = ""
    score: float = 0.0
    previous_score: Optional[float] = None
    attendance_rate: float = 0.0
    activity_events: int = 0


class TeacherAttendance(BaseModel):
    records: int = 0
    present: int = 0
    absent: int = 0
    late: int = 0
    excused: int = 0
    rate: float = 0.0
    classes_with_record: int = 0
    assigned_classes: int = 0


class TeacherActivity(BaseModel):
    content_events: int = 0
    content_updates: int = 0
    active_days: int = 0
    attendance_taken: int = 0
    notes_created: int = 0


class TeacherInput(BaseModel):
    version: int = CONTRACT_VERSION
    teacher: TeacherRef
    classes: List[TeacherClass] = Field(default_factory=list, max_length=200)
    courses: List[TeacherCourse] = Field(default_factory=list, max_length=200)
    content: TeacherContent = Field(default_factory=TeacherContent)
    quizzes: List[TeacherQuiz] = Field(default_factory=list, max_length=1000)
    students: List[TeacherStudent] = Field(default_factory=list, max_length=5000)
    attendance: TeacherAttendance = Field(default_factory=TeacherAttendance)
    activity: TeacherActivity = Field(default_factory=TeacherActivity)
    previous_score: Optional[float] = None
    generated_at: Optional[datetime] = None

    @field_validator("version")
    @classmethod
    def validate_version(cls, value: int) -> int:
        if value != CONTRACT_VERSION:
            raise ValueError(f"unsupported contract version: {value}")
        return value


class TopicResult(BaseModel):
    topic_id: str
    score: float = 0.0
    status: str = STATUS_UNKNOWN
    correct: int = 0
    questions: int = 0


class CourseResult(BaseModel):
    course_id: str
    score: float = 0.0
    status: str = STATUS_UNKNOWN
    class_count: int = 0
    student_count: int = 0
    attendance: float = 0.0
    activity: float = 0.0
    completion: float = 0.0
    quiz_count: int = 0
    completed_quiz_count: int = 0
    topics: List[TopicResult] = Field(default_factory=list)


class MetricResult(BaseModel):
    score: float = 0.0
    status: str = STATUS_UNKNOWN
    values: Dict[str, float] = Field(default_factory=dict)


class ProblemOutput(BaseModel):
    type: str
    severity: str = SEVERITY_INFO
    subject: str = ""
    topic_id: Optional[str] = None
    message: str = ""


class RecommendationOutput(BaseModel):
    type: str
    severity: str = SEVERITY_INFO
    subject: str = ""
    topic_id: Optional[str] = None
    title: str = ""
    message: str = ""
    action: str = ""
    priority: int = 0


class StudentResult(BaseModel):
    version: int = CONTRACT_VERSION
    subject_id: str
    overall_score: float = 0.0
    overall_status: str = STATUS_UNKNOWN
    courses: List[CourseResult] = Field(default_factory=list)
    attendance: MetricResult = Field(default_factory=MetricResult)
    activity: MetricResult = Field(default_factory=MetricResult)
    completion: MetricResult = Field(default_factory=MetricResult)
    weak_areas: List[TopicResult] = Field(default_factory=list)
    problems: List[ProblemOutput] = Field(default_factory=list)
    recommendations: List[RecommendationOutput] = Field(default_factory=list)
    metrics: Dict[str, float] = Field(default_factory=dict)
    generated_at: Optional[datetime] = None


class TeacherCourseResult(BaseModel):
    course_id: str
    name: str = ""
    score: float = 0.0
    status: str = STATUS_UNKNOWN
    class_count: int = 0
    student_count: int = 0
    average_student_score: float = 0.0
    improvement: float = 0.0


class TeacherClassResult(BaseModel):
    class_id: str
    name: str = ""
    score: float = 0.0
    status: str = STATUS_UNKNOWN
    student_count: int = 0
    attendance: float = 0.0
    participation: float = 0.0


class StudentOutcomeResult(BaseModel):
    average_score: float = 0.0
    improvement: float = 0.0
    participating_rate: float = 0.0
    participating_count: int = 0
    status: str = STATUS_UNKNOWN


class TeacherResult(BaseModel):
    version: int = CONTRACT_VERSION
    subject_id: str
    overall_score: float = 0.0
    overall_status: str = STATUS_UNKNOWN
    courses: List[TeacherCourseResult] = Field(default_factory=list)
    classes: List[TeacherClassResult] = Field(default_factory=list)
    student_outcomes: StudentOutcomeResult = Field(default_factory=StudentOutcomeResult)
    activity: MetricResult = Field(default_factory=MetricResult)
    attendance: MetricResult = Field(default_factory=MetricResult)
    content: MetricResult = Field(default_factory=MetricResult)
    quiz_metrics: MetricResult = Field(default_factory=MetricResult)
    problems: List[ProblemOutput] = Field(default_factory=list)
    recommendations: List[RecommendationOutput] = Field(default_factory=list)
    metrics: Dict[str, float] = Field(default_factory=dict)
    generated_at: Optional[datetime] = None
