package config

import (
	"flag"
	"fmt"
)

type Config struct {
	HTTPPort      int
	HTTPSPort     int
	LogFile       string
	BlockFile     string
	RedisAddr     string
	RedisPassword string
	GeoEnabled    bool // Whether geolocation is enabled
	GeoCacheSize  int  // Size of in-memory geolocation cache
	GeoDebug      bool // Whether to enable verbose geolocation logging
}

func ParseFlags() *Config {
	cfg := &Config{}

	flag.IntVar(&cfg.HTTPPort, "http-port", 3000, "HTTP proxy port")
	flag.IntVar(&cfg.HTTPSPort, "https-port", 3443, "HTTPS proxy port")
	flag.StringVar(&cfg.LogFile, "log-file", "proxy.log", "Log file path")
	flag.StringVar(&cfg.BlockFile, "blacklist", "", "File containing blacklisted domain patterns")
	flag.StringVar(&cfg.RedisAddr, "redis-addr", "localhost:6379", "Redis address")
	flag.StringVar(&cfg.RedisPassword, "redis-password", "xK9mP2vL5nQ8", "Redis password")
	flag.BoolVar(&cfg.GeoEnabled, "geo-enabled", true, "Enable geolocation tracking")
	flag.IntVar(&cfg.GeoCacheSize, "geo-cache-size", 10000, "Size of in-memory geolocation cache")
	flag.BoolVar(&cfg.GeoDebug, "geo-debug", false, "Enable verbose geolocation logging")

	flag.Parse()
	return cfg
}

func (c *Config) HTTPAddr() string {
	return fmt.Sprintf(":%d", c.HTTPPort)
}

func (c *Config) HTTPSAddr() string {
	return fmt.Sprintf(":%d", c.HTTPSPort)
}
