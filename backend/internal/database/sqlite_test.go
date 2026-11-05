package database

import (
	"strings"
	"testing"
)

func TestSQLiteDSNAddsConnectionParams(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "relative", path: "./lms.db", want: "./lms.db?" + sqliteParams},
		{name: "absolute windows", path: `C:\data\lms.db`, want: `C:\data\lms.db?` + sqliteParams},
		{name: "existing query", path: "./lms.db?mode=rwc", want: "./lms.db?mode=rwc&" + sqliteParams},
		{name: "memory untouched", path: ":memory:", want: ":memory:"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := sqliteDSN(test.path)
			if got != test.want {
				t.Fatalf("sqliteDSN(%q) = %q, want %q", test.path, got, test.want)
			}
			if test.path != ":memory:" {
				for _, param := range []string{"busy_timeout(5000)", "journal_mode(WAL)", "synchronous(NORMAL)", "foreign_keys(1)"} {
					if !strings.Contains(got, param) {
						t.Errorf("dsn %q missing %q", got, param)
					}
				}
			}
		})
	}
}
