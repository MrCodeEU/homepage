package scrapers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/mrcodeeu/homepage/internal/models"
	"github.com/mrcodeeu/homepage/internal/storage"
)

const (
	cacheKeyLinkedIn = "linkedin_data"
	userAgent        = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// LinkedInScraper implements the Scraper interface for LinkedIn public profiles
type LinkedInScraper struct {
	profileURL string
	cache      storage.Cache
	cacheTTL   time.Duration
	client     *http.Client
}

// NewLinkedInScraper creates a new LinkedIn scraper
func NewLinkedInScraper(profileURL string, cache storage.Cache) *LinkedInScraper {
	return &LinkedInScraper{
		profileURL: profileURL,
		cache:      cache,
		cacheTTL:   24 * time.Hour, // Cache for 24 hours
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the scraper name
func (l *LinkedInScraper) Name() string {
	return "linkedin"
}

// GetCached returns cached data or scrapes if needed
func (l *LinkedInScraper) GetCached() (any, error) {
	cached, err := l.cache.Get(cacheKeyLinkedIn)
	if err != nil {
		return nil, fmt.Errorf("cache error: %w", err)
	}

	if cached != nil {
		var data models.LinkedInData
		if err := json.Unmarshal(cached, &data); err != nil {
			return l.Refresh()
		}
		return data, nil
	}

	return l.Refresh()
}

// Scrape fetches fresh data from LinkedIn public profile
func (l *LinkedInScraper) Scrape() (any, error) {
	// Fetch the profile page
	html, err := l.fetchProfileHTML()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile: %w", err)
	}

	// Try to extract structured data from JSON-LD
	data, err := l.extractStructuredData(html)
	if err != nil {
		// If structured data extraction fails, try HTML parsing as fallback
		log.Printf("Warning: structured data extraction failed, attempting HTML parsing: %v", err)
		data, err = l.parseProfileHTML(html)
		if err != nil {
			return nil, fmt.Errorf("failed to parse profile: %w", err)
		}
	}

	return data, nil
}

// Refresh forces a fresh scrape and updates cache
func (l *LinkedInScraper) Refresh() (any, error) {
	data, err := l.Scrape()
	if err != nil {
		return nil, err
	}

	// Update cache
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := l.cache.Set(cacheKeyLinkedIn, jsonData, l.cacheTTL); err != nil {
		log.Printf("Warning: failed to update cache: %v", err)
	}

	return data, nil
}

// fetchProfileHTML fetches the HTML content of the profile page
func (l *LinkedInScraper) fetchProfileHTML() (string, error) {
	req, err := http.NewRequest("GET", l.profileURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers to mimic a browser
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := l.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}

// extractStructuredData tries to extract data from JSON-LD structured data
func (l *LinkedInScraper) extractStructuredData(html string) (*models.LinkedInData, error) {
	// LinkedIn often includes JSON-LD structured data in public profiles
	// Look for <script type="application/ld+json">
	jsonLDRegex := regexp.MustCompile(`<script type="application/ld\+json">(.*?)</script>`)
	matches := jsonLDRegex.FindAllStringSubmatch(html, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("no structured data found")
	}

	// Parse each JSON-LD block to find person data
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(match[1]), &jsonData); err != nil {
			continue
		}

		// Check if this is a Person schema
		if typeVal, ok := jsonData["@type"].(string); ok && typeVal == "Person" {
			return l.parseStructuredPersonData(jsonData)
		}
	}

	return nil, fmt.Errorf("no person structured data found")
}

// parseStructuredPersonData parses structured data into LinkedInData
func (l *LinkedInScraper) parseStructuredPersonData(data map[string]interface{}) (*models.LinkedInData, error) {
	result := &models.LinkedInData{
		Profile: models.LinkedInProfile{},
	}

	// Extract basic profile info
	if name, ok := data["name"].(string); ok {
		result.Profile.Name = name
	}
	if headline, ok := data["jobTitle"].(string); ok {
		result.Profile.Headline = headline
	}
	if location, ok := data["address"].(map[string]interface{}); ok {
		if addressLocality, ok := location["addressLocality"].(string); ok {
			result.Profile.Location = addressLocality
		}
	}
	if image, ok := data["image"].(string); ok {
		result.Profile.PhotoURL = image
	}
	if description, ok := data["description"].(string); ok {
		result.Profile.Summary = description
	}

	// Note: Work experience and education are typically not in structured data
	// These will need to be parsed from HTML or provided via fallback

	return result, nil
}

// parseProfileHTML parses the HTML content to extract profile data (fallback method)
func (l *LinkedInScraper) parseProfileHTML(html string) (*models.LinkedInData, error) {
	// This is a basic fallback parser
	// LinkedIn's HTML structure changes frequently, so this may need updates

	result := &models.LinkedInData{
		Profile: models.LinkedInProfile{},
		Experience: []models.LinkedInExperience{},
		Education: []models.LinkedInEducation{},
		Skills: []string{},
	}

	// Try to extract name from title tag
	titleRegex := regexp.MustCompile(`<title>(.*?)</title>`)
	if matches := titleRegex.FindStringSubmatch(html); len(matches) > 1 {
		// Title usually contains name
		title := strings.TrimSpace(matches[1])
		// Remove " | LinkedIn" suffix if present
		title = strings.TrimSuffix(title, " | LinkedIn")
		result.Profile.Name = title
	}

	// Try to extract headline/description from meta tags
	descRegex := regexp.MustCompile(`<meta name="description" content="(.*?)"`)
	if matches := descRegex.FindStringSubmatch(html); len(matches) > 1 {
		result.Profile.Headline = strings.TrimSpace(matches[1])
	}

	// Try to extract og:image for profile photo
	imageRegex := regexp.MustCompile(`<meta property="og:image" content="(.*?)"`)
	if matches := imageRegex.FindStringSubmatch(html); len(matches) > 1 {
		result.Profile.PhotoURL = strings.TrimSpace(matches[1])
	}

	// Note: Extracting work experience and education from HTML is complex
	// and LinkedIn's structure changes frequently. This is marked as TODO
	// for manual configuration or enhanced scraping logic.

	log.Printf("LinkedIn scraper: Basic profile data extracted. Work experience and education require manual configuration or enhanced scraping.")

	return result, nil
}

// ManualData allows setting LinkedIn data manually (for initial MVP)
func (l *LinkedInScraper) ManualData(data models.LinkedInData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := l.cache.Set(cacheKeyLinkedIn, jsonData, l.cacheTTL); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}
