package config

import (
	"testing"
	"time"
)

func env(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestResolvePerformanceURL(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"unset locally uses the dev default", map[string]string{}, "http://127.0.0.1:8000"},
		{"unset in development uses the dev default", map[string]string{"APP_ENV": "development"}, "http://127.0.0.1:8000"},
		{"unset in production off vercel stays empty", map[string]string{"APP_ENV": "production"}, ""},
		{"explicit wins and is trimmed", map[string]string{"PERFORMANCE_SERVICE_URL": " http://127.0.0.1:8000/ ", "VERCEL": "1", "VERCEL_URL": "x.vercel.app"}, "http://127.0.0.1:8000"},
		{"vercel preview uses deployment url", map[string]string{"VERCEL": "1", "VERCEL_ENV": "preview", "VERCEL_URL": "lms-abc.vercel.app", "VERCEL_PROJECT_PRODUCTION_URL": "lms.vercel.app"}, "https://lms-abc.vercel.app"},
		{"vercel production uses production domain", map[string]string{"VERCEL": "1", "VERCEL_ENV": "production", "VERCEL_URL": "lms-abc.vercel.app", "VERCEL_PROJECT_PRODUCTION_URL": "lms.vercel.app"}, "https://lms.vercel.app"},
		{"vercel production without production domain falls back", map[string]string{"VERCEL": "1", "VERCEL_ENV": "production", "VERCEL_URL": "lms-abc.vercel.app"}, "https://lms-abc.vercel.app"},
		{"vercel without any host stays empty (no localhost fallback)", map[string]string{"VERCEL": "1"}, ""},
		{"scheme is kept when present", map[string]string{"VERCEL": "1", "VERCEL_URL": "http://localhost:3000/"}, "http://localhost:3000"},
		{"vercel vars ignored when not on vercel", map[string]string{"VERCEL_URL": "lms-abc.vercel.app"}, "http://127.0.0.1:8000"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolvePerformanceURL(env(tc.env)); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolvePerformanceTimeout(t *testing.T) {
	cases := []struct {
		name    string
		env     map[string]string
		want    time.Duration
		wantErr bool
	}{
		{"local default", map[string]string{}, 15 * time.Second, false},
		{"vercel default is shorter", map[string]string{"VERCEL": "1"}, 8 * time.Second, false},
		{"explicit wins on vercel", map[string]string{"VERCEL": "1", "PERFORMANCE_TIMEOUT": "20s"}, 20 * time.Second, false},
		{"explicit locally", map[string]string{"PERFORMANCE_TIMEOUT": "3s"}, 3 * time.Second, false},
		{"invalid", map[string]string{"PERFORMANCE_TIMEOUT": "soon"}, 0, true},
		{"non-positive", map[string]string{"PERFORMANCE_TIMEOUT": "0s"}, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolvePerformanceTimeout(env(tc.env))
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolvePerformanceBypassSecret(t *testing.T) {
	if got := resolvePerformanceBypassSecret(env(map[string]string{})); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
	if got := resolvePerformanceBypassSecret(env(map[string]string{"VERCEL_AUTOMATION_BYPASS_SECRET": "auto"})); got != "auto" {
		t.Fatalf("got %q, want auto", got)
	}
	if got := resolvePerformanceBypassSecret(env(map[string]string{"PERFORMANCE_BYPASS_SECRET": "mine", "VERCEL_AUTOMATION_BYPASS_SECRET": "auto"})); got != "mine" {
		t.Fatalf("got %q, want mine", got)
	}
}
