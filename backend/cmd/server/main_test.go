package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mrcodeeu/homepage/internal/models"
	"github.com/mrcodeeu/homepage/internal/storage"
)

// setupTestData creates temporary test data files and initializes the dataLoader
func setupTestData(t *testing.T) (cleanup func()) {
	t.Helper()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "homepage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create test GitHub data
	githubData := models.GeneratedData{
		GeneratedAt: time.Now(),
		Source:      "github",
		Version:     "1.0",
		Data: []map[string]interface{}{
			{
				"name":        "test-project",
				"description": "A test project",
				"url":         "https://github.com/testuser/test-project",
				"stars":       10,
				"language":    "Go",
				"topics":      []string{"test"},
				"images":      []string{},
				"featured":    false,
			},
		},
	}

	// Create test LinkedIn data
	linkedInData := models.GeneratedData{
		GeneratedAt: time.Now(),
		Source:      "linkedin",
		Version:     "1.0",
		Data: models.LinkedInData{
			Profile: models.LinkedInProfile{
				Name:     "Test User",
				Headline: "Software Engineer",
				Location: "Test City",
				Summary:  "A test summary",
			},
			Experience: []models.LinkedInExperience{
				{
					Title:       "Engineer",
					Company:     "Test Corp",
					Location:    "Remote",
					StartDate:   "2020-01",
					EndDate:     "Present",
					Description: "Testing things",
				},
			},
			Education: []models.LinkedInEducation{
				{
					School:    "Test University",
					Degree:    "B.S.",
					Field:     "Computer Science",
					StartDate: "2016",
					EndDate:   "2020",
				},
			},
			Skills: []string{"Go", "TypeScript"},
		},
	}

	// Create test Strava data
	stravaData := models.GeneratedData{
		GeneratedAt: time.Now(),
		Source:      "strava",
		Version:     "1.0",
		Data: models.StravaData{
			TotalStats: models.StravaStats{
				Count:         100,
				Distance:      500000,
				MovingTime:    180000,
				ElapsedTime:   200000,
				ElevationGain: 5000,
			},
			YearToDateStats: models.StravaStats{
				Count:         50,
				Distance:      250000,
				MovingTime:    90000,
				ElapsedTime:   100000,
				ElevationGain: 2500,
			},
			RecentActivities: []models.StravaActivity{
				{
					ID:                 1,
					Name:               "Morning Run",
					Distance:           5000,
					MovingTime:         1800,
					ElapsedTime:        2000,
					TotalElevationGain: 50,
					Type:               "Run",
					StartDate:          time.Now(),
					AveragePace:        6.0,
					AverageSpeed:       2.78,
					MaxSpeed:           3.5,
				},
			},
			BestActivities:  models.StravaBestRecords{},
			PersonalRecords: []models.StravaRecord{},
		},
	}

	// Write test files
	writeTestJSON(t, tempDir, "github.json", githubData)
	writeTestJSON(t, tempDir, "linkedin.json", linkedInData)
	writeTestJSON(t, tempDir, "strava.json", stravaData)

	// Initialize the global dataLoader with test directory
	dataLoader = storage.NewDataLoader(tempDir)

	return func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: failed to clean up temp dir: %v", err)
		}
	}
}

func writeTestJSON(t *testing.T, dir, filename string, data interface{}) {
	t.Helper()

	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal test data for %s: %v", filename, err)
	}

	if err := os.WriteFile(filepath.Join(dir, filename), jsonData, 0644); err != nil {
		t.Fatalf("Failed to write test file %s: %v", filename, err)
	}
}

func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}

	if _, ok := response["time"]; !ok {
		t.Error("Expected 'time' field in response")
	}
}

func TestHandleCV(t *testing.T) {
	cleanup := setupTestData(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/cv", nil)
	w := httptest.NewRecorder()

	handleCV(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check required fields for LinkedInData format
	requiredFields := []string{"profile", "experience", "education", "skills"}
	for _, field := range requiredFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Expected field '%s' in CV response", field)
		}
	}

	// Check profile sub-fields
	if profile, ok := response["profile"].(map[string]interface{}); ok {
		profileFields := []string{"name", "headline", "location", "summary"}
		for _, field := range profileFields {
			if _, ok := profile[field]; !ok {
				t.Errorf("Expected field '%s' in profile", field)
			}
		}
	} else {
		t.Error("Expected 'profile' to be an object")
	}
}

func TestHandleProjects(t *testing.T) {
	cleanup := setupTestData(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	w := httptest.NewRecorder()

	handleProjects(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response) == 0 {
		t.Error("Expected at least one project in response")
	}

	// Check first project has required fields
	if len(response) > 0 {
		project := response[0]
		requiredFields := []string{"name", "description", "url", "stars", "language"}
		for _, field := range requiredFields {
			if _, ok := project[field]; !ok {
				t.Errorf("Expected field '%s' in project", field)
			}
		}
	}
}

func TestHandleStrava(t *testing.T) {
	cleanup := setupTestData(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/strava", nil)
	w := httptest.NewRecorder()

	handleStrava(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check required fields for StravaData format
	requiredFields := []string{"total_stats", "year_to_date_stats", "recent_activities", "best_activities", "personal_records"}
	for _, field := range requiredFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Expected field '%s' in Strava response", field)
		}
	}

	// Check total_stats sub-fields
	if stats, ok := response["total_stats"].(map[string]interface{}); ok {
		statsFields := []string{"count", "distance", "moving_time", "elapsed_time", "elevation_gain"}
		for _, field := range statsFields {
			if _, ok := stats[field]; !ok {
				t.Errorf("Expected field '%s' in total_stats", field)
			}
		}
	} else {
		t.Error("Expected 'total_stats' to be an object")
	}
}

func TestCORSMiddleware(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test with localhost origin (should be allowed)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Errorf("Expected CORS header to be 'http://localhost:5173', got '%s'", w.Header().Get("Access-Control-Allow-Origin"))
	}

	// Test without origin (should not set CORS header)
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	w2 := httptest.NewRecorder()

	handler.ServeHTTP(w2, req2)

	if w2.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Expected no CORS header for same-origin request, got '%s'", w2.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSPreflight(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for OPTIONS request, got %d", w.Code)
	}
}
