package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mrcodeeu/homepage/internal/models"
)

const (
	generatedDataDir = "./data/generated"
)

// DataLoader loads pre-generated data files
type DataLoader struct {
	dataDir string
}

// NewDataLoader creates a new data loader
func NewDataLoader(dataDir string) *DataLoader {
	if dataDir == "" {
		dataDir = generatedDataDir
	}
	return &DataLoader{
		dataDir: dataDir,
	}
}

// LoadGitHub loads GitHub projects data
func (d *DataLoader) LoadGitHub() (interface{}, error) {
	var wrapped models.GeneratedData
	if err := d.loadJSON("github.json", &wrapped); err != nil {
		return nil, err
	}
	return wrapped.Data, nil
}

// LoadStrava loads Strava data
func (d *DataLoader) LoadStrava() (*models.StravaData, error) {
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

// loadJSON loads and parses a JSON file
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
	filePath := filepath.Join(d.dataDir, fmt.Sprintf("%s.json", source))
	_, err := os.Stat(filePath)
	return err == nil
}
