package quizes

import (
	"testing"
	"time"
)

func TestDeadline(t *testing.T) {
	started := time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)
	closeAt := started.Add(2 * time.Hour)
	soon := started.Add(10 * time.Minute)

	cases := []struct {
		name string
		quiz Quiz
		want time.Time
		ok   bool
	}{
		{"duration only", Quiz{DurationMinutes: 30}, started.Add(30 * time.Minute), true},
		{"untimed with close time", Quiz{CloseAt: &closeAt}, closeAt, true},
		{"duration shorter than close time", Quiz{DurationMinutes: 30, CloseAt: &closeAt}, started.Add(30 * time.Minute), true},
		{"close time shorter than duration", Quiz{DurationMinutes: 30, CloseAt: &soon}, soon, true},
		{"no limit at all", Quiz{}, time.Time{}, false},
	}
	for _, tc := range cases {
		got, ok := tc.quiz.Deadline(started)
		if ok != tc.ok || !got.Equal(tc.want) {
			t.Errorf("%s: got %v %v, want %v %v", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}
