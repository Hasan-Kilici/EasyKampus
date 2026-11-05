"""Öğrenci ve öğretmen performans hesaplama motoru.

Tüm skorlar 0-100 aralığında yuvarlanır; veri yoksa `unknown` durumu üretilir.
Ağırlıklar tek yerde tanımlıdır ki backend ile kural farkları izlenebilir olsun.
"""

from __future__ import annotations

from datetime import datetime, timezone
from typing import Dict, Iterable, List, Optional, Tuple

from .models import (
    SEVERITY_HIGH,
    SEVERITY_INFO,
    SEVERITY_LOW,
    SEVERITY_MEDIUM,
    STATUS_EXCELLENT,
    STATUS_GOOD,
    STATUS_NEEDS_ATTENTION,
    STATUS_UNKNOWN,
    STATUS_WEAK,
    CourseResult,
    MetricResult,
    ProblemOutput,
    RecommendationOutput,
    StudentInput,
    StudentOutcomeResult,
    StudentResult,
    TeacherActivity,
    TeacherAttendance,
    TeacherClassResult,
    TeacherContent,
    TeacherCourseResult,
    TeacherInput,
    TeacherQuiz,
    TeacherResult,
    TopicResult,
)

QUIZ_WEIGHT = 50.0
ATTENDANCE_WEIGHT = 20.0
ACTIVITY_WEIGHT = 15.0
COMPLETION_WEIGHT = 15.0

PASS_SCORE = 50.0
WEAK_TOPIC_SCORE = 60.0
TARGET_ATTENDANCE = 85.0
TARGET_COMPLETION = 80.0
MAX_WEAK_AREAS = 5
MAX_PROBLEMS = 8
MAX_RECOMMENDATIONS = 6


def clamp(value: float) -> float:
    if value != value:  # NaN koruması
        return 0.0
    return round(min(100.0, max(0.0, float(value))), 2)


def percent(part: float, total: float) -> float:
    if total <= 0:
        return 0.0
    return clamp(part / total * 100.0)


def ratio(numerator: float, denominator: float) -> float:
    if denominator <= 0:
        return 0.0
    return round(float(numerator) / float(denominator), 4)


def status_for(score: float, has_data: bool = True) -> str:
    if not has_data:
        return STATUS_UNKNOWN
    if score >= 85.0:
        return STATUS_EXCELLENT
    if score >= 70.0:
        return STATUS_GOOD
    if score >= 50.0:
        return STATUS_NEEDS_ATTENTION
    return STATUS_WEAK


def metric(score: float, has_data: bool, **values: float) -> MetricResult:
    return MetricResult(
        score=clamp(score),
        status=status_for(clamp(score), has_data),
        values={key: round(float(value), 2) for key, value in values.items()},
    )


def now() -> datetime:
    return datetime.now(timezone.utc)


def weighted(parts: Iterable[Tuple[float, float]]) -> float:
    """(değer, ağırlık) çiftlerinden ağırlıklı ortalama; veri yoksa 0."""
    usable = [(value, weight) for value, weight in parts if weight > 0]
    total_weight = sum(weight for _, weight in usable)
    if total_weight <= 0:
        return 0.0
    return clamp(sum(value * weight for value, weight in usable) / total_weight)


def quiz_percent(score: float, max_score: float) -> float:
    if max_score <= 0:
        return 0.0
    return clamp(score / max_score * 100.0)


def linear_trend(quizzes: Iterable) -> Tuple[float, bool]:
    """Return the least-squares score slope over completed quiz attempts."""
    ordered = sorted(
        (quiz for quiz in quizzes if quiz.max_score > 0),
        key=lambda quiz: quiz.completed_at or quiz.started_at or datetime.min.replace(tzinfo=timezone.utc),
    )
    if len(ordered) < 2:
        return 0.0, False
    values = [quiz_percent(quiz.score, quiz.max_score) for quiz in ordered]
    x_mean = (len(values) - 1) / 2.0
    y_mean = sum(values) / len(values)
    denominator = sum((index - x_mean) ** 2 for index in range(len(values)))
    if denominator <= 0:
        return 0.0, False
    slope = sum((index - x_mean) * (value - y_mean) for index, value in enumerate(values)) / denominator
    return round(slope, 2), True


def topic_scores(quizzes: Iterable) -> Dict[str, TopicResult]:
    buckets: Dict[str, Dict[str, float]] = {}
    for quiz in quizzes:
        for answer in quiz.answers:
            if not answer.topic_id:
                continue
            bucket = buckets.setdefault(answer.topic_id, {"correct": 0.0, "questions": 0.0})
            bucket["questions"] += 1
            if answer.correct:
                bucket["correct"] += 1
    results: Dict[str, TopicResult] = {}
    for topic_id, bucket in buckets.items():
        questions = int(bucket["questions"])
        correct = int(bucket["correct"])
        score = percent(correct, questions)
        results[topic_id] = TopicResult(
            topic_id=topic_id,
            score=score,
            status=status_for(score, questions > 0),
            correct=correct,
            questions=questions,
        )
    return results


def activity_score(activity) -> float:
    """Etkinlik sinyali: ders görüntüleme, tamamlama, quiz ve çalışma süresi."""
    if activity.total_events <= 0:
        return 0.0
    study = min(100.0, activity.study_minutes / 120.0 * 100.0)
    return weighted(
        [
            (activity.lesson_completions, 30.0),
            (activity.quiz_completed, 30.0),
            (min(100.0, activity.lesson_views * 5.0), 20.0),
            (study, 20.0),
        ]
    )


def content_score(content: TeacherContent) -> Tuple[float, bool]:
    created = content.lessons_created + content.quizzes_created
    if created <= 0:
        return 0.0, False
    approved = content.lessons_approved + content.quizzes_approved
    rejected = content.lessons_rejected + content.quizzes_rejected
    return percent(approved, created), True


def teacher_activity_score(activity: TeacherActivity, classes: int) -> Tuple[float, bool]:
    if classes <= 0 and activity.content_events <= 0 and activity.attendance_taken <= 0:
        return 0.0, False
    return (
        weighted(
            [
                (min(100.0, activity.content_events * 10.0), 30.0),
                (min(100.0, activity.attendance_taken / max(classes, 1) * 20.0), 30.0),
                (min(100.0, activity.notes_created * 10.0), 20.0),
                (min(100.0, activity.active_days / 5.0 * 100.0), 20.0),
            ]
        ),
        True,
    )


def analyze_student(payload: StudentInput) -> StudentResult:
    result = StudentResult(subject_id=payload.student.id, generated_at=now())

    attendance_by_course = {item.course_id: item for item in payload.attendance.per_course}
    completion_by_course = {item.course_id: item for item in payload.lessons.per_course}
    quizzes_by_course: Dict[str, List] = {}
    for quiz in payload.quiz_results:
        quizzes_by_course.setdefault(quiz.course_id, []).append(quiz)

    class_ids = {item.id for item in payload.classes}
    course_ids = [course.id for course in payload.courses] or sorted(quizzes_by_course)

    for course_id in course_ids:
        course = next((item for item in payload.courses if item.id == course_id), None)
        course_quizzes = quizzes_by_course.get(course_id, [])
        topics = topic_scores(course_quizzes)

        quiz_score = 0.0
        has_quiz = bool(course_quizzes)
        if has_quiz:
            scores = [quiz_percent(quiz.score, quiz.max_score) for quiz in course_quizzes if quiz.max_score > 0]
            quiz_score = clamp(sum(scores) / len(scores)) if scores else 0.0

        attendance_row = attendance_by_course.get(course_id)
        has_attendance = attendance_row is not None and attendance_row.total > 0
        course_attendance = clamp(attendance_row.rate) if attendance_row else clamp(payload.attendance.rate)

        completion_row = completion_by_course.get(course_id)
        has_completion = completion_row is not None and completion_row.total > 0
        completion = clamp(completion_row.completion_rate) if completion_row else clamp(payload.lessons.completion_rate)

        has_activity = payload.activity.total_events > 0
        activity = activity_score(payload.activity)

        score = weighted(
            [
                (quiz_score, QUIZ_WEIGHT if has_quiz else 0.0),
                (course_attendance, ATTENDANCE_WEIGHT if has_attendance else 0.0),
                (activity, ACTIVITY_WEIGHT if has_activity else 0.0),
                (completion, COMPLETION_WEIGHT if has_completion else 0.0),
            ]
        )

        result.courses.append(
            CourseResult(
                course_id=course_id,
                score=score,
                status=status_for(score, has_quiz or has_attendance or has_completion or has_activity),
                class_count=len(course.class_ids) if course else len(class_ids),
                student_count=1,
                attendance=course_attendance,
                activity=clamp(activity),
                completion=completion,
                quiz_count=len({quiz.quiz_id for quiz in course_quizzes}),
                completed_quiz_count=len(course_quizzes),
                topics=sorted(topics.values(), key=lambda item: item.score),
            )
        )

    scored = [course for course in result.courses if course.status != STATUS_UNKNOWN]
    overall = clamp(sum(course.score for course in scored) / len(scored)) if scored else 0.0
    result.overall_score = overall
    result.overall_status = status_for(overall, bool(scored))

    has_attendance = payload.attendance.total > 0
    result.attendance = metric(
        payload.attendance.rate,
        has_attendance,
        present=payload.attendance.present,
        absent=payload.attendance.absent,
        late=payload.attendance.late,
        excused=payload.attendance.excused,
        total=payload.attendance.total,
    )

    has_activity = payload.activity.total_events > 0
    result.activity = metric(
        activity_score(payload.activity),
        has_activity,
        active_days=payload.activity.active_days,
        lesson_views=payload.activity.lesson_views,
        lesson_completions=payload.activity.lesson_completions,
        quiz_completed=payload.activity.quiz_completed,
        study_minutes=payload.activity.study_minutes,
    )

    has_completion = payload.lessons.total > 0
    result.completion = metric(
        payload.lessons.completion_rate,
        has_completion,
        completed=payload.lessons.completed,
        total=payload.lessons.total,
    )

    topic_totals: Dict[str, TopicResult] = {}
    for course in result.courses:
        for topic in course.topics:
            current = topic_totals.get(topic.topic_id)
            if current is None:
                topic_totals[topic.topic_id] = topic.model_copy(deep=True)
                continue
            current.correct += topic.correct
            current.questions += topic.questions
        topic_totals  # noqa: B018 - döngü sonrası kullanılıyor

    for topic in topic_totals.values():
        topic.score = percent(topic.correct, topic.questions)
        topic.status = status_for(topic.score, topic.questions > 0)

    result.weak_areas = sorted(
        (topic for topic in topic_totals.values() if topic.score < WEAK_TOPIC_SCORE),
        key=lambda item: item.score,
    )[:MAX_WEAK_AREAS]

    _student_problems(result, payload, overall)
    _student_recommendations(result, payload)
    result.metrics = {
        "overall_score": result.overall_score,
        "course_count": float(len(scored)),
        "quiz_average": clamp(
            sum(quiz_percent(quiz.score, quiz.max_score) for quiz in payload.quiz_results)
            / max(len(payload.quiz_results), 1)
        ) if payload.quiz_results else 0.0,
        "attendance_rate": clamp(payload.attendance.rate),
        "activity_score": result.activity.score,
        "completion_rate": clamp(payload.lessons.completion_rate),
        "weak_area_count": float(len(result.weak_areas)),
        "previous_score": clamp(payload.previous_score) if payload.previous_score is not None else 0.0,
        "improvement": round(overall - payload.previous_score, 2) if payload.previous_score is not None else 0.0,
    }
    trend, has_trend = linear_trend(payload.quiz_results)
    result.metrics["quiz_trend_slope"] = trend
    result.metrics["quiz_trend_has_data"] = 1.0 if has_trend else 0.0
    return result


def _student_problems(result: StudentResult, payload: StudentInput, overall: float) -> None:
    problems: List[ProblemOutput] = []

    if payload.attendance.total > 0 and payload.attendance.rate < TARGET_ATTENDANCE:
        severity = SEVERITY_HIGH if payload.attendance.rate < 60 else SEVERITY_MEDIUM
        problems.append(
            ProblemOutput(
                type="low_attendance",
                severity=severity,
                subject="attendance",
                message=(
                    f"Devamsızlık oranı %{payload.attendance.rate:.0f}; hedef %{TARGET_ATTENDANCE:.0f}."
                ),
            )
        )

    if payload.lessons.total > 0 and payload.lessons.completion_rate < TARGET_COMPLETION:
        problems.append(
            ProblemOutput(
                type="incomplete_lessons",
                severity=SEVERITY_MEDIUM,
                subject="completion",
                message=(
                    f"Ders tamamlama oranı %{payload.lessons.completion_rate:.0f};"
                    f" {payload.lessons.total - payload.lessons.completed} ders bekliyor."
                ),
            )
        )

    for topic in result.weak_areas[:3]:
        problems.append(
            ProblemOutput(
                type="weak_topic",
                severity=SEVERITY_MEDIUM if topic.score >= 40 else SEVERITY_HIGH,
                subject="topic",
                topic_id=topic.topic_id,
                message=(
                    f"Bu konuda {topic.questions} sorudan {topic.correct} doğru (%{topic.score:.0f})."
                ),
            )
        )

    if payload.activity.total_events == 0:
        problems.append(
            ProblemOutput(
                type="no_activity",
                severity=SEVERITY_LOW,
                subject="activity",
                message="Son dönemde kayıtlı etkinlik yok.",
            )
        )

    if payload.previous_score is not None and overall - payload.previous_score <= -5.0:
        problems.append(
            ProblemOutput(
                type="declining_trend",
                severity=SEVERITY_MEDIUM,
                subject="trend",
                message=(
                    f"Önceki puan %{payload.previous_score:.0f} iken güncel puan %{overall:.0f}."
                ),
            )
        )

    trend, has_trend = linear_trend(payload.quiz_results)
    if has_trend and trend <= -3.0:
        problems.append(
            ProblemOutput(
                type="declining_quiz_trend",
                severity=SEVERITY_MEDIUM if trend > -8.0 else SEVERITY_HIGH,
                subject="trend",
                message=f"Quiz skor trendi deneme başına %{trend:.1f} düşüyor.",
            )
        )

    result.problems = problems[:MAX_PROBLEMS]


def _student_recommendations(result: StudentResult, payload: StudentInput) -> None:
    recommendations: List[RecommendationOutput] = []
    priority = 1

    for problem in result.problems:
        if problem.type == "low_attendance":
            recommendations.append(
                RecommendationOutput(
                    type="attendance_recovery",
                    severity=problem.severity,
                    subject="attendance",
                    title="Devam takibi başlat",
                    message=problem.message,
                    action="Öğretmen ile görüşüp devamsızlık nedenini çöz ve eksik dersleri telafi et.",
                    priority=priority,
                )
            )
        elif problem.type == "incomplete_lessons":
            recommendations.append(
                RecommendationOutput(
                    type="lesson_backlog",
                    severity=problem.severity,
                    subject="completion",
                    title="Tamarlanmamış dersleri bitir",
                    message=problem.message,
                    action="Kaldığın dersten başlayarak kalan konuları tamamla.",
                    priority=priority,
                )
            )
        elif problem.type == "weak_topic":
            recommendations.append(
                RecommendationOutput(
                    type="topic_review",
                    severity=problem.severity,
                    subject="topic",
                    topic_id=problem.topic_id,
                    title="Konu tekrarı",
                    message=problem.message,
                    action="İlgili konunun örneklerini çözüp tekrar quiz çöz.",
                    priority=priority,
                )
            )
        elif problem.type == "no_activity":
            recommendations.append(
                RecommendationOutput(
                    type="engage",
                    severity=problem.severity,
                    subject="activity",
                    title="Düzenli çalışma planı",
                    message=problem.message,
                    action="Haftada en az üç gün ders görüntüle ve bir quiz çöz.",
                    priority=priority,
                )
            )
        elif problem.type == "declining_trend":
            recommendations.append(
                RecommendationOutput(
                    type="trend_support",
                    severity=problem.severity,
                    subject="trend",
                    title="Puan düşüşü",
                    message=problem.message,
                    action="Öğretmenden birebir destek talep et.",
                    priority=priority,
                )
            )
        priority += 1

    if not recommendations and result.overall_status in (STATUS_EXCELLENT, STATUS_GOOD):
        recommendations.append(
            RecommendationOutput(
                type="maintain",
                severity=SEVERITY_INFO,
                subject="overall",
                title="Dengeli ilerleme",
                message="Göstergeler hedef aralıkta.",
                action="Mevcut tempoyu sürdür, zayıf konuları önceliklendir.",
                priority=1,
            )
        )

    result.recommendations = recommendations[:MAX_RECOMMENDATIONS]


def analyze_teacher(payload: TeacherInput) -> TeacherResult:
    result = TeacherResult(subject_id=payload.teacher.id, generated_at=now())
    students = payload.students

    participation_threshold = PASS_SCORE
    participating = [student for student in students if student.score >= 0 and student.activity_events > 0]

    for course in payload.courses:
        course_students = [student for student in students if student.class_id in course.class_ids] if course.class_ids else students
        average = clamp(
            sum(student.score for student in course_students) / len(course_students)
        ) if course_students else clamp(course.average_student_score)
        previous = clamp(course.previous_average_student_score)
        improvement = round(average - previous, 2) if previous > 0 else 0.0
        course_students_average = average if course_students else clamp(course.average_student_score)
        score = clamp(course_students_average)
        result.courses.append(
            TeacherCourseResult(
                course_id=course.course_id,
                name=course.name,
                score=score,
                status=status_for(score, bool(course_students) or course.student_count > 0),
                class_count=len(course.class_ids),
                student_count=course.student_count or len(course_students),
                average_student_score=course_students_average,
                improvement=improvement,
            )
        )

    class_scores: Dict[str, List] = {}
    for student in students:
        if student.class_id:
            class_scores.setdefault(student.class_id, []).append(student)

    for teacher_class in payload.classes:
        members = class_scores.get(teacher_class.class_id, [])
        average = clamp(sum(student.score for student in members) / len(members)) if members else 0.0
        attendance = clamp(
            sum(student.attendance_rate for student in members) / len(members)
        ) if members else 0.0
        participation = clamp(
            sum(min(100.0, student.activity_events * 10.0) for student in members) / len(members)
        ) if members else 0.0
        score = weighted([(average, 60.0), (attendance, 25.0), (participation, 15.0)])
        result.classes.append(
            TeacherClassResult(
                class_id=teacher_class.class_id,
                name=teacher_class.name,
                score=score,
                status=status_for(score, bool(members) or teacher_class.student_count > 0),
                student_count=teacher_class.student_count or len(members),
                attendance=attendance,
                participation=participation,
            )
        )

    average_score = clamp(sum(student.score for student in students) / len(students)) if students else 0.0
    improvements = [
        student.score - student.previous_score
        for student in students
        if student.previous_score is not None
    ]
    result.student_outcomes = StudentOutcomeResult(
        average_score=average_score,
        improvement=round(sum(improvements) / len(improvements), 2) if improvements else 0.0,
        participating_rate=percent(len(participating), len(students)),
        participating_count=len(participating),
        status=status_for(average_score, bool(students)),
    )

    content_value, has_content = content_score(payload.content)
    result.content = metric(
        content_value,
        has_content,
        lessons_created=payload.content.lessons_created,
        lessons_approved=payload.content.lessons_approved,
        lessons_pending=payload.content.lessons_pending,
        quizzes_created=payload.content.quizzes_created,
        quizzes_approved=payload.content.quizzes_approved,
        questions_created=payload.content.questions_created,
    )

    has_attendance = payload.attendance.records > 0
    result.attendance = metric(
        payload.attendance.rate,
        has_attendance,
        records=payload.attendance.records,
        present=payload.attendance.present,
        absent=payload.attendance.absent,
        late=payload.attendance.late,
        assigned_classes=payload.attendance.assigned_classes,
        coverage=percent(payload.attendance.classes_with_record, max(payload.attendance.assigned_classes, 1)),
    )

    activity_value, has_activity = teacher_activity_score(payload.activity, len(payload.classes))
    result.activity = metric(
        activity_value,
        has_activity,
        content_events=payload.activity.content_events,
        attendance_taken=payload.activity.attendance_taken,
        notes_created=payload.activity.notes_created,
        active_days=payload.activity.active_days,
    )

    quiz_scores = [quiz.average_score for quiz in payload.quizzes if quiz.attempts > 0]
    pass_rates = [quiz.pass_rate for quiz in payload.quizzes if quiz.attempts > 0]
    has_quizzes = bool(quiz_scores)
    result.quiz_metrics = metric(
        clamp(sum(quiz_scores) / len(quiz_scores)) if quiz_scores else 0.0,
        has_quizzes,
        quiz_count=len(payload.quizzes),
        average_score=clamp(sum(quiz_scores) / len(quiz_scores)) if quiz_scores else 0.0,
        pass_rate=clamp(sum(pass_rates) / len(pass_rates)) if pass_rates else 0.0,
        attempts=float(sum(quiz.attempts for quiz in payload.quizzes)),
    )

    components = [
        (result.student_outcomes.average_score, 40.0 if students else 0.0),
        (result.attendance.score, ATTENDANCE_WEIGHT if has_attendance else 0.0),
        (result.activity.score, ACTIVITY_WEIGHT if has_activity else 0.0),
        (result.content.score, 10.0 if has_content else 0.0),
        (result.quiz_metrics.score, 10.0 if has_quizzes else 0.0),
    ]
    overall = weighted(components)
    result.overall_score = overall
    result.overall_status = status_for(overall, any(weight > 0 for _, weight in components))

    _teacher_problems(result, payload, overall)
    _teacher_recommendations(result, payload, overall)
    result.metrics = {
        "overall_score": overall,
        "class_count": float(len(result.classes)),
        "course_count": float(len(result.courses)),
        "student_count": float(len(students)),
        "average_student_score": result.student_outcomes.average_score,
        "participating_rate": result.student_outcomes.participating_rate,
        "attendance_rate": clamp(payload.attendance.rate),
        "activity_score": result.activity.score,
        "content_score": result.content.score,
        "quiz_average": result.quiz_metrics.score,
        "previous_score": clamp(payload.previous_score) if payload.previous_score is not None else 0.0,
        "improvement": round(overall - payload.previous_score, 2) if payload.previous_score is not None else 0.0,
    }
    return result


def _teacher_problems(result: TeacherResult, payload: TeacherInput, overall: float) -> None:
    problems: List[ProblemOutput] = []

    weak_courses = [course for course in result.courses if course.status in (STATUS_WEAK, STATUS_NEEDS_ATTENTION)]
    for course in weak_courses[:3]:
        problems.append(
            ProblemOutput(
                type="course_performance",
                severity=SEVERITY_HIGH if course.status == STATUS_WEAK else SEVERITY_MEDIUM,
                subject="course",
                message=f"{course.name} ortalaması %{course.average_student_score:.0f}.",
            )
        )

    if payload.attendance.records > 0 and payload.attendance.rate < TARGET_ATTENDANCE:
        problems.append(
            ProblemOutput(
                type="attendance_gap",
                severity=SEVERITY_MEDIUM,
                subject="attendance",
                message=(
                    f"Sınıfların %{payload.attendance.rate:.0f}'i yok; "
                    f"{payload.attendance.classes_with_record}/{payload.attendance.assigned_classes} sınıf kayıtlı."
                ),
            )
        )

    pending = payload.content.lessons_pending + payload.content.quizzes_pending
    if pending > 0:
        problems.append(
            ProblemOutput(
                type="pending_content",
                severity=SEVERITY_INFO,
                subject="content",
                message=f"Onay bekleyen {pending} içerik var.",
            )
        )

    if payload.previous_score is not None and overall - payload.previous_score <= -5.0:
        problems.append(
            ProblemOutput(
                type="declining_trend",
                severity=SEVERITY_MEDIUM,
                subject="trend",
                message=f"Önceki genel puan %{payload.previous_score:.0f}, güncel %{overall:.0f}.",
            )
        )

    if not problems and result.overall_status == STATUS_UNKNOWN:
        problems.append(
            ProblemOutput(
                type="insufficient_data",
                severity=SEVERITY_INFO,
                subject="overall",
                message="Analiz için yeterli veri yok.",
            )
        )

    result.problems = problems[:MAX_PROBLEMS]


def _teacher_recommendations(result: TeacherResult, payload: TeacherInput, overall: float) -> None:
    recommendations: List[RecommendationOutput] = []
    priority = 1

    for problem in result.problems:
        if problem.type == "course_performance":
            course = next((item for item in result.courses if item.name and item.name in problem.message), None)
            recommendations.append(
                RecommendationOutput(
                    type="course_support",
                    severity=problem.severity,
                    subject="course",
                    title="Derse özel destek",
                    message=problem.message,
                    action="Konu bazlı tekrar planı hazırla ve sınav öncesi destek oturumu yap.",
                    priority=priority,
                )
            )
        elif problem.type == "attendance_gap":
            recommendations.append(
                RecommendationOutput(
                    type="attendance_followup",
                    severity=problem.severity,
                    subject="attendance",
                    title="Devamsız sınıfları takip et",
                    message=problem.message,
                    action="Kayıt tutulmayan sınıflarda yoklama al ve veliye bildir.",
                    priority=priority,
                )
            )
        elif problem.type == "pending_content":
            recommendations.append(
                RecommendationOutput(
                    type="content_review",
                    severity=problem.severity,
                    subject="content",
                    title="Onay bekleyen içerikleri tamamla",
                    message=problem.message,
                    action="İçerikleri inceleyip onayla veya düzeltme iste.",
                    priority=priority,
                )
            )
        elif problem.type == "declining_trend":
            recommendations.append(
                RecommendationOutput(
                    type="trend_support",
                    severity=problem.severity,
                    subject="trend",
                    title="Genel düşüş",
                    message=problem.message,
                    action="Sınıf bazlı değerlendirme yapıp müfredatı gözden geçir.",
                    priority=priority,
                )
            )
        elif problem.type == "insufficient_data":
            recommendations.append(
                RecommendationOutput(
                    type="collect_data",
                    severity=problem.severity,
                    subject="overall",
                    title="Veri toplama",
                    message=problem.message,
                    action="Quiz ve yoklama kayıtlarını düzenli tut.",
                    priority=priority,
                )
            )
        priority += 1

    if not recommendations:
        recommendations.append(
            RecommendationOutput(
                type="maintain",
                severity=SEVERITY_INFO,
                subject="overall",
                title="Süreklilik",
                message=f"Genel puan %{overall:.0f}; hedef aralıkta.",
                action="Mevcut uygulamaları sürdür.",
                priority=1,
            )
        )

    result.recommendations = recommendations[:MAX_RECOMMENDATIONS]


def ping() -> bool:
    return True


__all__ = ["analyze_student", "analyze_teacher", "ping", "status_for", "clamp", "percent"]
