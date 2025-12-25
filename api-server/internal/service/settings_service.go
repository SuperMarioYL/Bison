package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Settings represents system settings (read-only, configured via Helm)
type Settings struct {
	PrometheusURL string `json:"prometheusUrl"`
	OpenCostURL   string `json:"opencostUrl"`
}

// SettingsService handles system settings
type SettingsService struct {
	prometheusURL string
	opencostURL   string
}

// NewSettingsService creates a new SettingsService with config from environment
func NewSettingsService(prometheusURL, opencostURL string) *SettingsService {
	return &SettingsService{
		prometheusURL: prometheusURL,
		opencostURL:   opencostURL,
	}
}

// GetSettings returns current settings (read-only)
func (s *SettingsService) GetSettings() Settings {
	return Settings{
		PrometheusURL: s.prometheusURL,
		OpenCostURL:   s.opencostURL,
	}
}

// GetPrometheusURL returns the configured Prometheus URL
func (s *SettingsService) GetPrometheusURL() string {
	return s.prometheusURL
}

// PrometheusMetric represents a Prometheus metric data point
type PrometheusMetric struct {
	Timestamp float64 `json:"timestamp"`
	Value     float64 `json:"value"`
}

// NodeMetrics represents metrics for a node
type NodeMetrics struct {
	CPUUsage    []PrometheusMetric `json:"cpuUsage"`
	MemoryUsage []PrometheusMetric `json:"memoryUsage"`
}

// QueryPrometheus queries Prometheus API
func (s *SettingsService) QueryPrometheus(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]PrometheusMetric, error) {
	if s.prometheusURL == "" {
		return nil, fmt.Errorf("prometheus URL not configured")
	}

	// Build query URL
	url := fmt.Sprintf("%s/api/v1/query_range?query=%s&start=%d&end=%d&step=%d",
		s.prometheusURL,
		query,
		start.Unix(),
		end.Unix(),
		int(step.Seconds()),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query prometheus: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prometheus returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Values [][]interface{}   `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed")
	}

	var metrics []PrometheusMetric
	if len(result.Data.Result) > 0 {
		for _, v := range result.Data.Result[0].Values {
			if len(v) >= 2 {
				ts, _ := v[0].(float64)
				val := 0.0
				switch vv := v[1].(type) {
				case string:
					fmt.Sscanf(vv, "%f", &val)
				case float64:
					val = vv
				}
				metrics = append(metrics, PrometheusMetric{
					Timestamp: ts,
					Value:     val,
				})
			}
		}
	}

	return metrics, nil
}

// GetNodeMetrics returns metrics for a specific node
func (s *SettingsService) GetNodeMetrics(ctx context.Context, nodeName string, hours int) (*NodeMetrics, error) {
	end := time.Now()
	start := end.Add(-time.Duration(hours) * time.Hour)
	step := time.Minute * 5

	// Query CPU usage
	cpuQuery := fmt.Sprintf(`100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle", instance=~"%s.*"}[5m])) * 100)`, nodeName)
	cpuMetrics, err := s.QueryPrometheus(ctx, cpuQuery, start, end, step)
	if err != nil {
		cpuMetrics = nil // Non-fatal, continue
	}

	// Query memory usage
	memQuery := fmt.Sprintf(`(1 - (node_memory_MemAvailable_bytes{instance=~"%s.*"} / node_memory_MemTotal_bytes{instance=~"%s.*"})) * 100`, nodeName, nodeName)
	memMetrics, err := s.QueryPrometheus(ctx, memQuery, start, end, step)
	if err != nil {
		memMetrics = nil // Non-fatal, continue
	}

	return &NodeMetrics{
		CPUUsage:    cpuMetrics,
		MemoryUsage: memMetrics,
	}, nil
}
