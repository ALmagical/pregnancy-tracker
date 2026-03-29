package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr           string
	DatabaseURL        string
	JWTSecret          string
	JWTExpireHours     int
	WeChatAppID        string
	WeChatAppSecret    string
	WeChatMock         bool
	PublicBaseURL      string
	COSBucketURL       string
	COSSecretID        string
	COSSecretKey       COSKey
	COSRegion          string
	COSPathPrefix      string
	LocalUploadDir     string
	OpenAIAPIKey       string
	OpenAIModel        string
	ExportCooldown     time.Duration
	RateLimitPerMinute int
}

type COSKey string

func (c COSKey) String() string { return string(c) }

func Load() *Config {
	jwtH, _ := strconv.Atoi(env("JWT_EXPIRE_HOURS", "720"))
	coolMin, _ := strconv.Atoi(env("EXPORT_COOLDOWN_MINUTES", "60"))
	rpm, _ := strconv.Atoi(env("RATE_LIMIT_PER_MINUTE", "120"))
	return &Config{
		HTTPAddr:           env("HTTP_ADDR", ":8080"),
		DatabaseURL:        env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/pregnancy?sslmode=disable"),
		JWTSecret:          env("JWT_SECRET", "dev-change-me-use-long-random-string"),
		JWTExpireHours:     jwtH,
		WeChatAppID:        os.Getenv("WECHAT_APP_ID"),
		WeChatAppSecret:    os.Getenv("WECHAT_APP_SECRET"),
		WeChatMock:         env("WECHAT_MOCK", "0") == "1",
		PublicBaseURL:      strings.TrimRight(env("PUBLIC_BASE_URL", "http://localhost:8080"), "/"),
		COSBucketURL:       strings.TrimRight(os.Getenv("COS_BUCKET_URL"), "/"),
		COSSecretID:        os.Getenv("COS_SECRET_ID"),
		COSSecretKey:       COSKey(os.Getenv("COS_SECRET_KEY")),
		COSRegion:          env("COS_REGION", "ap-guangzhou"),
		COSPathPrefix:      strings.Trim(strings.TrimSpace(env("COS_PATH_PREFIX", "reports")), "/"),
		LocalUploadDir:     env("LOCAL_UPLOAD_DIR", "./data/uploads"),
		OpenAIAPIKey:       os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:        env("OPENAI_MODEL", "gpt-4o-mini"),
		ExportCooldown:     time.Duration(max(1, coolMin)) * time.Minute,
		RateLimitPerMinute: max(10, rpm),
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
