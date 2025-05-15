package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go-proxy/internal/api"
	"go-proxy/internal/config"
	"go-proxy/internal/geo"
	"go-proxy/internal/logger"
	"go-proxy/internal/proxy"
	"go-proxy/internal/storage"
)

func main() {
	cfg := config.ParseFlags()

	// Print startup banner and configuration
	fmt.Printf("\n=== Proxy Server Configuration ===\n")
	fmt.Printf("🌐 HTTP Proxy: http://localhost:%d\n", cfg.HTTPPort)
	fmt.Printf("🔒 HTTPS Proxy: https://localhost:%d\n", cfg.HTTPSPort)
	fmt.Printf("📝 Log File: %s\n", cfg.LogFile)
	fmt.Printf("📊 Redis Address: %s\n", cfg.RedisAddr)
	fmt.Printf("🚫 Blacklist File: %s\n", cfg.BlockFile)
	fmt.Printf("🌍 Geolocation Enabled: %t\n", cfg.GeoEnabled)
	if cfg.GeoEnabled {
		fmt.Printf("🧠 Geolocation Cache Size: %d entries\n", cfg.GeoCacheSize)
	}
	fmt.Printf("===============================\n\n")

	// Initialize logger
	if err := logger.Init(cfg.LogFile); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ Logger initialized\n")

	// Initialize Redis
	if err := storage.InitRedis(cfg.RedisAddr, cfg.RedisPassword); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ Redis connection established\n")

	// Initialize proxy server
	proxyServer := proxy.NewServer(cfg)

	// Initialize API handlers
	apiHandler := api.NewHandler()

	// Create HTTP server mux
	httpMux := http.NewServeMux()

	// Register API routes
	httpMux.HandleFunc("/api/stats/daily", apiHandler.HandleDailyStats)
	httpMux.HandleFunc("/api/stats/hourly", apiHandler.HandleHourlyStats)
	httpMux.HandleFunc("/api/metrics", apiHandler.HandleMetrics)

	// Initialize geolocation system if enabled
	if cfg.GeoEnabled {
		if err := geo.Initialize(cfg.RedisAddr, cfg.GeoCacheSize); err != nil {
			log.Printf("⚠️ Warning: Geolocation system initialization failed: %v\n", err)
		} else {
			fmt.Printf("✅ Geolocation system initialized\n")

			// Add geolocation API endpoint
			geo.AddAPIHandler(httpMux)
		}
	} else {
		fmt.Printf("ℹ️ Geolocation tracking disabled\n")
	}

	// Register proxy handler
	httpMux.HandleFunc("/", proxyServer.HandleHTTP)

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: httpMux,
	}

	// Start HTTPS server
	httpsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPSPort),
		Handler: proxyServer, // This handles CONNECT requests for HTTPS
	}

	// Start both servers
	fmt.Printf("\n🚀 Starting proxy servers...\n")
	fmt.Printf("📡 HTTP proxy listening on http://localhost:%d\n", cfg.HTTPPort)
	fmt.Printf("📡 HTTPS proxy listening on https://localhost:%d\n", cfg.HTTPSPort)
	fmt.Printf("🌐 API endpoints available at http://localhost:%d/api/*\n", cfg.HTTPPort)
	fmt.Printf("\n💡 Configure your browser/system proxy settings to:\n")
	fmt.Printf("   HTTP Proxy:  localhost:%d\n", cfg.HTTPPort)
	fmt.Printf("   HTTPS Proxy: localhost:%d\n", cfg.HTTPSPort)
	fmt.Printf("\n📊 Statistics API endpoints:\n")
	fmt.Printf("   Daily stats:  http://localhost:%d/api/stats/daily\n", cfg.HTTPPort)
	fmt.Printf("   Hourly stats: http://localhost:%d/api/stats/hourly\n", cfg.HTTPPort)
	fmt.Printf("   Metrics:      http://localhost:%d/api/metrics\n", cfg.HTTPPort)
	fmt.Printf("   Geolocation:  http://localhost:%d/api/geo\n", cfg.HTTPPort)
	fmt.Printf("\n✨ Proxy server is ready!\n")

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server in a goroutine
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v\n", err)
		}
	}()

	// Start HTTPS server in a goroutine
	go func() {
		if err := httpsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTPS server error: %v\n", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\n🛑 Shutting down servers...")

	// Clean up resources
	if cfg.GeoEnabled {
		geo.Shutdown()
	}

	// Exit
	os.Exit(0)
}
