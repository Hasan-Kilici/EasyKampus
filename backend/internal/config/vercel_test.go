package config

import "testing"

func TestResolveDatabasePath(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"local default", map[string]string{}, "./lms.db"},
		{"local custom", map[string]string{"DATABASE_PATH": "./x.db"}, "./x.db"},
		{"vercel ignores read-only relative path", map[string]string{"VERCEL": "1", "DATABASE_PATH": "./lms.db"}, "/tmp/lms.db"},
		{"vercel keeps /tmp path", map[string]string{"VERCEL": "1", "DATABASE_PATH": "/tmp/a.db"}, "/tmp/a.db"},
		{"vercel default", map[string]string{"VERCEL": "1"}, "/tmp/lms.db"},
	}
	for _, tt := range tests {
		if got := resolveDatabasePath(env(tt.env)); got != tt.want {
			t.Errorf("%s: got %q want %q", tt.name, got, tt.want)
		}
	}
}
