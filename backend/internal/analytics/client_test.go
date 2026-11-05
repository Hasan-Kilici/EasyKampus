package analytics

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/performance"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewClient(server.URL, 2*time.Second), server
}

func TestAnalyzeStudentMapsPlatformFailuresToUnavailable(t *testing.T) {
	cases := []struct {
		name   string
		status int
		want   error
	}{
		{"service unavailable", http.StatusServiceUnavailable, core.ErrUnavailable},
		{"not implemented", http.StatusNotImplemented, core.ErrUnavailable},
		{"internal error (function crash)", http.StatusInternalServerError, core.ErrUnavailable},
		{"bad gateway", http.StatusBadGateway, core.ErrUnavailable},
		{"gateway timeout (function duration)", http.StatusGatewayTimeout, core.ErrUnavailable},
		{"deployment protection", http.StatusUnauthorized, core.ErrUnavailable},
		{"forbidden", http.StatusForbidden, core.ErrUnavailable},
		{"route missing", http.StatusNotFound, core.ErrUnavailable},
		{"rate limited", http.StatusTooManyRequests, core.ErrUnavailable},
		{"redirect to a login page", http.StatusTemporaryRedirect, core.ErrUnavailable},
		{"contract validation error", http.StatusUnprocessableEntity, core.ErrValidation},
		{"bad request", http.StatusBadRequest, core.ErrValidation},
		{"payload too large", http.StatusRequestEntityTooLarge, core.ErrValidation},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if tc.status >= 300 && tc.status < 400 {
					w.Header().Set("Location", "/login")
				}
				w.WriteHeader(tc.status)
			})
			_, err := client.AnalyzeStudent(context.Background(), performance.StudentInput{})
			if !errors.Is(err, tc.want) {
				t.Fatalf("status %d: got %v, want %v", tc.status, err, tc.want)
			}
		})
	}
}

func TestAnalyzeStudentTreatsNonJSONSuccessAsUnavailable(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<!doctype html><title>Login</title>"))
	})
	_, err := client.AnalyzeStudent(context.Background(), performance.StudentInput{})
	if !errors.Is(err, core.ErrUnavailable) {
		t.Fatalf("got %v, want ErrUnavailable", err)
	}
}

func TestAnalyzeStudentTreatsEmptySuccessBodyAsUnavailable(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	_, err := client.AnalyzeTeacher(context.Background(), performance.TeacherInput{})
	if !errors.Is(err, core.ErrUnavailable) {
		t.Fatalf("got %v, want ErrUnavailable", err)
	}
}

func TestAnalyzeStudentSuccessSendsHeaders(t *testing.T) {
	var got http.Header
	var gotPath string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})
	client.WithInternalToken(" tok ").WithBypassSecret("bypass")
	if _, err := client.AnalyzeStudent(context.Background(), performance.StudentInput{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/api/analyze/student" {
		t.Fatalf("path = %q", gotPath)
	}
	if got.Get("X-Contract-Version") == "" {
		t.Fatal("missing X-Contract-Version")
	}
	if got.Get("X-Internal-Token") != "tok" {
		t.Fatalf("X-Internal-Token = %q", got.Get("X-Internal-Token"))
	}
	if got.Get("x-vercel-protection-bypass") != "bypass" {
		t.Fatalf("bypass header = %q", got.Get("x-vercel-protection-bypass"))
	}
}

func TestSecretHeadersAreOmittedWhenNotConfigured(t *testing.T) {
	var got http.Header
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		_, _ = w.Write([]byte(`{}`))
	})
	if _, err := client.AnalyzeTeacher(context.Background(), performance.TeacherInput{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Get("X-Internal-Token") != "" || got.Get("x-vercel-protection-bypass") != "" {
		t.Fatalf("secret headers leaked without configuration: %v", got)
	}
}

func TestEmptyBaseURLIsUnavailable(t *testing.T) {
	client := NewClient("", time.Second)
	if _, err := client.AnalyzeStudent(context.Background(), performance.StudentInput{}); !errors.Is(err, core.ErrUnavailable) {
		t.Fatalf("analyze: got %v", err)
	}
	if err := client.Ping(context.Background()); !errors.Is(err, core.ErrUnavailable) {
		t.Fatalf("ping: got %v", err)
	}
}

func TestBaseURLStripsTrailingAPI(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client := NewClient(server.URL+"/api/", time.Second)
	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
	if gotPath != "/api/health" {
		t.Fatalf("path = %q, want /api/health", gotPath)
	}
}

func TestPingNonOKIsUnavailable(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) })
	if err := client.Ping(context.Background()); !errors.Is(err, core.ErrUnavailable) {
		t.Fatalf("got %v, want ErrUnavailable", err)
	}
}

func TestTimeoutIsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	client := NewClient(server.URL, 50*time.Millisecond)
	if _, err := client.AnalyzeStudent(context.Background(), performance.StudentInput{}); !errors.Is(err, core.ErrUnavailable) {
		t.Fatalf("got %v, want ErrUnavailable", err)
	}
}
