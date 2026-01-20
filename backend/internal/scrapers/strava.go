package scrapers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/mrcodeeu/homepage/internal/models"
	"github.com/mrcodeeu/homepage/internal/storage"
)

const (
	stravaAPIBase   = "https://www.strava.com/api/v3"
	stravaTokenURL  = "https://www.strava.com/oauth/token"
	cacheKeyStrava  = "strava_data"
	activityTypeRun = "Run"
)

// StravaScraper implements the Scraper interface for Strava API
type StravaScraper struct {
	clientID     string
	clientSecret string
	refreshToken string
	cache        storage.Cache
	cacheTTL     time.Duration
	client       *http.Client
	accessToken  string
	tokenExpiry  time.Time
}

// NewStravaScraper creates a new Strava scraper
func NewStravaScraper(clientID, clientSecret, refreshToken string, cache storage.Cache) *StravaScraper {
	return &StravaScraper{
		clientID:     clientID,
		clientSecret: clientSecret,
		refreshToken: refreshToken,
		cache:        cache,
		cacheTTL:     1 * time.Hour,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the scraper name
func (s *StravaScraper) Name() string {
	return "strava"
}

// tokenResponse represents Strava OAuth token response
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	ExpiresIn    int    `json:"expires_in"`
}

// stravaActivity represents activity from Strava API
type stravaActivity struct {
	ID                   int64   `json:"id"`
	Name                 string  `json:"name"`
	Distance             float64 `json:"distance"`
	MovingTime           float64 `json:"moving_time"`
	ElapsedTime          float64 `json:"elapsed_time"`
	TotalElevationGain   float64 `json:"total_elevation_gain"`
	Type                 string  `json:"type"`
	StartDate            string  `json:"start_date"`
	AverageSpeed         float64 `json:"average_speed"`
	MaxSpeed             float64 `json:"max_speed"`
	AverageHeartrate     float64 `json:"average_heartrate"`
	MaxHeartrate         float64 `json:"max_heartrate"`
}

// stravaStats represents athlete stats from Strava API
type stravaStats struct {
	AllRunTotals struct {
		Count         int     `json:"count"`
		Distance      float64 `json:"distance"`
		MovingTime    float64 `json:"moving_time"`
		ElapsedTime   float64 `json:"elapsed_time"`
		ElevationGain float64 `json:"elevation_gain"`
	} `json:"all_run_totals"`
	YTDRunTotals struct {
		Count         int     `json:"count"`
		Distance      float64 `json:"distance"`
		MovingTime    float64 `json:"moving_time"`
		ElapsedTime   float64 `json:"elapsed_time"`
		ElevationGain float64 `json:"elevation_gain"`
	} `json:"ytd_run_totals"`
}

// GetCached returns cached data or scrapes if needed
func (s *StravaScraper) GetCached() (any, error) {
	cached, err := s.cache.Get(cacheKeyStrava)
	if err != nil {
		return nil, fmt.Errorf("cache error: %w", err)
	}

	if cached != nil {
		var data models.StravaData
		if err := json.Unmarshal(cached, &data); err != nil {
			return s.Refresh()
		}
		return data, nil
	}

	return s.Refresh()
}

// Scrape fetches fresh data from Strava
func (s *StravaScraper) Scrape() (any, error) {
	// Ensure we have a valid access token
	if err := s.ensureAccessToken(); err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Fetch athlete stats
	stats, err := s.fetchAthleteStats()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stats: %w", err)
	}

	// Fetch recent running activities (last 30 days, max 200)
	activities, err := s.fetchActivities(200, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch activities: %w", err)
	}

	// Filter to running activities only
	runActivities := s.filterRunningActivities(activities)

	// Get recent 10 activities
	recentActivities := s.getRecentActivities(runActivities, 10)

	// Find best activities
	bestActivities := s.findBestActivities(runActivities)

	// Calculate personal records
	personalRecords := s.calculatePersonalRecords(runActivities)

	// Build result
	result := models.StravaData{
		TotalStats: models.StravaStats{
			Count:         stats.AllRunTotals.Count,
			Distance:      stats.AllRunTotals.Distance,
			MovingTime:    int(stats.AllRunTotals.MovingTime),
			ElapsedTime:   int(stats.AllRunTotals.ElapsedTime),
			ElevationGain: stats.AllRunTotals.ElevationGain,
		},
		YearToDateStats: models.StravaStats{
			Count:         stats.YTDRunTotals.Count,
			Distance:      stats.YTDRunTotals.Distance,
			MovingTime:    int(stats.YTDRunTotals.MovingTime),
			ElapsedTime:   int(stats.YTDRunTotals.ElapsedTime),
			ElevationGain: stats.YTDRunTotals.ElevationGain,
		},
		RecentActivities: recentActivities,
		BestActivities:   bestActivities,
		PersonalRecords:  personalRecords,
	}

	return result, nil
}

// Refresh forces a fresh scrape and updates cache
func (s *StravaScraper) Refresh() (any, error) {
	data, err := s.Scrape()
	if err != nil {
		return nil, err
	}

	// Update cache
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := s.cache.Set(cacheKeyStrava, jsonData, s.cacheTTL); err != nil {
		log.Printf("Warning: failed to update cache: %v", err)
	}

	return data, nil
}

// ensureAccessToken ensures we have a valid access token
func (s *StravaScraper) ensureAccessToken() error {
	// Check if we have a valid token
	if s.accessToken != "" && time.Now().Before(s.tokenExpiry) {
		return nil
	}

	// Exchange refresh token for access token
	data := url.Values{}
	data.Set("client_id", s.clientID)
	data.Set("client_secret", s.clientSecret)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", s.refreshToken)

	req, err := http.NewRequest("POST", stravaTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to exchange token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	s.accessToken = token.AccessToken
	s.tokenExpiry = time.Unix(token.ExpiresAt, 0)

	// Update refresh token if it changed
	if token.RefreshToken != "" {
		s.refreshToken = token.RefreshToken
	}

	return nil
}

// fetchAthleteStats fetches aggregate statistics
func (s *StravaScraper) fetchAthleteStats() (*stravaStats, error) {
	// Note: Strava requires athlete ID for stats endpoint
	// First, get athlete info to get the ID
	athleteURL := fmt.Sprintf("%s/athlete", stravaAPIBase)
	req, err := http.NewRequest("GET", athleteURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.accessToken))

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch athlete: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch athlete (status %d): %s", resp.StatusCode, string(body))
	}

	var athlete struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&athlete); err != nil {
		return nil, fmt.Errorf("failed to decode athlete: %w", err)
	}

	// Now fetch stats
	statsURL := fmt.Sprintf("%s/athletes/%d/stats", stravaAPIBase, athlete.ID)
	req, err = http.NewRequest("GET", statsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.accessToken))

	resp, err = s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch stats (status %d): %s", resp.StatusCode, string(body))
	}

	var stats stravaStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	return &stats, nil
}

// fetchActivities fetches activities from Strava
func (s *StravaScraper) fetchActivities(perPage, page int) ([]stravaActivity, error) {
	url := fmt.Sprintf("%s/athlete/activities?per_page=%d&page=%d", stravaAPIBase, perPage, page)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.accessToken))

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch activities: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch activities (status %d): %s", resp.StatusCode, string(body))
	}

	var activities []stravaActivity
	if err := json.NewDecoder(resp.Body).Decode(&activities); err != nil {
		return nil, fmt.Errorf("failed to decode activities: %w", err)
	}

	return activities, nil
}

// filterRunningActivities filters to running activities only
func (s *StravaScraper) filterRunningActivities(activities []stravaActivity) []models.StravaActivity {
	result := make([]models.StravaActivity, 0)
	for _, activity := range activities {
		if activity.Type == activityTypeRun {
			startDate, _ := time.Parse(time.RFC3339, activity.StartDate)
			averagePace := 0.0
			if activity.AverageSpeed > 0 {
				// Convert m/s to min/km
				averagePace = 1000.0 / (activity.AverageSpeed * 60)
			}

			result = append(result, models.StravaActivity{
				ID:                 activity.ID,
				Name:               activity.Name,
				Distance:           activity.Distance,
				MovingTime:         int(activity.MovingTime),
				ElapsedTime:        int(activity.ElapsedTime),
				TotalElevationGain: activity.TotalElevationGain,
				Type:               activity.Type,
				StartDate:          startDate,
				AveragePace:        averagePace,
				AverageSpeed:       activity.AverageSpeed,
				MaxSpeed:           activity.MaxSpeed,
				AverageHeartrate:   activity.AverageHeartrate,
				MaxHeartrate:       activity.MaxHeartrate,
			})
		}
	}
	return result
}

// getRecentActivities returns the N most recent activities
func (s *StravaScraper) getRecentActivities(activities []models.StravaActivity, count int) []models.StravaActivity {
	// Sort by date descending
	sorted := make([]models.StravaActivity, len(activities))
	copy(sorted, activities)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartDate.After(sorted[j].StartDate)
	})

	if len(sorted) > count {
		return sorted[:count]
	}
	return sorted
}

// findBestActivities finds the best activities by various metrics
func (s *StravaScraper) findBestActivities(activities []models.StravaActivity) models.StravaBestRecords {
	if len(activities) == 0 {
		return models.StravaBestRecords{}
	}

	best := models.StravaBestRecords{
		LongestDistance: activities[0],
		LongestTime:     activities[0],
		FastestPace:     activities[0],
		MostElevation:   activities[0],
	}

	for _, activity := range activities {
		if activity.Distance > best.LongestDistance.Distance {
			best.LongestDistance = activity
		}
		if activity.MovingTime > best.LongestTime.MovingTime {
			best.LongestTime = activity
		}
		if activity.AveragePace < best.FastestPace.AveragePace && activity.AveragePace > 0 {
			best.FastestPace = activity
		}
		if activity.TotalElevationGain > best.MostElevation.TotalElevationGain {
			best.MostElevation = activity
		}
	}

	return best
}

// calculatePersonalRecords finds PRs for standard distances
func (s *StravaScraper) calculatePersonalRecords(activities []models.StravaActivity) []models.StravaRecord {
	// Standard race distances (in meters)
	distances := map[string]float64{
		"5k":            5000,
		"10k":           10000,
		"half_marathon": 21097.5,
		"marathon":      42195,
	}

	records := make([]models.StravaRecord, 0)
	foundRecords := make(map[string]*models.StravaRecord)

	// Find fastest time for each distance (within 2% tolerance)
	for _, activity := range activities {
		for recordType, targetDistance := range distances {
			tolerance := targetDistance * 0.02 // 2% tolerance
			if activity.Distance >= targetDistance-tolerance && activity.Distance <= targetDistance+tolerance {
				existing, exists := foundRecords[recordType]
				if !exists || activity.MovingTime < existing.Time {
					foundRecords[recordType] = &models.StravaRecord{
						Type:     recordType,
						Time:     activity.MovingTime,
						Distance: activity.Distance,
						Date:     activity.StartDate,
						Activity: activity,
					}
				}
			}
		}
	}

	// Convert map to slice
	for _, record := range foundRecords {
		records = append(records, *record)
	}

	// Sort by distance
	sort.Slice(records, func(i, j int) bool {
		return distances[records[i].Type] < distances[records[j].Type]
	})

	return records
}
