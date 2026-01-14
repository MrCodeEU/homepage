package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mrcodeeu/homepage/internal/models"
)

const (
	generatedDataDir       = "./data/generated"
	defaultRefreshInterval = 4 * time.Hour
	githubRawBaseURL       = "https://raw.githubusercontent.com/MrCodeEU/homepage/main/data/generated"
)

// DataLoader loads pre-generated data files and supports auto-refresh from GitHub
type DataLoader struct {
	dataDir         string
	refreshInterval time.Duration
	httpClient      *http.Client
	mu              sync.RWMutex // Protects file access during refresh
}

// NewDataLoader creates a new data loader
func NewDataLoader(dataDir string) *DataLoader {
	if dataDir == "" {
		dataDir = generatedDataDir
	}
	return &DataLoader{
		dataDir:         dataDir,
		refreshInterval: defaultRefreshInterval,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetRefreshInterval sets a custom refresh interval
func (d *DataLoader) SetRefreshInterval(interval time.Duration) {
	d.refreshInterval = interval
}

// StartAutoRefresh starts a background goroutine that periodically fetches
// fresh data from the GitHub repository. This keeps the data up-to-date
// without requiring container restarts or external deployment triggers.
func (d *DataLoader) StartAutoRefresh(ctx context.Context) {
	log.Printf("Starting auto-refresh with interval: %v", d.refreshInterval)

	go func() {
		// Ensure data directory exists
		if err := os.MkdirAll(d.dataDir, 0755); err != nil {
			log.Printf("Warning: failed to create data directory: %v", err)
		}

		// Initial fetch on startup
		log.Println("Performing initial data fetch from GitHub...")
		d.refreshFromGitHub()

		ticker := time.NewTicker(d.refreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				log.Println("Scheduled data refresh from GitHub...")
				d.refreshFromGitHub()
			case <-ctx.Done():
				log.Println("Auto-refresh stopped")
				return
			}
		}
	}()
}

// refreshFromGitHub fetches the latest data files from the GitHub repository
func (d *DataLoader) refreshFromGitHub() {
	files := []string{"github.json", "linkedin.json", "strava.json"}
	successCount := 0

	for _, file := range files {
		if err := d.fetchAndSaveFile(file); err != nil {
			log.Printf("⚠ Failed to refresh %s: %v", file, err)
		} else {
			successCount++
		}
	}

	log.Printf("Data refresh complete: %d/%d files updated", successCount, len(files))
}

// fetchAndSaveFile downloads a single file from GitHub and saves it locally
func (d *DataLoader) fetchAndSaveFile(filename string) error {
	url := fmt.Sprintf("%s/%s", githubRawBaseURL, filename)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers to avoid caching issues
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Accept", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Validate JSON before saving
	var js json.RawMessage
	if err := json.Unmarshal(data, &js); err != nil {
		return fmt.Errorf("invalid JSON received: %w", err)
	}

	// Write to data directory with lock
	d.mu.Lock()
	defer d.mu.Unlock()

	filePath := filepath.Join(d.dataDir, filename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("✓ Updated %s (%d bytes)", filename, len(data))
	return nil
}

// LoadGitHub loads GitHub projects data
func (d *DataLoader) LoadGitHub() (interface{}, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var wrapped models.GeneratedData
	if err := d.loadJSON("github.json", &wrapped); err != nil {
		return nil, err
	}
	return wrapped.Data, nil
}

// LoadStrava loads Strava data
func (d *DataLoader) LoadStrava() (*models.StravaData, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var wrapped models.GeneratedData
	if err := d.loadJSON("strava.json", &wrapped); err != nil {
		return nil, err
	}

	// Convert map to StravaData struct
	jsonData, err := json.Marshal(wrapped.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal strava data: %w", err)
	}

	var stravaData models.StravaData
	if err := json.Unmarshal(jsonData, &stravaData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal strava data: %w", err)
	}

	return &stravaData, nil
}

// LoadLinkedIn loads LinkedIn data
func (d *DataLoader) LoadLinkedIn() (*models.LinkedInData, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var wrapped models.GeneratedData
	if err := d.loadJSON("linkedin.json", &wrapped); err != nil {
		return nil, err
	}

	// Convert map to LinkedInData struct
	jsonData, err := json.Marshal(wrapped.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal linkedin data: %w", err)
	}

	var linkedInData models.LinkedInData
	if err := json.Unmarshal(jsonData, &linkedInData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal linkedin data: %w", err)
	}

	return &linkedInData, nil
}

// loadJSON loads and parses a JSON file (caller must hold lock)
func (d *DataLoader) loadJSON(filename string, v interface{}) error {
	filePath := filepath.Join(d.dataDir, filename)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("data file not found: %s (run data generation first)", filename)
		}
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to parse JSON from %s: %w", filename, err)
	}

	return nil
}

// DataExists checks if a data file exists
func (d *DataLoader) DataExists(source string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	filePath := filepath.Join(d.dataDir, fmt.Sprintf("%s.json", source))
	_, err := os.Stat(filePath)
	return err == nil
}
