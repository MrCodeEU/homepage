package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mrcodeeu/homepage/internal/config"
	"github.com/mrcodeeu/homepage/internal/models"
	"github.com/mrcodeeu/homepage/internal/scrapers"
	"github.com/mrcodeeu/homepage/internal/storage"
)

const (
	dataDir    = "./data/generated"
	cacheDir   = "./.cache"
	appVersion = "1.0.0"
)

var (
	outputDir = flag.String("output", dataDir, "Output directory for generated data files")
	cachePath = flag.String("cache", cacheDir, "Cache directory for cookies and temporary data")
	sources   = flag.String("sources", "all", "Data sources to generate (all, github, strava, linkedin)")
	verbose   = flag.Bool("verbose", false, "Enable verbose logging")
)

func main() {
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if *verbose {
		log.Println("Starting data generation...")
		log.Printf("Output directory: %s", *outputDir)
		log.Printf("Sources: %s", *sources)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Load configuration
	cfg := config.Load()

	// Initialize persistent cache directory
	persistentCacheDir := *cachePath
	if err := os.MkdirAll(persistentCacheDir, 0755); err != nil {
		log.Fatalf("Failed to create cache directory: %v", err)
	}

	cache, err := storage.NewFileCache(persistentCacheDir)
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}

	if *verbose {
		log.Printf("Using cache directory: %s", persistentCacheDir)
	}

	// Track which sources to generate
	// Accept comma-separated values like "github,strava" in addition to "all"
	generateAll := *sources == "all"
	shouldGenerate := map[string]bool{
		"github":   generateAll,
		"strava":   generateAll,
		"linkedin": generateAll,
	}
	if !generateAll {
		for _, s := range strings.Split(*sources, ",") {
			s = strings.TrimSpace(s)
			if _, known := shouldGenerate[s]; known {
				shouldGenerate[s] = true
			} else if s != "" {
				log.Printf("Warning: unknown source %q (valid: github, strava, linkedin, all)", s)
			}
		}
	}

	// Track errors
	hasErrors := false

	// Generate GitHub data
	if shouldGenerate["github"] {
		if err := generateGitHub(cfg, cache, *outputDir); err != nil {
			log.Printf("Error generating GitHub data: %v", err)
			hasErrors = true
		} else if *verbose {
			log.Println("✓ GitHub data generated successfully")
		}
	}

	// Generate Strava data
	if shouldGenerate["strava"] {
		if err := generateStrava(cfg, cache, *outputDir); err != nil {
			log.Printf("Error generating Strava data: %v", err)
			hasErrors = true
		} else if *verbose {
			log.Println("✓ Strava data generated successfully")
		}
	}

	// Generate LinkedIn data
	if shouldGenerate["linkedin"] {
		if err := generateLinkedIn(cfg, cache, *outputDir); err != nil {
			log.Printf("Error generating LinkedIn data: %v", err)
			hasErrors = true
		} else if *verbose {
			log.Println("✓ LinkedIn data generated successfully")
		}
	}

	if *verbose {
		log.Println("Data generation completed!")
	}

	// Exit with error code if any generation failed
	if hasErrors {
		log.Println("Data generation completed with errors")
		os.Exit(1)
	}
}

func generateGitHub(cfg *config.Config, cache storage.Cache, outputDir string) error {
	log.Println("Generating GitHub data...")

	if cfg.GitHubUsername == "" {
		log.Println("ERROR: GITHUB_USERNAME environment variable is not set!")
		log.Println("Please set GITHUB_USERNAME to your GitHub username")
		return fmt.Errorf("GITHUB_USERNAME not set")
	}

	if cfg.GitHubToken == "" {
		log.Println("WARNING: GITHUB_TOKEN is not set - API rate limits will be very restrictive")
		log.Println("It's highly recommended to set a GitHub personal access token")
	}

	log.Printf("GitHub username: %s", cfg.GitHubUsername)
	log.Printf("GitHub token present: %v", cfg.GitHubToken != "")

	scraper := scrapers.NewGitHubScraper(cfg.GitHubUsername, cfg.GitHubToken, cache)
	data, err := scraper.Scrape()
	if err != nil {
		return fmt.Errorf("failed to scrape: %w", err)
	}

	if err := validateGitHubData(data); err != nil {
		return fmt.Errorf("GitHub data validation failed: %w", err)
	}

	return saveJSON(filepath.Join(outputDir, "github.json"), "github", data)
}

func generateStrava(cfg *config.Config, cache storage.Cache, outputDir string) error {
	log.Println("Generating Strava data...")

	if cfg.StravaClientID == "" || cfg.StravaClientSecret == "" || cfg.StravaRefreshToken == "" {
		return fmt.Errorf("strava credentials incomplete (need STRAVA_CLIENT_ID, STRAVA_CLIENT_SECRET, STRAVA_REFRESH_TOKEN)")
	}

	log.Println("Strava credentials verified")

	scraper := scrapers.NewStravaScraper(
		cfg.StravaClientID,
		cfg.StravaClientSecret,
		cfg.StravaRefreshToken,
		cache,
	)
	data, err := scraper.Scrape()
	if err != nil {
		return fmt.Errorf("failed to scrape: %w", err)
	}

	if err := validateStravaData(data); err != nil {
		return fmt.Errorf("strava data validation failed: %w", err)
	}

	return saveJSON(filepath.Join(outputDir, "strava.json"), "strava", data)
}

func generateLinkedIn(cfg *config.Config, cache storage.Cache, outputDir string) error {
	if cfg.LinkedInEmail == "" || cfg.LinkedInPassword == "" {
		return fmt.Errorf("LinkedIn credentials not set (need LINKEDIN_EMAIL and LINKEDIN_PASSWORD)")
	}

	log.Println("Generating LinkedIn data...")

	scraper := scrapers.NewLinkedInScraper(
		cfg.LinkedInEmail,
		cfg.LinkedInPassword,
		cfg.LinkedInTOTPSecret,
		cfg.LinkedInProfileURL,
		cache,
	)
	data, err := scraper.Scrape()
	if err != nil {
		return fmt.Errorf("failed to scrape: %w", err)
	}

	if err := validateLinkedInData(data); err != nil {
		return fmt.Errorf("LinkedIn data validation failed: %w", err)
	}

	return saveJSON(filepath.Join(outputDir, "linkedin.json"), "linkedin", data)
}

func validateGitHubData(data any) error {
	projects, ok := data.([]scrapers.Project)
	if !ok {
		return fmt.Errorf("unexpected data type: %T", data)
	}
	if len(projects) == 0 {
		return fmt.Errorf("no portfolio projects found")
	}
	return nil
}

func validateStravaData(data any) error {
	// Handle both value and pointer types
	var stravaData models.StravaData
	switch v := data.(type) {
	case models.StravaData:
		stravaData = v
	case *models.StravaData:
		if v == nil {
			return fmt.Errorf("strava data pointer is nil")
		}
		stravaData = *v
	default:
		return fmt.Errorf("unexpected data type: %T", data)
	}
	if stravaData.TotalStats.Count == 0 {
		return fmt.Errorf("no activities found")
	}
	return nil
}

func validateLinkedInData(data any) error {
	linkedInData, ok := data.(*models.LinkedInData)
	if !ok {
		return fmt.Errorf("unexpected data type: %T", data)
	}
	if linkedInData == nil {
		return fmt.Errorf("LinkedIn data is nil")
	}
	if linkedInData.Profile.Name == "" {
		return fmt.Errorf("profile name is empty")
	}
	if len(linkedInData.Experience) == 0 {
		return fmt.Errorf("no experience data found")
	}
	// Education and skills are treated as warnings — LinkedIn's DOM structure
	// occasionally changes, causing extraction to fail even when data is present.
	// Missing these fields should not block saving valid profile/experience data.
	if len(linkedInData.Education) == 0 {
		log.Println("Warning: no education data extracted (LinkedIn DOM may have changed)")
	}
	if len(linkedInData.Skills) == 0 {
		log.Println("Warning: no skills data extracted")
	}
	return nil
}

func saveJSON(filename, source string, innerData interface{}) error {
	newBytes, err := json.Marshal(innerData)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Skip writing if the data content hasn't changed (ignores generated_at).
	// This prevents spurious git diffs that would trigger unnecessary Docker builds.
	if existing, err := os.ReadFile(filename); err == nil {
		var wrapper struct {
			Data json.RawMessage `json:"data"`
		}
		if json.Unmarshal(existing, &wrapper) == nil {
			var compactExisting, compactNew bytes.Buffer
			if json.Compact(&compactExisting, wrapper.Data) == nil &&
				json.Compact(&compactNew, newBytes) == nil &&
				bytes.Equal(compactExisting.Bytes(), compactNew.Bytes()) {
				log.Printf("Skipped %s (no data changes)", filename)
				return nil
			}
		}
	}

	wrapped := models.GeneratedData{
		GeneratedAt: time.Now(),
		Source:      source,
		Version:     appVersion,
		Data:        innerData,
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Warning: failed to close file %s: %v", filename, closeErr)
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(wrapped); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	log.Printf("Saved: %s", filename)
	return nil
}
