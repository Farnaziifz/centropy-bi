// Package config loads process configuration from the environment. It is
// deliberately dependency-free (no viper/envconfig reflection walk) — a
// handful of explicit os.Getenv calls at startup, read once.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

type Config struct {
	Env      string // "development" | "production"
	HTTP     HTTPConfig
	DB       DBConfig
	AlefGym  AlefGymConfig
	Redis    RedisConfig
	Auth     AuthConfig
	GapGPT   GapGPTConfig
	Analysis AnalysisConfig
}

type HTTPConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	AllowedOrigins  []string
}

type DBConfig struct {
	DSN         string
	MaxOpenConn int
	MaxIdleConn int
	ConnMaxLife time.Duration
}

// AlefGymConfig is the read-only link back to the AlefGym production
// database — the source of truth for customers, orders and course expiry
// that segmentation is computed from. This service never writes to it.
type AlefGymConfig struct {
	DSN string
	// ExcludedUserIDs filters out known test/internal accounts from every
	// segment and sync query — see loyalty-club-roadmap.html phase 0
	// ("حذف حساب‌های تست از گزارش‌ها").
	ExcludedUserIDs []uuid.UUID
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type AuthConfig struct {
	JWTSecret string
	AccessTTL time.Duration
}

// GapGPTConfig is the OpenAI-compatible proxy used to classify why an
// overdue customer stopped buying, from their own chat messages — same
// provider findra/backend already uses for AI research, reused here with
// its own key so cost/usage stays attributable per service.
type GapGPTConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

// AnalysisConfig controls the daily AI-analysis job (see
// internal/application/analysis/command).
type AnalysisConfig struct {
	// RunInterval between automatic daily runs. Defaults to 24h; only
	// worth overriding in dev to test the scheduler without waiting a day.
	RunInterval time.Duration
	// RunOnStartup triggers one run immediately at boot instead of only
	// waiting for the first interval tick — off by default so a plain
	// `go run` during development never silently spends GapGPT credits.
	RunOnStartup bool
}

func Load() (*Config, error) {
	// Best-effort: only present in local dev, absent (fine) in
	// docker/production where real env vars are injected instead.
	_ = godotenv.Load()

	cfg := &Config{
		Env: getEnv("APP_ENV", "development"),
		HTTP: HTTPConfig{
			Port:        getEnv("HTTP_PORT", "8090"),
			ReadTimeout: getDuration("HTTP_READ_TIMEOUT", 5*time.Second),
			// 5m, not the usual ~10s: the LLM-batch trigger routes
			// (analysis/run, complaints/.../verify) can legitimately take
			// minutes (several sequential GapGPT calls) — net/http's
			// WriteTimeout caps the whole request regardless of any
			// per-route chi middleware, so it has to be raised globally.
			// Every other route finishes in well under a second either way.
			WriteTimeout:    getDuration("HTTP_WRITE_TIMEOUT", 5*time.Minute),
			IdleTimeout:     getDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getDuration("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
			AllowedOrigins:  getEnvList("CORS_ALLOWED_ORIGINS", "http://localhost:5173"),
		},
		DB: DBConfig{
			DSN:         getEnv("DATABASE_URL", "postgres://centropy_affilate:centropy_affilate@localhost:5433/centropy_affilate?sslmode=disable"),
			MaxOpenConn: getInt("DB_MAX_OPEN_CONN", 10),
			MaxIdleConn: getInt("DB_MAX_IDLE_CONN", 10),
			ConnMaxLife: getDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		AlefGym: AlefGymConfig{
			DSN:             getEnv("ALEFGYM_DATABASE_DSN", ""),
			ExcludedUserIDs: getUUIDList("ALEFGYM_EXCLUDED_USER_IDS"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getInt("REDIS_DB", 0),
		},
		Auth: AuthConfig{
			JWTSecret: getEnv("JWT_SECRET", ""),
			AccessTTL: getDuration("JWT_ACCESS_TTL", 8*time.Hour),
		},
		GapGPT: GapGPTConfig{
			APIKey:  getEnv("GAPGPT_API_KEY", ""),
			BaseURL: getEnv("GAPGPT_BASE_URL", "https://api.gapgpt.app/v1"),
			Model:   getEnv("GAPGPT_MODEL", "gpt-5.4"),
		},
		Analysis: AnalysisConfig{
			RunInterval:  getDuration("ANALYSIS_RUN_INTERVAL", 24*time.Hour),
			RunOnStartup: getBool("ANALYSIS_RUN_ON_STARTUP", false),
		},
	}

	if cfg.Env == "production" {
		if cfg.Auth.JWTSecret == "" {
			return nil, fmt.Errorf("config: JWT_SECRET is required in production")
		}
		if cfg.AlefGym.DSN == "" {
			return nil, fmt.Errorf("config: ALEFGYM_DATABASE_DSN is required")
		}
	}
	if cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = "dev-insecure-jwt-secret"
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

// getEnvList reads a comma-separated env var (e.g. multiple local dev
// front-ends on different ports, each needing its own CORS origin).
func getEnvList(key, fallback string) []string {
	raw := getEnv(key, fallback)
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// getUUIDList reads a comma-separated list of UUIDs, silently skipping
// anything that doesn't parse — a malformed entry here should never stop
// the process from booting.
func getUUIDList(key string) []uuid.UUID {
	raw := getEnv(key, "")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if id, err := uuid.Parse(p); err == nil {
			out = append(out, id)
		}
	}
	return out
}
