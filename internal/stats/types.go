package stats

import (
	"sync"
	"time"
)

type IPStats struct {
	IP               string    `json:"ip"`
	Hostname         string    `json:"hostname"`
	Connections      int64     `json:"connections"`
	RequestCount     int64     `json:"request_count"`
	BlockedAttempts  int64     `json:"blocked_attempts"`
	BytesTransferred uint64    `json:"bytes_transferred"`
	Blocked          bool      `json:"blocked"`
	LastSeen         time.Time `json:"last_seen"`
}

type ProxyStats struct {
	RequestCount     int64
	BlockedCount     int64
	BytesTransferred uint64
	StatusCodes      map[int]int64
	Methods          map[string]int64
	IPAddresses      map[string]IPStats
	mu               sync.RWMutex
}
