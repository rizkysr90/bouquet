package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Database
	DatabaseURL string

	// Database connection pool (optional; defaults used if not set)
	DBMaxOpenConns    int           // Max open connections (default 25)
	DBMaxIdleConns    int           // Max idle connections (default 5)
	DBConnMaxLifetime time.Duration // Max lifetime of a connection (default 5m)
	DBConnMaxIdleTime time.Duration // Max time a connection can be idle (default 5m)

	// Cloudinary
	CloudName string
	APIKey    string
	APISecret string

	// Application
	Port      string
	Env       string
	JWTSecret string

	// WhatsApp
	WhatsAppNumber string

	// Store Information
	StoreName     string
	StoreAddress  string
	ShopeeLink    string
	TiktokLink    string
	InstagramLink string
}

// Load reads environment variables and returns a Config struct
func Load() *Config {
	// Load .env file (ignore error in production where env vars are set directly)
	_ = godotenv.Load()

	return &Config{
		DatabaseURL:       getEnv("DATABASE_URL", ""),
		DBMaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		DBConnMaxIdleTime: getEnvDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
		CloudName:         getEnv("CLOUDINARY_CLOUD_NAME", ""),
		APIKey:         getEnv("CLOUDINARY_API_KEY", ""),
		APISecret:      getEnv("CLOUDINARY_API_SECRET", ""),
		Port:           getEnv("PORT", "3000"),
		Env:            getEnv("ENV", "development"),
		JWTSecret:      getEnv("JWT_SECRET", "dev-secret"),
		WhatsAppNumber: getEnv("WHATSAPP_NUMBER", ""),
		StoreName:      getEnv("STORE_NAME", "Ancaka Florist Supplier"),
		StoreAddress:   getEnv("STORE_ADDRESS", ""),
		ShopeeLink:     getEnv("SHOPEE_LINK", ""),
		TiktokLink:     getEnv("TIKTOK_LINK", ""),
		InstagramLink:  getEnv("INSTAGRAM_LINK", ""),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if n, err := strconv.Atoi(value); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil && d > 0 {
			return d
		}
	}
	return fallback
}

// Validate checks if required env vars are set
func (c *Config) Validate() error {
	required := map[string]string{
		"DATABASE_URL":          c.DatabaseURL,
		"CLOUDINARY_CLOUD_NAME": c.CloudName,
		"CLOUDINARY_API_KEY":    c.APIKey,
		"CLOUDINARY_API_SECRET": c.APISecret,
		"JWT_SECRET":            c.JWTSecret,
	}

	for key, value := range required {
		if value == "" {
			return fmt.Errorf("required environment variable %s is not set", key)
		}
	}

	return nil
}
