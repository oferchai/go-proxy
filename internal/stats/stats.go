package stats

import "time"

// HostStats represents statistics for a single host
type HostStats struct {
	Host             string    `json:"host"`
	IPs              string    `json:"ips"`
	Connections      int64     `json:"connections"`
	RequestCount     int64     `json:"request_count"`
	BlockedAttempts  int64     `json:"blocked_attempts"`
	BytesTransferred uint64    `json:"bytes_transferred"`
	Blocked          bool      `json:"blocked"`
	LastSeen         time.Time `json:"last_seen"`
}
