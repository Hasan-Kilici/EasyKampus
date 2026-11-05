package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/performance"
)

// Client, Go servisinin ölçülebilir girdilerini ayrı Python analiz servisine
// taşır; puanlama ve öneri üretimi yalnızca bu servis tarafından yapılır.
type Client struct {
	baseURL       string
	version       int
	http          *http.Client
	internalToken string
	bypassSecret  string
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "/api")
	if baseURL == "" {
		log.Printf("analytics: PERFORMANCE_SERVICE_URL is empty; performance endpoints will return 503")
		return &Client{version: performance.ContractVersion}
	}
	log.Printf("analytics: performance service at %s (timeout %s)", baseURL, timeout)
	return &Client{
		baseURL: baseURL,
		version: performance.ContractVersion,
		http: &http.Client{
			Timeout: timeout,
			// Yönlendirmeyi takip etme: Vercel Deployment Protection gibi katmanlar
			// bir giriş sayfasına yönlendirir; bunu başarılı yanıt sanmamalıyız.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

// WithInternalToken, performans servisindeki PERFORMANCE_INTERNAL_TOKEN ile
// eşleşen X-Internal-Token başlığını her isteğe ekler. Boşsa başlık gönderilmez.
func (client *Client) WithInternalToken(token string) *Client {
	client.internalToken = strings.TrimSpace(token)
	return client
}

// WithBypassSecret, Vercel Deployment Protection arkasındaki dağıtımlarda
// servis çağrısının geçebilmesi için x-vercel-protection-bypass başlığını ekler.
func (client *Client) WithBypassSecret(secret string) *Client {
	client.bypassSecret = strings.TrimSpace(secret)
	return client
}

func (client *Client) authorize(request *http.Request) {
	if client.internalToken != "" {
		request.Header.Set("X-Internal-Token", client.internalToken)
	}
	if client.bypassSecret != "" {
		request.Header.Set("x-vercel-protection-bypass", client.bypassSecret)
	}
}

func (client *Client) Ping(ctx context.Context) error {
	if client.baseURL == "" {
		return core.ErrUnavailable
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+"/api/health", nil)
	if err != nil {
		return translate(err)
	}
	client.authorize(request)
	response, err := client.http.Do(request)
	if err != nil {
		return translate(err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode != http.StatusOK {
		return core.ErrUnavailable
	}
	return nil
}

func (client *Client) AnalyzeStudent(ctx context.Context, input performance.StudentInput) (*performance.StudentResult, error) {
	var result performance.StudentResult
	if err := client.post(ctx, "/api/analyze/student", input, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (client *Client) AnalyzeTeacher(ctx context.Context, input performance.TeacherInput) (*performance.TeacherResult, error) {
	var result performance.TeacherResult
	if err := client.post(ctx, "/api/analyze/teacher", input, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (client *Client) post(ctx context.Context, path string, payload any, target any) error {
	if client.baseURL == "" {
		return core.ErrUnavailable
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return translate(err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return translate(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-Contract-Version", fmt.Sprintf("%d", client.version))
	client.authorize(request)

	response, err := client.http.Do(request)
	if err != nil {
		log.Printf("analytics %s request failed: %v", path, err)
		return translate(err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		log.Printf("analytics %s responded %d: %s", path, response.StatusCode, strings.TrimSpace(string(detail)))
		if unavailableStatus(response.StatusCode) {
			return core.ErrUnavailable
		}
		return core.ErrValidation
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		// 2xx ama JSON olmayan gövde: yanıt veren şey performans servisi değildir
		// (örn. bir koruma/giriş sayfası). Bu bir doğrulama hatası değil, servis yokluğudur.
		var syntaxError *json.SyntaxError
		if errors.As(err, &syntaxError) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			log.Printf("analytics %s returned a non-JSON body: %v", path, err)
			return core.ErrUnavailable
		}
		return translate(err)
	}
	return nil
}

// unavailableStatus, bağımlı servisin kendisinin ya da önündeki altyapının
// (Vercel koruması, yönlendirme, zaman aşımı, çökme) yanıt veremediği durumları
// ayırır. Bunlar istemci doğrulama hatası değildir ve 503 olarak görünmelidir.
// 400/413/422 gibi gerçek sözleşme/doğrulama hataları ErrValidation kalır.
func unavailableStatus(code int) bool {
	switch code {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
		http.StatusRequestTimeout, http.StatusTooManyRequests:
		return true
	}
	return (code >= 300 && code < 400) || code >= 500
}

func translate(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return core.ErrUnavailable
	}
	var urlError *url.Error
	if errors.As(err, &urlError) {
		return core.ErrUnavailable
	}
	var netError net.Error
	if errors.As(err, &netError) {
		return core.ErrUnavailable
	}
	var jsonError *json.SyntaxError
	if errors.As(err, &jsonError) {
		return core.ErrValidation
	}
	return err
}
