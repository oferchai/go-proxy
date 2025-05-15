package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go-proxy/internal/logger"
	"go-proxy/internal/metrics"
	"go-proxy/internal/storage"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// HandleDailyStats handles requests for daily or hourly statistics based on a date range
func (h *Handler) HandleDailyStats(w http.ResponseWriter, r *http.Request) {
	logger.Log("Handling stats request from %s", r.RemoteAddr)

	var fromDate, toDate time.Time
	var err error
	var hostFilterStr = ""
	var granularity = "day" // Default to day

	switch r.Method {
	case http.MethodGet:
		fromStr := r.URL.Query().Get("from_date")
		toStr := r.URL.Query().Get("to_date")
		hostFilterStr = r.URL.Query().Get("host_filter")
		
		// Get granularity if provided
		if g := r.URL.Query().Get("granularity"); g != "" {
			granularity = g
		}

		if fromStr == "" || toStr == "" {
			sendJSONResponse(w, StatsResponse{
				Error: "Missing from_date or to_date parameters",
			}, http.StatusBadRequest)
			return
		}

		fromDate, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			sendJSONResponse(w, StatsResponse{
				Error: "Invalid from_date format. Use YYYY-MM-DD",
			}, http.StatusBadRequest)
			return
		}

		toDate, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			sendJSONResponse(w, StatsResponse{
				Error: "Invalid to_date format. Use YYYY-MM-DD",
			}, http.StatusBadRequest)
			return
		}

	case http.MethodPost:
		var req DailyStatsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendJSONResponse(w, StatsResponse{
				Error: "Invalid request format",
			}, http.StatusBadRequest)
			return
		}

		fromDate, err = time.Parse("2006-01-02", req.FromDate)
		if err != nil {
			sendJSONResponse(w, StatsResponse{
				Error: "Invalid from_date format. Use YYYY-MM-DD",
			}, http.StatusBadRequest)
			return
		}

		toDate, err = time.Parse("2006-01-02", req.ToDate)
		if err != nil {
			sendJSONResponse(w, StatsResponse{
				Error: "Invalid to_date format. Use YYYY-MM-DD",
			}, http.StatusBadRequest)
			return
		}

		hostFilterStr = req.HostFilter
		
		// Use granularity if provided
		if req.Granularity != "" {
			granularity = req.Granularity
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate granularity
	if granularity != "day" && granularity != "hour" {
		sendJSONResponse(w, StatsResponse{
			Error: "Invalid granularity. Use 'day' or 'hour'",
		}, http.StatusBadRequest)
		return
	}

	// Add one day to toDate to include the entire last day
	toDate = toDate.Add(24 * time.Hour)

	keys, records, err := storage.GetDailyStats(fromDate, toDate, hostFilterStr, granularity)
	if err != nil {
		logger.Log("API Error: Failed to fetch %s stats: %v", granularity, err)
		sendJSONResponse(w, StatsResponse{
			Error: "Failed to fetch data: " + err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	logger.Log("%s stats query: %v to %v, found %d records",
		granularity, fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"), len(keys))

	sendJSONResponse(w, StatsResponse{
		Keys:    keys,
		Records: records,
	}, http.StatusOK)
}

// HandleHourlyStats handles requests for hourly statistics
func (h *Handler) HandleHourlyStats(w http.ResponseWriter, r *http.Request) {
	logger.Log("Handling hourly stats request from %s", r.RemoteAddr)

	var date time.Time
	var fromHour, toHour int
	var err error

	switch r.Method {
	case http.MethodGet:
		dateStr := r.URL.Query().Get("date")
		fromHourStr := r.URL.Query().Get("from_hour")
		toHourStr := r.URL.Query().Get("to_hour")

		if dateStr == "" || fromHourStr == "" || toHourStr == "" {
			sendJSONResponse(w, StatsResponse{
				Error: "Missing required parameters",
			}, http.StatusBadRequest)
			return
		}

		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			sendJSONResponse(w, StatsResponse{
				Error: "Invalid date format. Use YYYY-MM-DD",
			}, http.StatusBadRequest)
			return
		}

		fmt.Sscanf(fromHourStr, "%d", &fromHour)
		fmt.Sscanf(toHourStr, "%d", &toHour)

		if fromHour < 0 || fromHour > 23 || toHour < 0 || toHour > 23 {
			sendJSONResponse(w, StatsResponse{
				Error: "Hours must be between 0 and 23",
			}, http.StatusBadRequest)
			return
		}

	case http.MethodPost:
		var req HourlyStatsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendJSONResponse(w, StatsResponse{
				Error: "Invalid request format",
			}, http.StatusBadRequest)
			return
		}

		date, err = time.Parse("2006-01-02", req.Date)
		if err != nil {
			sendJSONResponse(w, StatsResponse{
				Error: "Invalid date format. Use YYYY-MM-DD",
			}, http.StatusBadRequest)
			return
		}

		fromHour = req.FromHour
		toHour = req.ToHour

		if fromHour < 0 || fromHour > 23 || toHour < 0 || toHour > 23 {
			sendJSONResponse(w, StatsResponse{
				Error: "Hours must be between 0 and 23",
			}, http.StatusBadRequest)
			return
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	keys, records, err := storage.GetHourlyStats(date, fromHour, toHour)
	if err != nil {
		logger.Log("API Error: Failed to fetch hourly stats: %v", err)
		sendJSONResponse(w, StatsResponse{
			Error: "Failed to fetch data: " + err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	logger.Log("Hourly stats query: %v (%02d:00-%02d:00), found %d records",
		date.Format("2006-01-02"), fromHour, toHour, len(keys))

	sendJSONResponse(w, StatsResponse{
		Keys:    keys,
		Records: records,
	}, http.StatusOK)
}

func (h *Handler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	// Get the last hour's stats
	now := time.Now()
	hourAgo := now.Add(-1 * time.Hour)

	// Get data by hour granularity
	_, records, err := storage.GetDailyStats(hourAgo, now, "", "hour")
	if err != nil {
		http.Error(w, "Failed to fetch metrics", http.StatusInternalServerError)
		return
	}

	metrics := metrics.TransformHostStats(records)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func sendJSONResponse(w http.ResponseWriter, response interface{}, statusCode int) {
	// Add CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
