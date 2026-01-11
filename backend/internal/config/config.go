package config

import (
	"os"
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
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
