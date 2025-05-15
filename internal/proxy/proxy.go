package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"go-proxy/internal/config"
	"go-proxy/internal/geo" // Add geolocation package
	"go-proxy/internal/logger"
	"go-proxy/internal/stats"
	"go-proxy/internal/storage"
)

type Server struct {
	cfg         *config.Config
	blockedRegs []*regexp.Regexp
	stats       *ProxyStats
	statsMutex  sync.RWMutex
}

type ProxyStats struct {
	HostStats map[string]*stats.HostStats
}

func NewServer(cfg *config.Config) *Server {
	s := &Server{
		cfg:         cfg,
		blockedRegs: make([]*regexp.Regexp, 0),
		stats: &ProxyStats{
			HostStats: make(map[string]*stats.HostStats),
		},
	}

	// Start periodic stats saving
	go s.periodicStatsSave()

	// Load blacklist if file is specified
	if cfg.BlockFile != "" {
		if err := s.loadBlacklist(); err != nil {
			logger.Log("Error loading blacklist: %v", err)
		}
	}

	// Start stats monitoring
	s.startStatsMonitoring()

	return s
}

// Add method to load blacklist
func (s *Server) loadBlacklist() error {
	file, err := os.Open(s.cfg.BlockFile)
	if err != nil {
		return fmt.Errorf("failed to open blacklist file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		pattern := strings.TrimSpace(scanner.Text())
		if pattern == "" || strings.HasPrefix(pattern, "#") {
			continue
		}

		reg, err := regexp.Compile(pattern)
		if err != nil {
			logger.Log("Invalid regex pattern '%s': %v", pattern, err)
			continue
		}
		s.blockedRegs = append(s.blockedRegs, reg)
	}

	logger.Log("Loaded %d blacklist patterns", len(s.blockedRegs))
	return scanner.Err()
}

// Update isBlocked method
func (s *Server) isBlocked(host string) bool {
	if len(s.blockedRegs) == 0 {
		return false
	}

	for _, reg := range s.blockedRegs {
		if reg.MatchString(host) {
			return true
		}
	}
	return false
}

// Implement http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// This method will handle all HTTPS requests
	s.HandleHTTPS(w, r)
}

// Add method to periodically save stats
func (s *Server) periodicStatsSave() {
	ticker := time.NewTicker(60 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.saveStatsToRedis()
	}
}

// Add method to save accumulated stats to Redis
func (s *Server) saveStatsToRedis() {
	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()

	now := time.Now()

	for host, stats := range s.stats.HostStats {
		if stats.Connections > 0 || stats.BlockedAttempts > 0 {
			stats.LastSeen = now

			err := storage.RecordHostActivity(host, stats.Blocked, stats.BytesTransferred)
			if err != nil {
				logger.Log("Error saving stats for host %s: %v", host, err)
				continue
			}

			// Reset counters after saving
			stats.Connections = 0
			stats.BlockedAttempts = 0
			stats.BytesTransferred = 0
		}
	}
}

func (s *Server) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	// Get the original host
	host := r.Host
	if host == "" {
		host = r.URL.Host
	}

	// Clean the host
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	blocked := s.isBlocked(host)
	if blocked {
		logger.Log("BLOCKED HTTP: %s", host)
		s.updateStats(host, blocked, 0, false)
		http.Error(w, "Blocked", http.StatusForbidden)
		return
	}

	// Create a new request to forward
	outReq := new(http.Request)
	*outReq = *r // Copy the original request

	// Ensure we're not forwarding the original connection settings
	outReq.Close = false
	outReq.RequestURI = ""

	// Create a counting writer to track bytes
	countingWriter := &CountingWriter{ResponseWriter: w}

	// Create client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make the request
	resp, err := client.Do(outReq)
	if err != nil {
		fmt.Printf("Error proxying request: %v\n", err)
		http.Error(w, "Error proxying request", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy headers
	for k, v := range resp.Header {
		w.Header()[k] = v
	}

	// Set status code
	w.WriteHeader(resp.StatusCode)

	// Copy the response body
	written, err := io.Copy(countingWriter, resp.Body)
	if err != nil {
		fmt.Printf("Error copying response: %v\n", err)
		return
	}

	s.updateStats(host, blocked, uint64(written), true)
}

// CountingWriter to track response size
type CountingWriter struct {
	http.ResponseWriter
	BytesWritten uint64
}

func (w *CountingWriter) Write(bytes []byte) (int, error) {
	n, err := w.ResponseWriter.Write(bytes)
	w.BytesWritten += uint64(n)
	return n, err
}

func (s *Server) HandleHTTPS(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	blocked := s.isBlocked(host)

	s.updateStats(host, blocked, 0, true)

	if r.Method != "CONNECT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if blocked {
		logger.Log("BLOCKED HTTPS: %s", host)
		http.Error(w, "Blocked", http.StatusForbidden)
		return
	}

	destConn, err := net.DialTimeout("tcp", host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	go s.transfer(host, destConn, clientConn, true)
	go s.transfer(host, clientConn, destConn, false)
}

func (s *Server) transfer(host string, dest io.WriteCloser, src io.ReadCloser, logCall bool) {
	defer dest.Close()
	defer src.Close()
	writenBVytes, err := io.Copy(dest, src)
	if err != nil && logCall {
		s.updateStats(host, false, uint64(writenBVytes), false)
		return
	}

}

// Add method to update in-memory stats
func (s *Server) updateStats(host string, blocked bool, bytes uint64, incrementConnections bool) {
	// Extract host without port
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Record geolocation data asynchronously
	geo.RecordHostLocation(host)

	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()

	hostStats, exists := s.stats.HostStats[host]
	if !exists {
		// Resolve IPs for the host
		ips, err := net.LookupHost(host)
		ipList := "unknown"
		if err == nil && len(ips) > 0 {
			ipList = strings.Join(ips, ",")
		}

		hostStats = &stats.HostStats{
			Host:     host,
			IPs:      ipList,
			LastSeen: time.Now(),
		}
		s.stats.HostStats[host] = hostStats
	}

	if incrementConnections {
		hostStats.Connections++
	}

	hostStats.BytesTransferred += bytes
	if blocked {
		if incrementConnections {
			hostStats.BlockedAttempts++
		}
		hostStats.Blocked = true
	}
}

func (s *Server) startStatsMonitoring() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			storage.DisplayAllHostStats()
		}
	}()
}
