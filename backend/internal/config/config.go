package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port string

	// GitHub
	GitHubToken    string
	GitHubUsername string

	// Strava
	StravaClientID     string
	StravaClientSecret string
	StravaRefreshToken string

	// LinkedIn
	LinkedInEmail      string
	LinkedInPassword   string
	LinkedInTOTPSecret string
	LinkedInProfileURL string

	// Cache settings
	CacheDir      string
	CacheTTLHours int

	// Data refresh settings
	DataRefreshInterval time.Duration
}

func Load() *Config {
	return &Config{
		Port: getEnv("PORT", "8080"),

		GitHubToken:    os.Getenv("GITHUB_TOKEN"),
		GitHubUsername: getEnv("GITHUB_USERNAME", "mrcodeeu"),

		StravaClientID:     os.Getenv("STRAVA_CLIENT_ID"),
		StravaClientSecret: os.Getenv("STRAVA_CLIENT_SECRET"),
		StravaRefreshToken: os.Getenv("STRAVA_REFRESH_TOKEN"),

		LinkedInEmail:      os.Getenv("LINKEDIN_EMAIL"),
		LinkedInPassword:   os.Getenv("LINKEDIN_PASSWORD"),
		LinkedInTOTPSecret: os.Getenv("LINKEDIN_TOTP_SECRET"),
		LinkedInProfileURL: getEnv("LINKEDIN_PROFILE_URL", "https://linkedin.com/in/mrcodeeu"),

		CacheDir:      getEnv("CACHE_DIR", "./data/cache"),
		CacheTTLHours: 24,

		DataRefreshInterval: getEnvDuration("DATA_REFRESH_HOURS", 4) * time.Hour,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvDuration(key string, defaultHours int) time.Duration {
	if value := os.Getenv(key); value != "" {
		if hours, err := strconv.Atoi(value); err == nil && hours > 0 {
			return time.Duration(hours)
		}
	}
	return time.Duration(defaultHours)
}
