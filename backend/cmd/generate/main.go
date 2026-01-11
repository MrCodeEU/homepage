package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mrcodeeu/homepage/internal/config"
	"github.com/mrcodeeu/homepage/internal/models"
	"github.com/mrcodeeu/homepage/internal/scrapers"
	"github.com/mrcodeeu/homepage/internal/storage"
)

const (
	dataDir     = "./data/generated"
	appVersion  = "1.0.0"
)

var (
	outputDir = flag.String("output", dataDir, "Output directory for generated data files")
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

	// Initialize cache (temporary, for scraper compatibility)
	tempCacheDir := filepath.Join(os.TempDir(), "homepage-generate-cache")
	cache, err := storage.NewFileCache(tempCacheDir)
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempCacheDir); err != nil {
			log.Printf("Warning: failed to clean up temp cache: %v", err)
		}
	}()

	// Track which sources to generate
	generateAll := *sources == "all"
	shouldGenerate := map[string]bool{
		"github":   generateAll || *sources == "github",
		"strava":   generateAll || *sources == "strava",
		"linkedin": generateAll || *sources == "linkedin",
	}

	// Generate GitHub data
	if shouldGenerate["github"] {
		if err := generateGitHub(cfg, cache, *outputDir); err != nil {
			log.Printf("Error generating GitHub data: %v", err)
		} else if *verbose {
			log.Println("✓ GitHub data generated successfully")
		}
	}

	// Generate Strava data
	if shouldGenerate["strava"] {
		if err := generateStrava(cfg, cache, *outputDir); err != nil {
			log.Printf("Error generating Strava data: %v", err)
		} else if *verbose {
			log.Println("✓ Strava data generated successfully")
		}
	}

	// Generate LinkedIn data
	if shouldGenerate["linkedin"] {
		if err := generateLinkedIn(cfg, cache, *outputDir); err != nil {
			log.Printf("Error generating LinkedIn data: %v", err)
		} else if *verbose {
			log.Println("✓ LinkedIn data generated successfully")
		}
	}

	if *verbose {
		log.Println("Data generation completed!")
	}
}

func generateGitHub(cfg *config.Config, cache storage.Cache, outputDir string) error {
	if cfg.GitHubUsername == "" {
		return fmt.Errorf("GITHUB_USERNAME not set")
	}

	log.Println("Generating GitHub data...")

	scraper := scrapers.NewGitHubScraper(cfg.GitHubUsername, cfg.GitHubToken, cache)
	data, err := scraper.Scrape()
	if err != nil {
		return fmt.Errorf("failed to scrape: %w", err)
	}

	wrapped := models.GeneratedData{
		GeneratedAt: time.Now(),
		Source:      "github",
		Version:     appVersion,
		Data:        data,
	}

	return saveJSON(filepath.Join(outputDir, "github.json"), wrapped)
}

func generateStrava(cfg *config.Config, cache storage.Cache, outputDir string) error {
	if cfg.StravaClientID == "" || cfg.StravaClientSecret == "" || cfg.StravaRefreshToken == "" {
		return fmt.Errorf("Strava credentials not set (need STRAVA_CLIENT_ID, STRAVA_CLIENT_SECRET, STRAVA_REFRESH_TOKEN)")
	}

	log.Println("Generating Strava data...")

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

	wrapped := models.GeneratedData{
		GeneratedAt: time.Now(),
		Source:      "strava",
		Version:     appVersion,
		Data:        data,
	}

	return saveJSON(filepath.Join(outputDir, "strava.json"), wrapped)
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
		// For MVP, if scraping fails, create a placeholder
		log.Printf("Warning: LinkedIn scraping failed (%v), creating placeholder", err)
		data = createLinkedInPlaceholder()
	}

	wrapped := models.GeneratedData{
		GeneratedAt: time.Now(),
		Source:      "linkedin",
		Version:     appVersion,
		Data:        data,
	}

	return saveJSON(filepath.Join(outputDir, "linkedin.json"), wrapped)
}

func createLinkedInPlaceholder() *models.LinkedInData {
	return &models.LinkedInData{
		Profile: models.LinkedInProfile{
			Name:     "Your Name",
			Headline: "Software Engineer",
			Location: "Vienna, Austria",
			Summary:  "LinkedIn data requires manual configuration or authentication. See README for setup instructions.",
		},
		Experience: []models.LinkedInExperience{
			{
				Title:       "Software Engineer",
				Company:     "Tech Company",
				Location:    "Remote",
				StartDate:   "2020-01",
				EndDate:     "Present",
				Description: "Building awesome software",
			},
		},
		Education: []models.LinkedInEducation{
			{
				School:    "University",
				Degree:    "Bachelor of Science",
				Field:     "Computer Science",
				StartDate: "2014",
				EndDate:   "2018",
			},
		},
		Skills: []string{"Go", "TypeScript", "Docker", "Kubernetes"},
	}
}

func saveJSON(filename string, data interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	log.Printf("Saved: %s", filename)
	return nil
}
