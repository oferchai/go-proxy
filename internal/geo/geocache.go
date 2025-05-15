package geo

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	lru "github.com/hashicorp/golang-lru"
)

// GeoData represents geolocation information
type GeoData struct {
	CountryCode string  `json:"country_code"`
	CountryName string  `json:"country"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Region      string  `json:"region"`
	TimeZone    string  `json:"timezone"`
}

// GeoJSResponse represents the response from the GeoJS API
type GeoJSResponse struct {
	IP          string  `json:"ip"`
	CountryCode string  `json:"country_code"`
	CountryName string  `json:"country"`
	Region      string  `json:"region"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	TimeZone    string  `json:"timezone"`
}

// GeoCache provides thread-safe geolocation lookups with caching
type GeoCache struct {
	memCache    *lru.Cache
	mutex       sync.RWMutex
	redisPool   *redis.Pool
	httpClient  *http.Client
	rateLimiter *time.Ticker // Basic rate limiter for API calls
	debugMode   bool         // When true, logs detailed information
}

// NewGeoCache initializes the geolocation system with Redis and memory cache
func NewGeoCache(redisDSN string, memoryCacheSize int, debug bool) (*GeoCache, error) {
	// Initialize in-memory LRU cache
	memCache, err := lru.New(memoryCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create LRU cache: %w", err)
	}

	// Initialize Redis connection pool
	redisPool := &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", redisDSN)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}

	// Initialize HTTP client with reasonable timeouts
	httpClient := &http.Client{
		Timeout: 2 * time.Second, // Short timeout since this is for synchronous requests
	}

	// Create a basic rate limiter (1 request per second)
	rateLimiter := time.NewTicker(time.Second)

	// Create the geo cache
	cache := &GeoCache{
		memCache:    memCache,
		redisPool:   redisPool,
		httpClient:  httpClient,
		rateLimiter: rateLimiter,
		debugMode:   debug,
	}

	// Start a background goroutine to discard ticker values if not used
	go func() {
		for range rateLimiter.C {
			// Discard ticker values when not consumed by API calls
		}
	}()

	return cache, nil
}

// lookupGeoJS performs a lookup using the GeoJS API
func (g *GeoCache) lookupGeoJS(host string) (*GeoData, error) {
	// Rate limit API calls
	<-g.rateLimiter.C

	// Make API request to GeoJS
	resp, err := g.httpClient.Get(fmt.Sprintf("https://get.geojs.io/v1/ip/geo/%s.json", host))
	if err != nil {
		return nil, fmt.Errorf("GeoJS API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GeoJS API returned non-OK status: %d", resp.StatusCode)
	}

	// Read and parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read GeoJS response: %w", err)
	}

	var geoJSResp GeoJSResponse
	if err := json.Unmarshal(body, &geoJSResp); err != nil {
		return nil, fmt.Errorf("failed to parse GeoJS response: %w", err)
	}

	// Convert to our internal format
	geoData := &GeoData{
		CountryCode: geoJSResp.CountryCode,
		CountryName: geoJSResp.CountryName,
		City:        geoJSResp.City,
		Latitude:    geoJSResp.Latitude,
		Longitude:   geoJSResp.Longitude,
		Region:      geoJSResp.Region,
		TimeZone:    geoJSResp.TimeZone,
	}

	return geoData, nil
}

// getFromRedis attempts to retrieve geolocation data from Redis
func (g *GeoCache) getFromRedis(host string) (*GeoData, error) {
	conn := g.redisPool.Get()
	defer conn.Close()

	// Try to get data from Redis
	redisKey := fmt.Sprintf("geo:%s", host)
	data, err := redis.Bytes(conn.Do("GET", redisKey))
	if err != nil {
		if err == redis.ErrNil {
			// Key not found, not an error
			return nil, nil
		}
		return nil, fmt.Errorf("Redis GET failed: %w", err)
	}

	// Parse the JSON data
	var geoData GeoData
	if err := json.Unmarshal(data, &geoData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal geo data: %w", err)
	}

	return &geoData, nil
}

// saveToRedis saves geolocation data to Redis
func (g *GeoCache) saveToRedis(host string, data *GeoData) error {
	conn := g.redisPool.Get()
	defer conn.Close()

	// Convert data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal geo data: %w", err)
	}

	// Store in Redis with 7-day expiration
	redisKey := fmt.Sprintf("geo:%s", host)
	_, err = conn.Do("SETEX", redisKey, 86400*7, jsonData)
	if err != nil {
		return fmt.Errorf("Redis SETEX failed: %w", err)
	}

	return nil
}

// Lookup performs a geolocation lookup with caching
func (g *GeoCache) Lookup(host string) (*GeoData, error) {
	// Check in-memory cache first (fast path)
	g.mutex.RLock()
	if data, found := g.memCache.Get(host); found {
		g.mutex.RUnlock()
		return data.(*GeoData), nil
	}
	g.mutex.RUnlock()

	// Check Redis next
	data, err := g.getFromRedis(host)
	if err != nil {
		// Log Redis error but continue to API lookup 
		// Reduced verbosity, only log if debug enabled
		if g.debugMode {
			g.logError("Redis lookup error: %v", err)
		}
	} else if data != nil {
		// Found in Redis, update in-memory cache
		g.mutex.Lock()
		g.memCache.Add(host, data)
		g.mutex.Unlock()
		return data, nil
	}

	// Not found in local caches, perform API lookup
	geoData, err := g.lookupGeoJS(host)
	if err != nil {
		return nil, fmt.Errorf("geolocation failed: %w", err)
	}

	// Store in both caches
	g.mutex.Lock()
	g.memCache.Add(host, geoData)
	g.mutex.Unlock()

	// Save to Redis asynchronously
	go func() {
		if err := g.saveToRedis(host, geoData); err != nil {
			g.logError("Failed to save geo data to Redis: %v", err)
		}
	}()

	return geoData, nil
}

// LookupAsync performs a non-blocking geolocation lookup
// This is ideal for proxy usage where you don't want to block the request handling
func (g *GeoCache) LookupAsync(host string) {
	go func() {
		_, _ = g.Lookup(host)
	}()
}

// Close cleans up resources used by the geo cache
func (g *GeoCache) Close() {
	g.rateLimiter.Stop()
	g.redisPool.Close()
}

// logError logs an error message if debug mode is enabled
func (g *GeoCache) logError(format string, args ...interface{}) {
	if g.debugMode {
		log.Printf("[GEO ERROR] "+format, args...)
	}
}

// logInfo logs an informational message if debug mode is enabled
func (g *GeoCache) logInfo(format string, args ...interface{}) {
	if g.debugMode {
		log.Printf("[GEO INFO] "+format, args...)
	}
}
