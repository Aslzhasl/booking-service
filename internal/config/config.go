package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	DBSSLMode         string
	UserServiceURL    string
	ListingServiceURL string
	JWTSecret         string
	HTTPPort          string
}

func LoadConfig() *Config {
	// Попытка загрузить .env (не обязательно, но удобно для локальной разработки)
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env not found, using environment variables")
	}

	return &Config{
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBUser:            getEnv("DB_USER", "postgres"),
		DBPassword:        getEnv("DB_PASSWORD", ""),
		DBName:            getEnv("DB_NAME", "bookingdb"),
		DBSSLMode:         getEnv("DB_SSLMODE", "disable"),
		UserServiceURL:    getEnv("USER_SERVICE_URL", ""),
		ListingServiceURL: getEnv("LISTING_SERVICE_URL", ""),
		JWTSecret:         getEnv("JWT_SECRET", ""),
		HTTPPort:          getEnv("HTTP_PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
