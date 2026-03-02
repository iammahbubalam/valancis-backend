package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	Env                string
	LogLevel           string
	DBUrl              string
	GoogleClientID     string
	JWTSecret          string
	AllowedOrigin      string
	FrontendURL        string // Explicit Frontend URL for emails, sitemaps, etc.
	GoogleTokenInfoURL string
	GoogleClientSecret string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	// DB Config
	DBMaxConns        int32
	DBMinConns        int32
	DBMaxConnIdleTime time.Duration
	// R2 Storage
	R2AccountID       string
	R2AccessKeyID     string
	R2AccessKeySecret string
	R2BucketName      string
	R2PublicURL       string
	// Cache (L9 Standard)
	CacheCategoryTTL time.Duration
	CacheProductTTL  time.Duration
	CacheSitemapTTL  time.Duration
	// Upload Configuration
	MaxUploadSizeMB int64
	R2UploadTimeout time.Duration
	// Business Rules
	MaxCartQuantity int

	// Marketing / Analytics (L9)
	FacebookPixelID     string
	FacebookAccessToken string
	FacebookAPIVersion  string
}

func LoadConfig() *Config {
	// 1. Check if a specific config file is requested via env var
	configFile := os.Getenv("CONFIG_FILE")
	if configFile != "" {
		if err := godotenv.Load(configFile); err != nil {
			log.Printf("Warning: Failed to load config file '%s': %v", configFile, err)
		} else {
			log.Printf("Loaded configuration from %s", configFile)
		}
	} else {
		// 2. Default fallback: Try loading .env (standard local dev)
		// We don't error here because in pure docker/prod envs, .env might not exist,
		// and we rely on system env vars.
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found or error loading it, relying on system env vars")
		}
	}

	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		Env:                getEnv("ENV", "development"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		DBUrl:              getEnv("DB_DSN", ""),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		JWTSecret:          getEnv("JWT_SECRET", "default_secret_CHANGE_ME"),
		AllowedOrigin:      getEnv("ALLOWED_ORIGIN", "http://localhost:3000"),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:3000"),
		GoogleTokenInfoURL: getEnv("GOOGLE_TOKEN_INFO_URL", "https://www.googleapis.com/oauth2/v3/tokeninfo?access_token=%s"),
		AccessTokenExpiry:  getDurationEnv("ACCESS_TOKEN_EXPIRY", time.Hour*24),    // Default 24h
		RefreshTokenExpiry: getDurationEnv("REFRESH_TOKEN_EXPIRY", time.Hour*24*7), // Default 7d

		DBMaxConns:        getInt32Env("DB_MAX_CONNS", 50),
		DBMinConns:        getInt32Env("DB_MIN_CONNS", 10),
		DBMaxConnIdleTime: getDurationEnv("DB_MAX_CONN_IDLE_TIME", time.Minute*15),

		// R2 Storage
		R2AccountID:       getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
		R2AccessKeySecret: getEnv("R2_ACCESS_KEY_SECRET", ""),
		R2BucketName:      getEnv("R2_BUCKET_NAME", ""),
		R2PublicURL:       getEnv("R2_PUBLIC_URL", ""),

		// Cache defaults: 30m Category, 10m Product, 6h Sitemap
		CacheCategoryTTL: getDurationEnv("CACHE_CATEGORY_TTL", 30*time.Minute),
		CacheProductTTL:  getDurationEnv("CACHE_PRODUCT_TTL", 10*time.Minute),
		CacheSitemapTTL:  getDurationEnv("CACHE_SITEMAP_TTL", 6*time.Hour),

		// Upload defaults: 10MB max, 30s timeout
		MaxUploadSizeMB: getInt64Env("MAX_UPLOAD_SIZE_MB", 10),
		R2UploadTimeout: getDurationEnv("R2_UPLOAD_TIMEOUT", 30*time.Second),

		// Business rules: 1000 max cart quantity
		MaxCartQuantity: getIntEnv("MAX_CART_QUANTITY", 1000),

		// Marketing
		FacebookPixelID:     getEnv("FACEBOOK_PIXEL_ID", ""),
		FacebookAccessToken: getEnv("FACEBOOK_ACCESS_TOKEN", ""),
		FacebookAPIVersion:  getEnv("FACEBOOK_API_VERSION", "v19.0"),
	}

	cfg.Validate()
	return cfg
}

func (c *Config) Validate() {
	if c.DBUrl == "" {
		log.Fatal("CRITICAL: DB_DSN environment variable is required")
	}
	if c.JWTSecret == "default_secret_CHANGE_ME" {
		log.Println("WARNING: Using default JWT secret. Setting up for failure in production.")
	}
	if c.GoogleClientID == "" {
		log.Fatal("CRITICAL: GOOGLE_CLIENT_ID is required")
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
		log.Printf("Invalid duration for %s, using fallback", key)
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
		log.Printf("Invalid int for %s, using fallback", key)
	}
	return fallback
}

func getInt64Env(key string, fallback int64) int64 {
	if value, exists := os.LookupEnv(key); exists {
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
		}
		log.Printf("Invalid int64 for %s, using fallback", key)
	}
	return fallback
}
