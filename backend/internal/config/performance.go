package config

import (
	"errors"
	"strings"
	"time"
)

// Performans servisi ayarları. Saf fonksiyonlar olarak tutulur ki ortam
// değişkenlerine dokunmadan test edilebilsin.

// defaultLocalPerformanceURL, Python servisinin yerel varsayılan adresidir
// (performance/app/main.py: PERFORMANCE_PORT varsayılanı 8000).
const defaultLocalPerformanceURL = "http://127.0.0.1:8000"

func onVercel(getenv func(string) string) bool {
	return getenv("VERCEL") != ""
}

// resolvePerformanceURL, PERFORMANCE_SERVICE_URL tanımlıysa onu kullanır.
// Tanımsızsa: Vercel dışında ve APP_ENV=production değilse yerel varsayılan
// (http://127.0.0.1:8000) kullanılır. Vercel üzerinde çalışılıyorsa servisi aynı projenin kendi
// adresinde arar: vercel.json, /api/analyze/* ve /api/health isteklerini aynı
// dağıtımdaki Python function'ına yönlendirir. Production'da kararlı üretim
// alan adı, aksi halde dağıtımın kendi adresi kullanılır.
func resolvePerformanceURL(getenv func(string) string) string {
	if value := strings.TrimSpace(getenv("PERFORMANCE_SERVICE_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	if !onVercel(getenv) {
		// Yerel geliştirme: .env eski olsa bile servis varsayılan adreste aranır.
		// (Adres boş kalırsa istemci hiç istek atmadan her çağrıda 503 döner.)
		if getenv("APP_ENV") == "production" {
			return ""
		}
		return defaultLocalPerformanceURL
	}
	host := ""
	if getenv("VERCEL_ENV") == "production" {
		host = strings.TrimSpace(getenv("VERCEL_PROJECT_PRODUCTION_URL"))
	}
	if host == "" {
		host = strings.TrimSpace(getenv("VERCEL_URL"))
	}
	if host == "" {
		return ""
	}
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}
	return strings.TrimRight(host, "/")
}

// resolvePerformanceTimeout: PERFORMANCE_TIMEOUT her zaman önceliklidir.
// Varsayılan yerelde 15 sn, Vercel'de 8 sn'dir; çünkü çağıran function'ın kendi
// süre sınırı içinde kalması gerekir (Hobby planında kısa olabilir).
func resolvePerformanceTimeout(getenv func(string) string) (time.Duration, error) {
	if value := strings.TrimSpace(getenv("PERFORMANCE_TIMEOUT")); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed <= 0 {
			return 0, errors.New("PERFORMANCE_TIMEOUT must be a positive duration")
		}
		return parsed, nil
	}
	if onVercel(getenv) {
		return 8 * time.Second, nil
	}
	return 15 * time.Second, nil
}

// resolvePerformanceBypassSecret, Vercel Deployment Protection'ın arkasındaki
// bir dağıtımda servis çağrısının 401 almaması için kullanılan "Protection
// Bypass for Automation" gizli anahtarıdır. Tanımsızsa boştur.
func resolvePerformanceBypassSecret(getenv func(string) string) string {
	if value := strings.TrimSpace(getenv("PERFORMANCE_BYPASS_SECRET")); value != "" {
		return value
	}
	return strings.TrimSpace(getenv("VERCEL_AUTOMATION_BYPASS_SECRET"))
}
