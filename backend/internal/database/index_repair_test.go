package database

import (
	"strings"
	"testing"
)

func TestIndexCoversColumns(t *testing.T) {
	columns := []string{"snapshot_id", "course_id"}
	cases := []struct {
		name       string
		definition string
		want       bool
	}{
		{"gorm style with backticks", "CREATE UNIQUE INDEX `idx_snapshot_courses` ON `snapshot_courses`(`snapshot_id`,`course_id`) WHERE deleted_at IS NULL", true},
		{"plain with spaces", "CREATE UNIQUE INDEX idx_snapshot_courses ON snapshot_courses(snapshot_id, course_id) WHERE deleted_at IS NULL", true},
		{"upper case", "CREATE UNIQUE INDEX I ON T(SNAPSHOT_ID, COURSE_ID)", true},
		{"legacy single column", "CREATE UNIQUE INDEX `idx_snapshot_courses` ON `snapshot_courses`(`snapshot_id`) WHERE deleted_at IS NULL", false},
		{"extra trailing column", "CREATE UNIQUE INDEX i ON t(snapshot_id, course_id, topic_id)", false},
		{"extra leading column", "CREATE UNIQUE INDEX i ON t(x, snapshot_id, course_id)", false},
		{"different order", "CREATE UNIQUE INDEX i ON t(course_id, snapshot_id)", false},
		{"empty", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := indexCoversColumns(tc.definition, columns); got != tc.want {
				t.Fatalf("indexCoversColumns(%q) = %v, want %v", tc.definition, got, tc.want)
			}
		})
	}
}

func TestUniqueIndexSpecsAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, spec := range uniqueIndexSpecs {
		if seen[spec.Name] {
			t.Fatalf("duplicate spec %s", spec.Name)
		}
		seen[spec.Name] = true
		statement := spec.createStatement()
		if !strings.HasPrefix(statement, "CREATE UNIQUE INDEX "+spec.Name+" ON "+spec.Table+"(") {
			t.Fatalf("unexpected statement %q", statement)
		}
		if !indexCoversColumns(statement, spec.Columns) {
			t.Fatalf("statement %q does not cover its own columns", statement)
		}
		if spec.Where != "" && !strings.HasSuffix(statement, " WHERE "+spec.Where) {
			t.Fatalf("statement %q missing WHERE", statement)
		}
		if spec.Where == "" && strings.Contains(statement, "WHERE") {
			t.Fatalf("statement %q has unexpected WHERE", statement)
		}
	}
	for _, name := range []string{"idx_snapshot_courses", "idx_snapshot_topics", "idx_classes_school_level_no"} {
		if !seen[name] {
			t.Fatalf("missing spec %s", name)
		}
	}
}
