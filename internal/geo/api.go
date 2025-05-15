package geo

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gomodule/redigo/redis"
)

// GeoAPIResponse represents the response format for the geo API endpoint
type GeoAPIResponse struct {
	Records map[string]*GeoData `json:"records"`
}

// AddAPIHandler adds a handler for geolocation data to the provided HTTP ServeMux
func AddAPIHandler(mux *http.ServeMux) {
	mux.HandleFunc("/api/geo", handleGeoAPI)
}

// handleGeoAPI handles requests for geolocation data
func handleGeoAPI(w http.ResponseWriter, r *http.Request) {
	// Only support GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if geocache is initialized
	if globalGeoCache == nil {
		http.Error(w, "Geolocation system not initialized", http.StatusInternalServerError)
		return
	}

	// Get all geolocation data from Redis
	response, err := getAllGeoData()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get geolocation data: %v", err), http.StatusInternalServerError)
		return
	}

	// Set headers and encode response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// getAllGeoData retrieves all geolocation data from Redis
func getAllGeoData() (*GeoAPIResponse, error) {
	conn := globalGeoCache.redisPool.Get()
	defer conn.Close()

	// Get all geo keys
	keys, err := redis.Strings(conn.Do("KEYS", "geo:*"))
	if err != nil {
		return nil, fmt.Errorf("failed to get geo keys: %w", err)
	}

	records := make(map[string]*GeoData)

	// Get data for each key
	for _, key := range keys {
		host := key[4:] // Remove "geo:" prefix

		// Get data from Redis
		data, err := redis.Bytes(conn.Do("GET", key))
		if err != nil {
			continue
		}

		// Parse JSON data
		var geoData GeoData
		if err := json.Unmarshal(data, &geoData); err != nil {
			continue
		}

		records[host] = &geoData
	}

	return &GeoAPIResponse{Records: records}, nil
}
