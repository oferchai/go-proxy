package geo

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/gomodule/redigo/redis"
)

var (
	// Global instance of the geocache
	globalGeoCache *GeoCache
)

// Initialize sets up the global geocache instance
func Initialize(redisDSN string, memoryCacheSize int) error {
	var err error
	globalGeoCache, err = NewGeoCache(redisDSN, memoryCacheSize)
	if err != nil {
		return fmt.Errorf("failed to initialize geo cache: %w", err)
	}

	// Load data from Redis into memory cache on startup
	if err := preloadFromRedis(); err != nil {
		log.Printf("Warning: Failed to preload geo data from Redis: %v", err)
		// Non-fatal error, continue
	}

	return nil
}

// preloadFromRedis loads existing geolocation data from Redis into memory cache
func preloadFromRedis() error {
	// Exit if geocache not initialized
	if globalGeoCache == nil {
		return fmt.Errorf("geocache not initialized")
	}

	conn := globalGeoCache.redisPool.Get()
	defer conn.Close()

	// Get all geo keys
	keys, err := redis.Strings(conn.Do("KEYS", "geo:*"))
	if err != nil {
		return fmt.Errorf("failed to get geo keys from Redis: %w", err)
	}

	log.Printf("Preloading %d geolocation records from Redis", len(keys))

	// Load data for each key
	for _, key := range keys {
		host := strings.TrimPrefix(key, "geo:")

		// Get the data
		data, err := redis.Bytes(conn.Do("GET", key))
		if err != nil {
			continue
		}

		// Parse the JSON data
		var geoData GeoData
		if err := json.Unmarshal(data, &geoData); err != nil {
			continue
		}

		// Add to memory cache
		globalGeoCache.mutex.Lock()
		globalGeoCache.memCache.Add(host, &geoData)
		globalGeoCache.mutex.Unlock()
	}

	return nil
}

// RecordHostLocation asynchronously records geolocation data for a host
// This function is meant to be called from your proxy handler
func RecordHostLocation(host string) {
	if globalGeoCache == nil {
		return
	}

	// Clean up the host - remove port if present
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	// Skip private IPs and localhost
	if isPrivateIP(host) {
		return
	}

	// Perform lookup asynchronously
	globalGeoCache.LookupAsync(host)
}

// isPrivateIP checks if the given string is a private/local IP address
func isPrivateIP(ip string) bool {
	// Check if it's a valid IP
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Check if it's a private IP
	privateIPBlocks := []string{
		"127.0.0.0/8",    // localhost
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 (link-local)
		"::1/128",        // IPv6 localhost
		"fc00::/7",       // IPv6 unique local addr
		"fe80::/10",      // IPv6 link-local
	}

	for _, block := range privateIPBlocks {
		_, ipNet, err := net.ParseCIDR(block)
		if err != nil {
			continue
		}
		if ipNet.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// Shutdown cleans up resources used by the geolocation system
func Shutdown() {
	if globalGeoCache != nil {
		globalGeoCache.Close()
	}
}
