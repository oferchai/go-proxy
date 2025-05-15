package api

import "go-proxy/internal/stats"

// DailyStatsRequest represents the request structure for daily statistics
type DailyStatsRequest struct {
	FromDate    string `json:"from_date"`    // Format: "2024-03-22"
	ToDate      string `json:"to_date"`      // Format: "2024-03-24"
	HostFilter  string `json:"host_filter"`  // Format: "example.com"
	Granularity string `json:"granularity"`  // "day" or "hour"
}

// HourlyStatsRequest represents the request structure for hourly statistics
type HourlyStatsRequest struct {
	Date     string `json:"date"`      // Format: "2024-03-22"
	FromHour int    `json:"from_hour"` // 0-23
	ToHour   int    `json:"to_hour"`   // 0-23
}

// StatsResponse represents the response structure for both endpoints
type StatsResponse struct {
	Keys    []string                   `json:"keys"`
	Records map[string]stats.HostStats `json:"records"`
	Error   string                     `json:"error,omitempty"`
}
