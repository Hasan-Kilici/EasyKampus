package courses

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/base"
	"lms/backend/pkg/domain/topics"
)

func modelWithID() base.Model { return base.Model{ID: uuid.New()} }

func sampleTopics() []topics.Topic {
	return []topics.Topic{
		{Model: modelWithID(), Name: "Değişkenler", Active: true, StudyHours: 10},
		{Model: modelWithID(), Name: "Döngüler", Active: true, StudyHours: 12.5},
		{Model: modelWithID(), Name: "Pasif konu", Active: false, StudyHours: 99},
	}
}

func TestComputeWorkloadSumsActiveTopicsHomeworkAndProject(t *testing.T) {
	course := Course{HomeworkHours: 20, ProjectHours: 17.5}
	got, err := ComputeWorkload(course, sampleTopics(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TopicHours != 22.5 || got.TotalHours != 60 {
		t.Fatalf("topic hours %.2f total %.2f, want 22.5 and 60 (inactive topic must be ignored)", got.TopicHours, got.TotalHours)
	}
	if got.HoursPerECTS != 30 || got.ECTS != 2 || got.ECTSExact != 2 {
		t.Fatalf("ects %.2f exact %.2f at %.0f h/ects, want 2 / 2 / 30", got.ECTS, got.ECTSExact, got.HoursPerECTS)
	}
	if len(got.Topics) != 2 {
		t.Fatalf("expected the 2 active topics listed, got %d", len(got.Topics))
	}
}

func TestComputeWorkloadRoundsToNearestHalfECTS(t *testing.T) {
	cases := []struct {
		hours, perECTS float64
		want, exact    float64
	}{
		{hours: 74, perECTS: 30, want: 2.5, exact: 2.47},
		{hours: 140, perECTS: 25, want: 5.5, exact: 5.6},
		{hours: 0, perECTS: 30, want: 0, exact: 0},
		{hours: 7, perECTS: 30, want: 0, exact: 0.23},
	}
	for _, tc := range cases {
		got, err := ComputeWorkload(Course{ProjectHours: tc.hours}, nil, tc.perECTS)
		if err != nil {
			t.Fatalf("%v: %v", tc, err)
		}
		if got.ECTS != tc.want || got.ECTSExact != tc.exact {
			t.Errorf("%.0f h at %.0f: ects %.2f exact %.2f, want %.2f / %.2f", tc.hours, tc.perECTS, got.ECTS, got.ECTSExact, tc.want, tc.exact)
		}
	}
}

func TestComputeWorkloadRejectsOutOfRangeHoursPerECTS(t *testing.T) {
	for _, value := range []float64{10, 24.9, 30.1, 60, -5} {
		if _, err := ComputeWorkload(Course{}, nil, value); !errors.Is(err, core.ErrValidation) {
			t.Errorf("hours_per_ects %.1f should be rejected, got %v", value, err)
		}
	}
	for _, value := range []float64{25, 27.5, 30} {
		if _, err := ComputeWorkload(Course{}, nil, value); err != nil {
			t.Errorf("hours_per_ects %.1f should be accepted, got %v", value, err)
		}
	}
}

func TestCourseValidateChecksHoursAndDepartment(t *testing.T) {
	valid := Course{SchoolID: uuid.New(), Name: "Programlama", Code: "PRG-101", Department: "Bilgisayar Programcılığı"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid course rejected: %v", err)
	}
	bad := []Course{valid, valid, valid}
	bad[0].HomeworkHours = -1
	bad[1].ProjectHours = MaxCourseHours + 1
	bad[2].Department = string(make([]byte, 151))
	for index, course := range bad {
		if err := course.Validate(); !errors.Is(err, core.ErrValidation) {
			t.Errorf("case %d should be rejected, got %v", index, err)
		}
	}
}
