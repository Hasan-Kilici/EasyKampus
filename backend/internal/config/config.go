package config

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment              string
	Port                     string
	DatabasePath             string
	TursoDatabaseURL         string
	TursoAuthToken           string
	StatelessAuth            bool
	JWTSecret                string
	JWTExpiration            time.Duration
	CORSOrigins              []string
	RateLimit                int
	RateWindow               time.Duration
	CacheSize                int
	CacheTTL                 time.Duration
	SessionCacheSize         int
	SessionCacheTTL          time.Duration
	SuperAdminEmail          string
	SuperAdminPassword       string
	PerformanceServiceURL    string
	PerformanceTimeout       time.Duration
	PerformanceInternalToken string
	PerformanceBypassSecret  string
	PerformanceCacheSize     int
	PerformanceCacheTTL      time.Duration
	PerformanceRateLimit     int

	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2Bucket          string
	R2Endpoint        string
	R2PublicURL       string
}

func Load() (Config, error) {
	// Vercel'de ortam değişkenleri panelden gelir. Depodaki .env (APP_PORT=3000,
	// DATABASE_PATH=./lms.db, APP_ENV=development ...) paketle birlikte gitse bile
	// yüklenmemeli: Vercel sunucudan PORT'u dinlemesini bekler ve dosya sistemi salt-okunurdur.
	if os.Getenv("VERCEL") == "" {
		loadDotEnv()
	}

	port := os.Getenv("APP_PORT")
	if os.Getenv("VERCEL") != "" || port == "" {
		if value := os.Getenv("PORT"); value != "" {
			port = value
		}
	}
	if port == "" {
		port = "3000"
	}
	if _, err := strconv.Atoi(port); err != nil {
		return Config{}, errors.New("APP_PORT must be a number")
	}

	secret := os.Getenv("JWT_SECRET")
	if strings.TrimSpace(secret) == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}
	expiration := 24 * time.Hour
	if value := os.Getenv("JWT_EXPIRATION"); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed <= 0 {
			return Config{}, errors.New("JWT_EXPIRATION must be a positive duration")
		}
		expiration = parsed
	}
	rateLimit := 300
	if value := os.Getenv("RATE_LIMIT"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			return Config{}, errors.New("RATE_LIMIT must be a non-negative number (0 disables the global limit)")
		}
		rateLimit = parsed
	}
	cacheSize := 1000
	if value := os.Getenv("CACHE_SIZE"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return Config{}, errors.New("CACHE_SIZE must be a positive number")
		}
		cacheSize = parsed
	}
	cacheTTL := 5 * time.Minute
	if value := os.Getenv("CACHE_TTL"); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed <= 0 {
			return Config{}, errors.New("CACHE_TTL must be a positive duration")
		}
		cacheTTL = parsed
	}
	sessionCacheSize := 1000
	if value := os.Getenv("SESSION_CACHE_SIZE"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return Config{}, errors.New("SESSION_CACHE_SIZE must be a positive number")
		}
		sessionCacheSize = parsed
	}
	sessionCacheTTL := 30 * time.Second
	if value := os.Getenv("SESSION_CACHE_TTL"); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed <= 0 {
			return Config{}, errors.New("SESSION_CACHE_TTL must be a positive duration")
		}
		sessionCacheTTL = parsed
	}

	origins := strings.Split(os.Getenv("CORS_ORIGINS"), ",")
	if len(origins) == 1 && strings.TrimSpace(origins[0]) == "" {
		origins = []string{"http://localhost:5173"}
	}
	for index := range origins {
		origins[index] = strings.TrimSpace(origins[index])
	}

	databasePath := resolveDatabasePath(os.Getenv)

	performanceURL := resolvePerformanceURL(os.Getenv)
	performanceTimeout, err := resolvePerformanceTimeout(os.Getenv)
	if err != nil {
		return Config{}, err
	}
	performanceCacheSize := 500
	if value := os.Getenv("PERFORMANCE_CACHE_SIZE"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return Config{}, errors.New("PERFORMANCE_CACHE_SIZE must be a positive number")
		}
		performanceCacheSize = parsed
	}
	performanceCacheTTL := 5 * time.Minute
	if value := os.Getenv("PERFORMANCE_CACHE_TTL"); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed <= 0 {
			return Config{}, errors.New("PERFORMANCE_CACHE_TTL must be a positive duration")
		}
		performanceCacheTTL = parsed
	}
	performanceRateLimit := 30
	if value := os.Getenv("PERFORMANCE_RATE_LIMIT"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return Config{}, errors.New("PERFORMANCE_RATE_LIMIT must be a positive number")
		}
		performanceRateLimit = parsed
	}

	return Config{
		Environment:              os.Getenv("APP_ENV"),
		Port:                     port,
		DatabasePath:             databasePath,
		TursoDatabaseURL:         strings.TrimRight(strings.TrimSpace(os.Getenv("TURSO_DATABASE_URL")), "/"),
		TursoAuthToken:           strings.TrimSpace(os.Getenv("TURSO_AUTH_TOKEN")),
		StatelessAuth:            os.Getenv("VERCEL") != "",
		JWTSecret:                secret,
		JWTExpiration:            expiration,
		CORSOrigins:              origins,
		RateLimit:                rateLimit,
		RateWindow:               time.Minute,
		CacheSize:                cacheSize,
		CacheTTL:                 cacheTTL,
		SessionCacheSize:         sessionCacheSize,
		SessionCacheTTL:          sessionCacheTTL,
		SuperAdminEmail:          strings.TrimSpace(os.Getenv("SUPERADMIN_EMAIL")),
		SuperAdminPassword:       os.Getenv("SUPERADMIN_PASSWORD"),
		PerformanceServiceURL:    strings.TrimRight(performanceURL, "/"),
		PerformanceTimeout:       performanceTimeout,
		PerformanceInternalToken: strings.TrimSpace(os.Getenv("PERFORMANCE_INTERNAL_TOKEN")),
		PerformanceBypassSecret:  resolvePerformanceBypassSecret(os.Getenv),
		PerformanceCacheSize:     performanceCacheSize,
		PerformanceCacheTTL:      performanceCacheTTL,
		PerformanceRateLimit:     performanceRateLimit,
		R2AccountID:              strings.TrimSpace(os.Getenv("R2_ACCOUNT_ID")),
		R2AccessKeyID:            strings.TrimSpace(os.Getenv("R2_ACCESS_KEY_ID")),
		R2SecretAccessKey:        strings.TrimSpace(os.Getenv("R2_SECRET_ACCESS_KEY")),
		R2Bucket:                 strings.TrimSpace(os.Getenv("R2_BUCKET")),
		R2Endpoint:               strings.TrimRight(strings.TrimSpace(os.Getenv("R2_ENDPOINT")), "/"),
		R2PublicURL:              strings.TrimRight(strings.TrimSpace(os.Getenv("R2_PUBLIC_URL")), "/"),
	}, nil
}

func loadDotEnv() {
	directory, err := os.Getwd()
	if err != nil {
		return
	}
	for {
		path := filepath.Join(directory, ".env")
		if _, err := os.Stat(path); err == nil {
			_ = godotenv.Load(path)
			return
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return
		}
		directory = parent
	}
}

// resolveDatabasePath: Vercel'de yalnızca /tmp yazılabilir. DATABASE_PATH /tmp
// altında değilse (örn. ./lms.db) yok sayılır; aksi halde SQLite açılışta
// "readonly database" hatasıyla çöker ve her istek 500 döner.
func resolveDatabasePath(getenv func(string) string) string {
	value := strings.TrimSpace(getenv("DATABASE_PATH"))
	if getenv("VERCEL") == "" {
		if value == "" {
			return "./lms.db"
		}
		return value
	}
	if value == ":memory:" || strings.HasPrefix(filepath.ToSlash(value), "/tmp/") {
		return value
	}
	return "/tmp/lms.db"
}
