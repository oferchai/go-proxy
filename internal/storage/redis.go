package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-proxy/internal/logger"
	"go-proxy/internal/stats"

	"github.com/redis/go-redis/v9"
)

var (
	rdb *redis.Client
	ctx = context.Background()
)

func InitRedis(addr, password string) error {
	rdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	return checkConnection()
}

func checkConnection() error {
	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %v", err)
	}
	logger.Log("Successfully connected to Redis: %s", pong)
	return nil
}

func RecordHostActivity(host string, blocked bool, bytesTransferred uint64) error {
	// Log the incoming request
	fmt.Printf("\n=== Recording Host Activity ===\n")
	fmt.Printf("Host: %s\nBlocked: %v\nBytes Transferred: %d\n", host, blocked, bytesTransferred)

	if host == "" {
		fmt.Printf("❌ Error: Invalid host (empty)\n")
		return fmt.Errorf("invalid host: empty")
	}

	// Clean the host - remove any port number if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
		fmt.Printf("📝 Cleaned host (removed port): %s\n", host)
	}

	// Create timeframe-based keys
	now := time.Now()
	hourKey := fmt.Sprintf("HOST:%s:HOUR:%s", host, now.Format("2006-01-02-15"))
	dayKey := fmt.Sprintf("HOST:%s:DAY:%s", host, now.Format("2006-01-02"))

	fmt.Printf("🔑 Hour Key: %s\n", hourKey)
	fmt.Printf("🔑 Day Key: %s\n", dayKey)

	// Handle hourly stats - TTL: 15 days
	if err := updateHostStats(hourKey, host, blocked, bytesTransferred, 15*24*time.Hour); err != nil {
		fmt.Printf("❌ Error updating hourly stats: %v\n", err)
		return err
	}

	// Handle daily stats - TTL: 3 months (90 days)
	if err := updateHostStats(dayKey, host, blocked, bytesTransferred, 90*24*time.Hour); err != nil {
		fmt.Printf("❌ Error updating daily stats: %v\n", err)
		return err
	}

	return nil
}

func updateHostStats(key, host string, blocked bool, bytesTransferred uint64, expiration time.Duration) error {
	var hostStats stats.HostStats
	val, err := rdb.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		fmt.Printf("❌ Redis error getting stats for key %s: %v\n", key, err)
		return err
	}

	if err == redis.Nil {
		fmt.Printf("🆕 New stats entry for key: %s\n", key)
		// For new hosts, try to resolve IP addresses
		ips, err := net.LookupHost(host)
		ipList := "unknown"
		if err == nil && len(ips) > 0 {
			ipList = strings.Join(ips, ",")
			fmt.Printf("📍 Resolved IPs: %s\n", ipList)
		} else {
			fmt.Printf("⚠️ Could not resolve IPs for host\n")
		}

		hostStats = stats.HostStats{
			Host:             host,
			IPs:              ipList,
			Connections:      1,
			RequestCount:     1,
			BlockedAttempts:  0,
			BytesTransferred: bytesTransferred,
			Blocked:          blocked,
			LastSeen:         time.Now(),
		}
	} else {
		fmt.Printf("🔄 Updating existing stats for key: %s\n", key)
		if err := json.Unmarshal([]byte(val), &hostStats); err != nil {
			fmt.Printf("❌ Error unmarshaling stats: %v\n", err)
			return err
		}

		hostStats.Connections++
		hostStats.RequestCount++
		hostStats.BytesTransferred += bytesTransferred
		hostStats.LastSeen = time.Now()
		if blocked {
			hostStats.BlockedAttempts++
			hostStats.Blocked = true
		}
	}

	data, err := json.Marshal(hostStats)
	if err != nil {
		fmt.Printf("❌ Error marshaling stats: %v\n", err)
		return err
	}

	err = rdb.Set(ctx, key, data, expiration).Err()
	if err != nil {
		fmt.Printf("❌ Error saving to Redis: %v\n", err)
		return err
	}

	fmt.Printf("✅ Successfully updated stats for key: %s\n", key)
	fmt.Printf("⏰ Expiration: %v\n", expiration)
	return nil
}

func GetIPHistory(ip string) ([]stats.IPStats, error) {
	timeframesKey := fmt.Sprintf("IP:%s:timeframes", ip)
	keys, err := rdb.SMembers(ctx, timeframesKey).Result()
	if err != nil {
		return nil, err
	}

	var history []stats.IPStats
	for _, key := range keys {
		val, err := rdb.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var stats stats.IPStats
		if err := json.Unmarshal([]byte(val), &stats); err != nil {
			continue
		}
		history = append(history, stats)
	}

	return history, nil
}

func GetTimeframeData(start, end time.Time) ([]string, map[string]stats.HostStats, error) {
	fmt.Printf("\n=== Querying Timeframe Data ===\n")
	fmt.Printf("Start: %v\nEnd: %v\n", start, end)

	// Get all keys matching both hour and day patterns
	patterns := []string{"HOST:*:HOUR:*", "HOST:*:DAY:*"}
	var allKeys []string

	for _, pattern := range patterns {
		keys, err := rdb.Keys(ctx, pattern).Result()
		if err != nil {
			fmt.Printf("❌ Error getting keys for pattern %s: %v\n", pattern, err)
			continue
		}
		allKeys = append(allKeys, keys...)
	}

	var filteredKeys []string
	records := make(map[string]stats.HostStats)

	fmt.Printf("🔍 Found %d total keys to examine\n", len(allKeys))

	for _, key := range allKeys {
		// Extract timestamp from key
		parts := strings.Split(key, ":")
		if len(parts) < 4 {
			fmt.Printf("⚠️ Invalid key format: %s\n", key)
			continue
		}

		// Parse the timestamp based on whether it's hourly or daily
		var keyTime time.Time
		var err error
		if parts[2] == "HOUR" {
			keyTime, err = time.Parse("2006-01-02-15", parts[3])
		} else {
			keyTime, err = time.Parse("2006-01-02", parts[3])
		}

		if err != nil {
			fmt.Printf("❌ Error parsing time from key %s: %v\n", key)
			continue
		}

		// Check if the key's timestamp falls within our range
		if keyTime.Before(start) || keyTime.After(end) {
			continue
		}

		val, err := rdb.Get(ctx, key).Result()
		if err != nil {
			fmt.Printf("❌ Error reading key %s: %v\n", key, err)
			continue
		}

		var stats stats.HostStats
		if err := json.Unmarshal([]byte(val), &stats); err != nil {
			fmt.Printf("❌ Error parsing stats for key %s: %v\n", key, err)
			continue
		}

		filteredKeys = append(filteredKeys, key)
		records[key] = stats

		fmt.Printf("✅ Added record for key: %s\n", key)
		fmt.Printf("   Host: %s, Requests: %d, Bytes: %d\n",
			stats.Host, stats.RequestCount, stats.BytesTransferred)
	}

	fmt.Printf("📊 Found %d matching records\n", len(filteredKeys))
	fmt.Printf("=== End Query ===\n\n")

	return filteredKeys, records, nil
}

func CheckRedisConnection() error {
	result, err := rdb.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("Redis connection error: %v", err)
	}
	fmt.Printf("Redis connection test: %s\n", result)
	return nil
}

// Update DisplayAllHostStats to handle both hourly and daily stats
func DisplayAllHostStats() {
	fmt.Printf("\n=== Current Redis Host Statistics ===\n")

	// Get all keys matching our pattern
	patterns := []string{"HOST:*:HOUR:*", "HOST:*:DAY:*"}

	for _, pattern := range patterns {
		fmt.Printf("\n📊 Checking pattern: %s\n", pattern)
		keys, err := rdb.Keys(ctx, pattern).Result()
		if err != nil {
			fmt.Printf("❌ Error getting keys for pattern %s: %v\n", pattern, err)
			continue
		}

		fmt.Printf("📑 Found %d records\n", len(keys))

		// Sort keys for consistent display
		sort.Strings(keys)

		for _, key := range keys {
			val, err := rdb.Get(ctx, key).Result()
			if err != nil {
				fmt.Printf("❌ Error reading key %s: %v\n", key, err)
				continue
			}

			var stats stats.HostStats
			if err := json.Unmarshal([]byte(val), &stats); err != nil {
				fmt.Printf("❌ Error parsing stats for key %s: %v\n", key, err)
				continue
			}

			// Calculate time since last seen
			timeSince := time.Since(stats.LastSeen).Round(time.Second)

			fmt.Printf("\n🔑 Key: %s\n", key)
			fmt.Printf("📊 Statistics:\n")
			fmt.Printf("   Host: %s\n", stats.Host)
			fmt.Printf("   IPs: %s\n", stats.IPs)
			fmt.Printf("   Connections: %d\n", stats.Connections)
			fmt.Printf("   Requests: %d\n", stats.RequestCount)
			fmt.Printf("   Bytes Transferred: %d\n", stats.BytesTransferred)
			fmt.Printf("   Blocked Attempts: %d\n", stats.BlockedAttempts)
			fmt.Printf("   Blocked Status: %v\n", stats.Blocked)
			fmt.Printf("   Last Seen: %v (%v ago)\n", stats.LastSeen, timeSince)
			fmt.Printf("-------------------\n")
		}
	}

	fmt.Printf("=== End Statistics ===\n\n")
}

// GetDailyStats retrieves host statistics for a date range with specified granularity
func GetDailyStats(fromDate, toDate time.Time, hostFilter string, granularity string) ([]string, map[string]stats.HostStats, error) {
	// Default to day granularity if not specified
	if granularity == "" {
		granularity = "day"
	}

	var pattern string
	if granularity == "hour" {
		pattern = "HOST:*:HOUR:*"
	} else {
		pattern = "HOST:*:DAY:*"
	}

	// Apply host filter if provided
	if hostFilter != "" {
		if granularity == "hour" {
			pattern = "HOST:*" + hostFilter + "*:HOUR:*"
		} else {
			pattern = "HOST:*" + hostFilter + "*:DAY:*"
		}
	}

	var filteredKeys []string
	records := make(map[string]stats.HostStats)

	keys, err := rdb.Keys(ctx, pattern).Result()
	if err != nil {
		fmt.Printf("❌ Error getting keys: %v\n", err)
		return nil, nil, err
	}

	fmt.Printf("🔍 Found %d total keys to examine with pattern: %s\n", len(keys), pattern)

	for _, key := range keys {
		// Extract date from key based on granularity
		parts := strings.Split(key, ":")
		if len(parts) != 4 {
			fmt.Printf("⚠️ Invalid key format: %s\n", key)
			continue
		}

		var keyDate time.Time
		if granularity == "hour" {
			// Format: HOST:example.com:HOUR:2024-03-22-15
			hourParts := strings.Split(parts[3], "-")
			if len(hourParts) != 4 {
				fmt.Printf("⚠️ Invalid hour format: %s\n", parts[3])
				continue
			}
			dateStr := fmt.Sprintf("%s-%s-%s", hourParts[0], hourParts[1], hourParts[2])
			keyDate, err = time.Parse("2006-01-02", dateStr)
		} else {
			// Format: HOST:example.com:DAY:2024-03-22
			keyDate, err = time.Parse("2006-01-02", parts[3])
		}

		if err != nil {
			fmt.Printf("❌ Error parsing date from key %s: %v\n", key, err)
			continue
		}

		// Check if the key's date falls within our range
		if keyDate.Before(fromDate) || keyDate.After(toDate) {
			continue
		}

		val, err := rdb.Get(ctx, key).Result()
		if err != nil {
			fmt.Printf("❌ Error reading key %s: %v\n", key, err)
			continue
		}

		var stats stats.HostStats
		if err := json.Unmarshal([]byte(val), &stats); err != nil {
			fmt.Printf("❌ Error parsing stats for key %s: %v\n", key, err)
			continue
		}

		filteredKeys = append(filteredKeys, key)
		records[key] = stats

		fmt.Printf("✅ Added record for key: %s\n", key)
		fmt.Printf("   Host: %s, Requests: %d, Bytes: %d\n",
			stats.Host, stats.RequestCount, stats.BytesTransferred)
	}

	fmt.Printf("📊 Found %d matching records\n", len(filteredKeys))
	fmt.Printf("=== End Query ===\n\n")

	return filteredKeys, records, nil
}

// GetHourlyStats retrieves host statistics for specific hours in a day
func GetHourlyStats(date time.Time, fromHour, toHour int) ([]string, map[string]stats.HostStats, error) {
	fmt.Printf("\n=== Querying Hourly Stats ===\n")
	fmt.Printf("Date: %v\nHours: %02d:00-%02d:00\n",
		date.Format("2006-01-02"), fromHour, toHour)

	pattern := fmt.Sprintf("HOST:*:HOUR:%s-*", date.Format("2006-01-02"))
	var filteredKeys []string
	records := make(map[string]stats.HostStats)

	keys, err := rdb.Keys(ctx, pattern).Result()
	if err != nil {
		fmt.Printf("❌ Error getting keys: %v\n", err)
		return nil, nil, err
	}

	fmt.Printf("🔍 Found %d total keys to examine\n", len(keys))

	for _, key := range keys {
		// Extract hour from key (format: HOST:example.com:HOUR:2024-03-22-15)
		parts := strings.Split(key, ":")
		if len(parts) != 4 {
			fmt.Printf("⚠️ Invalid key format: %s\n", key)
			continue
		}

		// Split the date-hour part
		dateHourParts := strings.Split(parts[3], "-")
		if len(dateHourParts) != 4 {
			fmt.Printf("⚠️ Invalid date-hour format: %s\n", parts[3])
			continue
		}

		hour, err := strconv.Atoi(dateHourParts[3])
		if err != nil {
			fmt.Printf("❌ Error parsing hour from key %s: %v\n", key, err)
			continue
		}

		// Check if the hour falls within our range
		if hour < fromHour || hour > toHour {
			continue
		}

		val, err := rdb.Get(ctx, key).Result()
		if err != nil {
			fmt.Printf("❌ Error reading key %s: %v\n", key, err)
			continue
		}

		var stats stats.HostStats
		if err := json.Unmarshal([]byte(val), &stats); err != nil {
			fmt.Printf("❌ Error parsing stats for key %s: %v\n", key, err)
			continue
		}

		filteredKeys = append(filteredKeys, key)
		records[key] = stats

		fmt.Printf("✅ Added record for key: %s\n", key)
		fmt.Printf("   Host: %s, Hour: %02d:00, Requests: %d, Bytes: %d\n",
			stats.Host, hour, stats.RequestCount, stats.BytesTransferred)
	}

	fmt.Printf("📊 Found %d matching records\n", len(filteredKeys))
	fmt.Printf("=== End Query ===\n\n")

	return filteredKeys, records, nil
}
