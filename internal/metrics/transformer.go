package metrics

import "go-proxy/internal/stats"

type MetricPoint struct {
	Timestamp  int64   `json:"timestamp"`
	Value      float64 `json:"value"`
	Host       string  `json:"host"`
	MetricType string  `json:"type"`
}

func TransformHostStats(stats map[string]stats.HostStats) []MetricPoint {
	var metrics []MetricPoint

	for _, stat := range stats {
		timestamp := stat.LastSeen.Unix() * 1000 // Grafana expects milliseconds

		// Add different metrics
		metrics = append(metrics, MetricPoint{
			Timestamp:  timestamp,
			Value:      float64(stat.Connections),
			Host:       stat.Host,
			MetricType: "connections",
		})

		metrics = append(metrics, MetricPoint{
			Timestamp:  timestamp,
			Value:      float64(stat.BytesTransferred),
			Host:       stat.Host,
			MetricType: "bytes",
		})

		metrics = append(metrics, MetricPoint{
			Timestamp:  timestamp,
			Value:      float64(stat.BlockedAttempts),
			Host:       stat.Host,
			MetricType: "blocked",
		})
	}

	return metrics
}
