package models

import "time"

// GeneratedData wraps any data source with metadata
type GeneratedData struct {
	GeneratedAt time.Time   `json:"generated_at"`
	Source      string      `json:"source"`
	Version     string      `json:"version"`
	Data        interface{} `json:"data"`
}

// StravaData contains all Strava-related data
type StravaData struct {
	TotalStats       StravaStats       `json:"total_stats"`
	YearToDateStats  StravaStats       `json:"year_to_date_stats"`
	RecentActivities []StravaActivity  `json:"recent_activities"`
	BestActivities   StravaBestRecords `json:"best_activities"`
	PersonalRecords  []StravaRecord    `json:"personal_records"`
}

// StravaStats contains aggregate statistics
type StravaStats struct {
	Count         int     `json:"count"`
	Distance      float64 `json:"distance"`       // meters
	MovingTime    int     `json:"moving_time"`    // seconds
	ElapsedTime   int     `json:"elapsed_time"`   // seconds
	ElevationGain float64 `json:"elevation_gain"` // meters
}

// StravaActivity represents a single activity
type StravaActivity struct {
	ID                 int64     `json:"id"`
	Name               string    `json:"name"`
	Distance           float64   `json:"distance"`             // meters
	MovingTime         int       `json:"moving_time"`          // seconds
	ElapsedTime        int       `json:"elapsed_time"`         // seconds
	TotalElevationGain float64   `json:"total_elevation_gain"` // meters
	Type               string    `json:"type"`
	StartDate          time.Time `json:"start_date"`
	AveragePace        float64   `json:"average_pace"`  // min/km
	AverageSpeed       float64   `json:"average_speed"` // m/s
	MaxSpeed           float64   `json:"max_speed"`     // m/s
	AverageHeartrate   float64   `json:"average_heartrate,omitempty"`
	MaxHeartrate       float64   `json:"max_heartrate,omitempty"`
}

// StravaBestRecords contains best/longest activities
type StravaBestRecords struct {
	LongestDistance StravaActivity `json:"longest_distance"`
	LongestTime     StravaActivity `json:"longest_time"`
	FastestPace     StravaActivity `json:"fastest_pace"`
	MostElevation   StravaActivity `json:"most_elevation"`
}

// StravaRecord represents a personal record
type StravaRecord struct {
	Type     string         `json:"type"`     // "5k", "10k", "half_marathon", "marathon", etc.
	Time     int            `json:"time"`     // seconds
	Distance float64        `json:"distance"` // meters
	Date     time.Time      `json:"date"`
	Activity StravaActivity `json:"activity"`
}

// LinkedInData contains LinkedIn profile data
type LinkedInData struct {
	Profile    LinkedInProfile      `json:"profile"`
	Experience []LinkedInExperience `json:"experience"`
	Education  []LinkedInEducation  `json:"education"`
	Skills     []string             `json:"skills"`
}

// LinkedInProfile contains basic profile information
type LinkedInProfile struct {
	Name     string `json:"name"`
	Headline string `json:"headline"`
	Location string `json:"location"`
	Summary  string `json:"summary"`
	PhotoURL string `json:"photo_url,omitempty"`
}

// LinkedInExperience represents work experience
type LinkedInExperience struct {
	Title       string `json:"title"`
	Company     string `json:"company"`
	CompanyLogo string `json:"company_logo,omitempty"`
	Location    string `json:"location"`
	StartDate   string `json:"start_date"` // "YYYY-MM" format
	EndDate     string `json:"end_date"`   // "YYYY-MM" or "Present"
	Description string `json:"description"`
	Duration    string `json:"duration,omitempty"`
}

// LinkedInEducation represents education
type LinkedInEducation struct {
	School      string `json:"school"`
	SchoolLogo  string `json:"school_logo,omitempty"`
	Degree      string `json:"degree"`
	Field       string `json:"field"`
	StartDate   string `json:"start_date"` // "YYYY" format
	EndDate     string `json:"end_date"`   // "YYYY" format
	Description string `json:"description,omitempty"`
}
