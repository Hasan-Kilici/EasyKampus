"""Sözleşme ve hesaplama testleri (pytest)."""

from __future__ import annotations

from datetime import datetime, timezone

import pytest
from fastapi.testclient import TestClient

from app.analysis import analyze_student, analyze_teacher, linear_trend, status_for
from app.main import app
from app.models import (
    AttendanceCourse,
    AttendanceInput,
    CourseCompletion,
    CourseRef,
    LessonInput,
    QuizResult,
    StudentInput,
    StudentRef,
    TeacherAttendance,
    TeacherContent,
    TeacherInput,
    TeacherRef,
    TopicRef,
)

client = TestClient(app)


def test_health_reports_contract_version() -> None:
    response = client.get("/api/health")
    assert response.status_code == 200
    body = response.json()
    assert body["status"] == "ok"
    assert body["contract_version"] == 1


def test_status_thresholds() -> None:
    assert status_for(92) == "excellent"
    assert status_for(75) == "good"
    assert status_for(55) == "needs_attention"
    assert status_for(20) == "weak"
    assert status_for(0, has_data=False) == "unknown"


def test_student_analysis_flags_weak_topic_and_attendance() -> None:
    now = datetime.now(timezone.utc)
    payload = StudentInput(
        student=StudentRef(id="s1", name="Ada", class_ids=["c1"]),
        classes=[],
        courses=[CourseRef(id="co1", name="Matematik", class_ids=["c1"], topics=[TopicRef(id="t1", name="Oran")])],
        quiz_results=[
            QuizResult(
                attempt_id="a1",
                quiz_id="q1",
                course_id="co1",
                class_id="c1",
                name="Quiz 1",
                score=4,
                max_score=10,
                started_at=now,
                completed_at=now,
                duration_seconds=600,
                answers=[
                    {"question_id": "q1a", "topic_id": "t1", "correct": True, "points": 4},
                    {"question_id": "q1b", "topic_id": "t1", "correct": False, "points": 0},
                ],
            )
        ],
        attendance=AttendanceInput(
            total=10,
            present=5,
            absent=5,
            rate=50.0,
            per_course=[AttendanceCourse(course_id="co1", total=10, present=5, absent=5, rate=50.0)],
        ),
        lessons=LessonInput(
            total=10,
            completed=4,
            completion_rate=40.0,
            per_course=[CourseCompletion(course_id="co1", total=10, completed=4, completion_rate=40.0)],
        ),
    )

    result = analyze_student(payload)

    assert result.version == 1
    assert result.subject_id == "s1"
    assert len(result.courses) == 1
    assert result.courses[0].quiz_count == 1
    assert result.weak_areas and result.weak_areas[0].topic_id == "t1"
    assert result.weak_areas[0].score == 50.0
    assert any(problem.type == "low_attendance" for problem in result.problems)
    assert any(problem.type == "incomplete_lessons" for problem in result.problems)
    assert any(item.type == "topic_review" for item in result.recommendations)
    assert 0 <= result.overall_score <= 100


def test_student_analysis_without_data_is_unknown() -> None:
    result = analyze_student(StudentInput(student=StudentRef(id="s-empty")))
    assert result.overall_status == "unknown"
    assert result.courses == []
    assert result.metrics["overall_score"] == 0.0
    assert result.recommendations


def test_linear_trend_detects_decline() -> None:
    now = datetime.now(timezone.utc)
    quizzes = [
        QuizResult(attempt_id="a1", quiz_id="q1", course_id="c1", class_id="cl1", score=90, max_score=100, completed_at=now),
        QuizResult(attempt_id="a2", quiz_id="q2", course_id="c1", class_id="cl1", score=70, max_score=100, completed_at=now),
        QuizResult(attempt_id="a3", quiz_id="q3", course_id="c1", class_id="cl1", score=50, max_score=100, completed_at=now),
    ]
    slope, has_data = linear_trend(quizzes)
    assert has_data is True
    assert slope == -20.0


def test_student_endpoint_round_trip() -> None:
    response = client.post(
        "/api/analyze/student",
        json={"student": {"id": "s2", "name": "Grace"}, "quiz_results": []},
    )
    assert response.status_code == 200
    body = response.json()
    assert body["subject_id"] == "s2"
    assert "overall_status" in body and "metrics" in body


def test_teacher_analysis_scores_classes_and_courses() -> None:
    payload = TeacherInput(
        teacher=TeacherRef(id="t1", name="Grace"),
        content=TeacherContent(
            lessons_created=10,
            lessons_approved=9,
            lessons_rejected=1,
            quizzes_created=4,
            quizzes_approved=4,
        ),
        students=[
            {"student_id": "s1", "class_id": "c1", "score": 90.0, "previous_score": 80.0, "attendance_rate": 95.0, "activity_events": 12},
            {"student_id": "s2", "class_id": "c1", "score": 70.0, "previous_score": 75.0, "attendance_rate": 70.0, "activity_events": 4},
        ],
        attendance=TeacherAttendance(
            records=40,
            present=34,
            absent=6,
            rate=85.0,
            classes_with_record=2,
            assigned_classes=2,
        ),
    )

    result = analyze_teacher(payload)

    assert result.subject_id == "t1"
    assert result.student_outcomes.average_score == 80.0
    assert result.student_outcomes.improvement == 2.5
    assert result.student_outcomes.participating_count == 2
    assert result.content.status in ("excellent", "good")
    assert 0 <= result.overall_score <= 100


def test_teacher_endpoint_round_trip() -> None:
    response = client.post(
        "/api/analyze/teacher",
        json={"teacher": {"id": "t-empty"}},
    )
    assert response.status_code == 200
    body = response.json()
    assert body["subject_id"] == "t-empty"
    assert body["overall_status"] == "unknown"


def test_contract_rejects_unsupported_version() -> None:
    response = client.post(
        "/api/analyze/student",
        json={"version": 2, "student": {"id": "s-version"}},
    )
    assert response.status_code == 422


def test_contract_rejects_inconsistent_attendance_counts() -> None:
    response = client.post(
        "/api/analyze/student",
        json={
            "student": {"id": "s-attendance"},
            "attendance": {"total": 2, "present": 2, "absent": 1},
        },
    )
    assert response.status_code == 422


def test_contract_fields_match_go_names() -> None:
    """Go sözleşmesindeki ana alan adları Python tarafında da aynı."""
    student = analyze_student(StudentInput(student=StudentRef(id="s3")))
    teacher = analyze_teacher(TeacherInput(teacher=TeacherRef(id="t3")))
    assert set(student.model_dump()) >= {
        "version",
        "subject_id",
        "overall_score",
        "overall_status",
        "courses",
        "attendance",
        "activity",
        "completion",
        "weak_areas",
        "problems",
        "recommendations",
        "metrics",
        "generated_at",
    }
    assert set(teacher.model_dump()) >= {
        "version",
        "subject_id",
        "overall_score",
        "overall_status",
        "courses",
        "classes",
        "student_outcomes",
        "activity",
        "attendance",
        "content",
        "quiz_metrics",
        "problems",
        "recommendations",
        "metrics",
        "generated_at",
    }


if __name__ == "__main__":
    raise SystemExit(pytest.main([__file__, "-q"]))


def test_analyze_is_open_when_internal_token_unset(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.delenv("PERFORMANCE_INTERNAL_TOKEN", raising=False)
    response = client.post("/api/analyze/student", json={"student": {"id": "open"}})
    assert response.status_code == 200


def test_analyze_requires_internal_token_when_configured(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("PERFORMANCE_INTERNAL_TOKEN", "s3cret")
    payload = {"student": {"id": "guarded"}}
    assert client.post("/api/analyze/student", json=payload).status_code == 401
    wrong = client.post("/api/analyze/student", json=payload, headers={"X-Internal-Token": "nope"})
    assert wrong.status_code == 401
    right = client.post("/api/analyze/student", json=payload, headers={"X-Internal-Token": "s3cret"})
    assert right.status_code == 200
    teacher = client.post("/api/analyze/teacher", json={"teacher": {"id": "t"}}, headers={"X-Internal-Token": "s3cret"})
    assert teacher.status_code == 200


def test_health_stays_open_when_internal_token_configured(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("PERFORMANCE_INTERNAL_TOKEN", "s3cret")
    assert client.get("/api/health").status_code == 200


def test_vercel_entrypoint_imports_from_any_cwd(tmp_path, monkeypatch: pytest.MonkeyPatch) -> None:
    import importlib.util
    import pathlib

    monkeypatch.chdir(tmp_path)
    path = pathlib.Path(__file__).resolve().parents[1] / "api" / "index.py"
    spec = importlib.util.spec_from_file_location("vercel_index", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    assert module.app is app


def test_course_and_topic_accept_department_and_description_fields() -> None:
    response = client.post(
        "/api/analyze/student",
        json={
            "student": {"id": "s-dept"},
            "courses": [
                {
                    "id": "c1",
                    "name": "Programlama",
                    "department": "Bilgisayar Programcılığı",
                    "department_keywords": "yazılım, donanım",
                    "topics": [{"id": "t1", "name": "Değişkenler", "description": "Değişken tanımlama ve veri tipleri anlatılır.", "weight": 1}],
                }
            ],
        },
    )
    assert response.status_code == 200
    # Python servisi bu alanları ilgi puanı için kullanmaz; yalnızca kabul eder.
    assert "relevance" not in str(response.json().get("courses", []))
