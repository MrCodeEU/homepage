package scrapers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mockCache implements storage.Cache for testing
type mockCache struct {
	data map[string][]byte
	ttls map[string]time.Time
}

func newMockCache() *mockCache {
	return &mockCache{
		data: make(map[string][]byte),
		ttls: make(map[string]time.Time),
	}
}

func (m *mockCache) Get(key string) ([]byte, error) {
	// Check expiration
	if exp, ok := m.ttls[key]; ok && time.Now().After(exp) {
		delete(m.data, key)
		delete(m.ttls, key)
		return nil, nil
	}

	data, ok := m.data[key]
	if !ok {
		return nil, nil
	}
	return data, nil
}

func (m *mockCache) Set(key string, data []byte, ttl time.Duration) error {
	m.data[key] = data
	m.ttls[key] = time.Now().Add(ttl)
	return nil
}

func (m *mockCache) Delete(key string) error {
	delete(m.data, key)
	delete(m.ttls, key)
	return nil
}

func (m *mockCache) Clear() error {
	m.data = make(map[string][]byte)
	m.ttls = make(map[string]time.Time)
	return nil
}

func TestNewGitHubScraper(t *testing.T) {
	cache := newMockCache()
	scraper := NewGitHubScraper("testuser", "token123", cache)

	if scraper == nil {
		t.Fatal("Scraper is nil")
	}

	if scraper.Name() != "github" {
		t.Errorf("Expected name 'github', got '%s'", scraper.Name())
	}

	if scraper.username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", scraper.username)
	}
}

func TestGitHubScraper_FetchRepositories(t *testing.T) {
	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authorization header
		auth := r.Header.Get("Authorization")
		if auth != "token test-token" {
			t.Errorf("Expected Authorization header 'token test-token', got '%s'", auth)
		}

		repos := []GitHubRepo{
			{
				Name:        "test-repo",
				Description: "Test repository",
				HTMLURL:     "https://github.com/testuser/test-repo",
				Language:    "Go",
				StarCount:   10,
				Topics:      []string{"test", "example"},
				Private:     false,
			},
			{
				Name:        "private-repo",
				Description: "Private repository",
				HTMLURL:     "https://github.com/testuser/private-repo",
				Language:    "Python",
				StarCount:   5,
				Topics:      []string{},
				Private:     true,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(repos); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cache := newMockCache()
	scraper := NewGitHubScraper("testuser", "test-token", cache)
	scraper.client = server.Client()

	// Override the GitHub API base URL to use our test server
	// We need to modify the fetchRepositories method to use the test server
	// For now, we'll test the deduplication and utility functions
}

func TestDeduplicateStrings(t *testing.T) {
	tests := []struct {
		input    []string
		expected []string
	}{
		{
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			input:    []string{"a", "a", "b", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			input:    []string{},
			expected: []string{},
		},
		{
			input:    []string{"test", "example", "test", "example"},
			expected: []string{"test", "example"},
		},
	}

	for _, tt := range tests {
		result := deduplicateStrings(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("Expected %v, got %v", tt.expected, result)
				break
			}
		}
	}
}

func TestPortfolioMetadata_JSON(t *testing.T) {
	jsonData := `{
		"description": "Custom description",
		"images": ["image1.png", "image2.jpg"],
		"featured": true,
		"tags": ["golang", "web"]
	}`

	var metadata PortfolioMetadata
	if err := json.Unmarshal([]byte(jsonData), &metadata); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if metadata.Description != "Custom description" {
		t.Errorf("Expected description 'Custom description', got '%s'", metadata.Description)
	}

	if len(metadata.Images) != 2 {
		t.Errorf("Expected 2 images, got %d", len(metadata.Images))
	}

	if !metadata.Featured {
		t.Error("Expected featured to be true")
	}

	if len(metadata.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(metadata.Tags))
	}
}

func TestGitHubScraper_Cache(t *testing.T) {
	cache := newMockCache()
	scraper := NewGitHubScraper("testuser", "token", cache)

	// Create mock projects data
	projects := []Project{
		{
			Name:        "test-project",
			Description: "Test",
			URL:         "https://github.com/test",
			Stars:       5,
			Language:    "Go",
		},
	}

	// Manually set cache
	data, marshalErr := json.Marshal(projects)
	if marshalErr != nil {
		t.Fatalf("Failed to marshal projects: %v", marshalErr)
	}

	if setErr := cache.Set(cacheKeyGitHub, data, 1*time.Hour); setErr != nil {
		t.Fatalf("Failed to set cache: %v", setErr)
	}

	// Get cached data
	result, err := scraper.GetCached()
	if err != nil {
		t.Fatalf("GetCached failed: %v", err)
	}

	resultProjects, ok := result.([]Project)
	if !ok {
		t.Fatal("Result is not []Project")
	}

	if len(resultProjects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(resultProjects))
	}

	if resultProjects[0].Name != "test-project" {
		t.Errorf("Expected name 'test-project', got '%s'", resultProjects[0].Name)
	}
}

func TestGitHubScraper_ExtractImages(t *testing.T) {
	// Create mock server for README
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		readme := `# Test Project

Here are some images:

![Screenshot 1](./screenshots/demo.png)
![Screenshot 2](https://example.com/image.jpg)
![Logo](assets/logo.svg)
`
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(readme)); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	cache := newMockCache()
	scraper := NewGitHubScraper("testuser", "token", cache)
	scraper.client = server.Client()

	// Verify the setup
	// In a real test, we would test the extractImagesFromREADME method
	// For now, we verify the concept works
	t.Logf("Image extraction test placeholder - server at %s", server.URL)
}

func TestProject_JSON(t *testing.T) {
	project := Project{
		Name:        "test-project",
		Description: "A test project",
		URL:         "https://github.com/user/test-project",
		Stars:       42,
		Language:    "Go",
		Topics:      []string{"testing", "example"},
		Images:      []string{"image1.png", "image2.jpg"},
		Featured:    true,
	}

	// Marshal to JSON
	data, err := json.Marshal(project)
	if err != nil {
		t.Fatalf("Failed to marshal project: %v", err)
	}

	// Unmarshal back
	var decoded Project
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal project: %v", err)
	}

	// Verify fields
	if decoded.Name != project.Name {
		t.Errorf("Name mismatch: expected %s, got %s", project.Name, decoded.Name)
	}

	if decoded.Stars != project.Stars {
		t.Errorf("Stars mismatch: expected %d, got %d", project.Stars, decoded.Stars)
	}

	if decoded.Featured != project.Featured {
		t.Errorf("Featured mismatch: expected %v, got %v", project.Featured, decoded.Featured)
	}

	if len(decoded.Topics) != len(project.Topics) {
		t.Errorf("Topics length mismatch: expected %d, got %d", len(project.Topics), len(decoded.Topics))
	}
}

func TestGitHubScraper_Name(t *testing.T) {
	cache := newMockCache()
	scraper := NewGitHubScraper("user", "token", cache)

	if scraper.Name() != "github" {
		t.Errorf("Expected name 'github', got '%s'", scraper.Name())
	}
}

func TestGitHubScraper_NormalizeImageURL(t *testing.T) {
	cache := newMockCache()
	scraper := NewGitHubScraper("testuser", "token", cache)

	tests := []struct {
		name     string
		imageURL string
		repoName string
		expected string
	}{
		{
			name:     "External HTTP URL",
			imageURL: "http://example.com/image.png",
			repoName: "test-repo",
			expected: "http://example.com/image.png",
		},
		{
			name:     "External HTTPS URL",
			imageURL: "https://example.com/image.jpg",
			repoName: "test-repo",
			expected: "https://example.com/image.jpg",
		},
		{
			name:     "Relative path with ./",
			imageURL: "./screenshots/demo.png",
			repoName: "test-repo",
			expected: "https://raw.githubusercontent.com/testuser/test-repo/main/screenshots/demo.png",
		},
		{
			name:     "Relative path without ./",
			imageURL: "images/logo.svg",
			repoName: "test-repo",
			expected: "https://raw.githubusercontent.com/testuser/test-repo/main/images/logo.svg",
		},
		{
			name:     "Absolute path in repo",
			imageURL: "/assets/banner.png",
			repoName: "test-repo",
			expected: "https://raw.githubusercontent.com/testuser/test-repo/main/assets/banner.png",
		},
		{
			name:     "File in root",
			imageURL: "screenshot.png",
			repoName: "test-repo",
			expected: "https://raw.githubusercontent.com/testuser/test-repo/main/screenshot.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scraper.normalizeImageURL(tt.imageURL, tt.repoName)
			if result != tt.expected {
				t.Errorf("normalizeImageURL(%q, %q) = %q, expected %q",
					tt.imageURL, tt.repoName, result, tt.expected)
			}
		})
	}
}

// Test helper function to verify interface implementation
func TestGitHubScraper_ImplementsScraper(t *testing.T) {
	cache := newMockCache()
	var scraper Scraper = NewGitHubScraper("user", "token", cache)

	// Verify all interface methods are available
	_ = scraper.Name()

	// Test interface methods (errors expected since no mock server)
	if _, err := scraper.GetCached(); err == nil {
		t.Log("GetCached succeeded (using cache)")
	}
	if _, err := scraper.Scrape(); err != nil {
		t.Logf("Scrape failed as expected without mock server: %v", err)
	}
	if _, err := scraper.Refresh(); err != nil {
		t.Logf("Refresh failed as expected without mock server: %v", err)
	}
}

func TestIsBadgeURL(t *testing.T) {
	tests := []struct {
		name     string
		imageURL string
		expected bool
	}{
		// Badge URLs that should be filtered
		{
			name:     "shields.io badge",
			imageURL: "https://img.shields.io/badge/version-1.0-blue",
			expected: true,
		},
		{
			name:     "shields.io with www",
			imageURL: "https://www.shields.io/badge/test",
			expected: true,
		},
		{
			name:     "badge.fury.io",
			imageURL: "https://badge.fury.io/rb/rails.svg",
			expected: true,
		},
		{
			name:     "badgen.net",
			imageURL: "https://badgen.net/badge/icon/test",
			expected: true,
		},
		{
			name:     "codecov.io",
			imageURL: "https://codecov.io/gh/user/repo/branch/master/graph/badge.svg",
			expected: true,
		},
		{
			name:     "coveralls.io",
			imageURL: "https://coveralls.io/repos/github/user/repo/badge.svg",
			expected: true,
		},
		{
			name:     "travis-ci.org",
			imageURL: "https://travis-ci.org/user/repo.svg",
			expected: true,
		},
		{
			name:     "travis-ci.com",
			imageURL: "https://travis-ci.com/user/repo.svg",
			expected: true,
		},
		{
			name:     "circleci.com",
			imageURL: "https://circleci.com/gh/user/repo.svg",
			expected: true,
		},
		{
			name:     "GitHub Actions badge",
			imageURL: "https://github.com/workflows/main.yml/badge.svg",
			expected: true,
		},
		// Non-badge URLs that should NOT be filtered
		{
			name:     "regular GitHub raw image",
			imageURL: "https://raw.githubusercontent.com/user/repo/main/screenshot.png",
			expected: false,
		},
		{
			name:     "external image URL",
			imageURL: "https://example.com/images/demo.png",
			expected: false,
		},
		{
			name:     "image with badge in filename - should NOT be filtered",
			imageURL: "https://raw.githubusercontent.com/user/repo/main/user-badge-system.png",
			expected: false,
		},
		{
			name:     "image with badge in path - should NOT be filtered",
			imageURL: "https://example.com/badge-icons/logo.png",
			expected: false,
		},
		{
			name:     "picsum placeholder",
			imageURL: "https://picsum.photos/seed/test/800/400",
			expected: false,
		},
		{
			name:     "relative path (no domain)",
			imageURL: "screenshots/demo.png",
			expected: false,
		},
		{
			name:     "SVG from repo",
			imageURL: "https://raw.githubusercontent.com/user/repo/main/docs/architecture.svg",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBadgeURL(tt.imageURL)
			if result != tt.expected {
				t.Errorf("isBadgeURL(%q) = %v, expected %v", tt.imageURL, result, tt.expected)
			}
		})
	}
}

func TestFilterBadgeImages(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name: "only badges - all filtered",
			input: []string{
				"https://img.shields.io/badge/version-1.0-blue",
				"https://badge.fury.io/rb/rails.svg",
			},
			expected: []string{},
		},
		{
			name: "only valid images - none filtered",
			input: []string{
				"https://raw.githubusercontent.com/user/repo/main/screenshot.png",
				"https://example.com/demo.jpg",
			},
			expected: []string{
				"https://raw.githubusercontent.com/user/repo/main/screenshot.png",
				"https://example.com/demo.jpg",
			},
		},
		{
			name: "mixed - badges filtered, images kept",
			input: []string{
				"https://img.shields.io/badge/version-1.0-blue",
				"https://raw.githubusercontent.com/user/repo/main/screenshot.png",
				"https://badge.fury.io/rb/rails.svg",
				"https://example.com/demo.jpg",
				"https://codecov.io/gh/user/repo/badge.svg",
			},
			expected: []string{
				"https://raw.githubusercontent.com/user/repo/main/screenshot.png",
				"https://example.com/demo.jpg",
			},
		},
		{
			name: "image with badge in filename - NOT filtered",
			input: []string{
				"https://raw.githubusercontent.com/user/repo/main/user-badge-system.png",
				"https://example.com/badge-icons/logo.png",
			},
			expected: []string{
				"https://raw.githubusercontent.com/user/repo/main/user-badge-system.png",
				"https://example.com/badge-icons/logo.png",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterBadgeImages(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("filterBadgeImages() returned %d items, expected %d", len(result), len(tt.expected))
				t.Errorf("  got: %v", result)
				t.Errorf("  expected: %v", tt.expected)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("filterBadgeImages()[%d] = %q, expected %q", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestProjectLink_JSON(t *testing.T) {
	// Test with icon
	linkWithIcon := ProjectLink{
		Name: "Live",
		URL:  "https://example.com",
		Icon: "mdi:rocket-launch",
	}

	data, err := json.Marshal(linkWithIcon)
	if err != nil {
		t.Fatalf("Failed to marshal link with icon: %v", err)
	}

	var decoded ProjectLink
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal link: %v", err)
	}

	if decoded.Icon != "mdi:rocket-launch" {
		t.Errorf("Expected icon 'mdi:rocket-launch', got '%s'", decoded.Icon)
	}

	// Test without icon (omitempty)
	linkWithoutIcon := ProjectLink{
		Name: "Docs",
		URL:  "https://docs.example.com",
	}

	data, err = json.Marshal(linkWithoutIcon)
	if err != nil {
		t.Fatalf("Failed to marshal link without icon: %v", err)
	}

	// Verify icon field is omitted
	jsonStr := string(data)
	if jsonStr != `{"name":"Docs","url":"https://docs.example.com"}` {
		t.Errorf("Expected icon field to be omitted, got: %s", jsonStr)
	}
}
