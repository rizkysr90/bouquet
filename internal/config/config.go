package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	// Database
	DatabaseURL string

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
	StoreName    string
	StoreAddress string
	ShopeeLink   string
}

// Load reads environment variables and returns a Config struct
func Load() *Config {
	// Load .env file (ignore error in production where env vars are set directly)
	_ = godotenv.Load()

	return &Config{
		DatabaseURL:    getEnv("DATABASE_URL", ""),
		CloudName:      getEnv("CLOUDINARY_CLOUD_NAME", ""),
		APIKey:         getEnv("CLOUDINARY_API_KEY", ""),
		APISecret:      getEnv("CLOUDINARY_API_SECRET", ""),
		Port:           getEnv("PORT", "3000"),
		Env:            getEnv("ENV", "development"),
		JWTSecret:      getEnv("JWT_SECRET", "dev-secret"),
		WhatsAppNumber: getEnv("WHATSAPP_NUMBER", ""),
		StoreName:      getEnv("STORE_NAME", "Aslam Flower Supply"),
		StoreAddress:   getEnv("STORE_ADDRESS", ""),
		ShopeeLink:     getEnv("SHOPEE_LINK", ""),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
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
