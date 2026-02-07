package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// LabeledMetricSeries represents a Prometheus metric series with labels
type LabeledMetricSeries struct {
	Labels  map[string]string  `json:"labels"`
	Metrics []PrometheusMetric `json:"metrics"`
}

// NodeMetrics represents metrics for a node
type NodeMetrics struct {
	CPUUsage    []PrometheusMetric `json:"cpuUsage"`
	MemoryUsage []PrometheusMetric `json:"memoryUsage"`
	// Network IO
	NetworkReceive  []PrometheusMetric `json:"networkReceive,omitempty"`
	NetworkTransmit []PrometheusMetric `json:"networkTransmit,omitempty"`
	// RDMA IO
	RdmaReceive  []PrometheusMetric `json:"rdmaReceive,omitempty"`
	RdmaTransmit []PrometheusMetric `json:"rdmaTransmit,omitempty"`
	// GPU (NVIDIA DCGM)
	GpuUtilization []PrometheusMetric    `json:"gpuUtilization,omitempty"`
	GpuMemoryUtil  []PrometheusMetric    `json:"gpuMemoryUtil,omitempty"`
	GpuPerDevice   []LabeledMetricSeries `json:"gpuPerDevice,omitempty"`
	// NPU (Huawei Ascend)
	NpuUtilization []PrometheusMetric `json:"npuUtilization,omitempty"`
	NpuMemoryUtil  []PrometheusMetric `json:"npuMemoryUtil,omitempty"`
	NpuTemperature []PrometheusMetric `json:"npuTemperature,omitempty"`
}

// NodeMetricsRequest holds parameters for querying node metrics
type NodeMetricsRequest struct {
	NodeName string
	Hours    int
	HasGpu   bool
	HasNpu   bool
}

// prometheusResponse is the JSON structure returned by Prometheus query_range API
type prometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Values [][]interface{}   `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

// queryPrometheusRaw executes a Prometheus range query and returns the raw response
func (s *SettingsService) queryPrometheusRaw(ctx context.Context, query string, start, end time.Time, step time.Duration) (*prometheusResponse, error) {
	if s.prometheusURL == "" {
		return nil, fmt.Errorf("prometheus URL not configured")
	}

	params := url.Values{}
	params.Set("query", query)
	params.Set("start", fmt.Sprintf("%d", start.Unix()))
	params.Set("end", fmt.Sprintf("%d", end.Unix()))
	params.Set("step", fmt.Sprintf("%d", int(step.Seconds())))
	fullURL := fmt.Sprintf("%s/api/v1/query_range?%s", s.prometheusURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
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

	var result prometheusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed")
	}

	return &result, nil
}

// parseMetricValues extracts PrometheusMetric slice from raw Prometheus values
func parseMetricValues(values [][]interface{}) []PrometheusMetric {
	var metrics []PrometheusMetric
	for _, v := range values {
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
	return metrics
}

// QueryPrometheus queries Prometheus API and returns the first result series
func (s *SettingsService) QueryPrometheus(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]PrometheusMetric, error) {
	result, err := s.queryPrometheusRaw(ctx, query, start, end, step)
	if err != nil {
		return nil, err
	}

	if len(result.Data.Result) > 0 {
		return parseMetricValues(result.Data.Result[0].Values), nil
	}

	return nil, nil
}

// QueryPrometheusMultiSeries queries Prometheus API and returns all result series with labels
func (s *SettingsService) QueryPrometheusMultiSeries(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]LabeledMetricSeries, error) {
	result, err := s.queryPrometheusRaw(ctx, query, start, end, step)
	if err != nil {
		return nil, err
	}

	var series []LabeledMetricSeries
	for _, r := range result.Data.Result {
		series = append(series, LabeledMetricSeries{
			Labels:  r.Metric,
			Metrics: parseMetricValues(r.Values),
		})
	}

	return series, nil
}

// GetNodeMetrics returns metrics for a specific node
func (s *SettingsService) GetNodeMetrics(ctx context.Context, req NodeMetricsRequest) (*NodeMetrics, error) {
	end := time.Now()
	start := end.Add(-time.Duration(req.Hours) * time.Hour)
	step := time.Minute * 5
	node := req.NodeName

	result := &NodeMetrics{}

	// --- Always query: CPU, Memory, Network, RDMA ---

	// CPU usage (%)
	cpuQuery := fmt.Sprintf(`100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle", instance=~"%s.*"}[5m])) * 100)`, node)
	result.CPUUsage, _ = s.QueryPrometheus(ctx, cpuQuery, start, end, step)

	// Memory usage (%)
	memQuery := fmt.Sprintf(`(1 - (node_memory_MemAvailable_bytes{instance=~"%s.*"} / node_memory_MemTotal_bytes{instance=~"%s.*"})) * 100`, node, node)
	result.MemoryUsage, _ = s.QueryPrometheus(ctx, memQuery, start, end, step)

	// Network receive (bytes/sec, excluding virtual interfaces)
	netRecvQuery := fmt.Sprintf(`sum(rate(node_network_receive_bytes_total{instance=~"%s.*",device!~"lo|docker.*|veth.*|br.*|cni.*|flannel.*|cali.*|tunl.*|kube.*|virbr.*"}[5m]))`, node)
	result.NetworkReceive, _ = s.QueryPrometheus(ctx, netRecvQuery, start, end, step)

	// Network transmit (bytes/sec)
	netTransQuery := fmt.Sprintf(`sum(rate(node_network_transmit_bytes_total{instance=~"%s.*",device!~"lo|docker.*|veth.*|br.*|cni.*|flannel.*|cali.*|tunl.*|kube.*|virbr.*"}[5m]))`, node)
	result.NetworkTransmit, _ = s.QueryPrometheus(ctx, netTransQuery, start, end, step)

	// RDMA receive (bytes/sec, InfiniBand via node_exporter)
	rdmaRecvQuery := fmt.Sprintf(`sum(rate(node_infiniband_port_data_received_bytes_total{instance=~"%s.*"}[5m]))`, node)
	result.RdmaReceive, _ = s.QueryPrometheus(ctx, rdmaRecvQuery, start, end, step)

	// RDMA transmit (bytes/sec)
	rdmaTransQuery := fmt.Sprintf(`sum(rate(node_infiniband_port_data_transmitted_bytes_total{instance=~"%s.*"}[5m]))`, node)
	result.RdmaTransmit, _ = s.QueryPrometheus(ctx, rdmaTransQuery, start, end, step)

	// --- Conditional: GPU (DCGM) ---
	if req.HasGpu {
		// Average GPU SM utilization (%)
		gpuUtilQuery := fmt.Sprintf(`avg(DCGM_FI_DEV_GPU_UTIL{Hostname="%s"} or DCGM_FI_DEV_GPU_UTIL{instance=~"%s.*"})`, node, node)
		result.GpuUtilization, _ = s.QueryPrometheus(ctx, gpuUtilQuery, start, end, step)

		// Average GPU memory utilization (%)
		gpuMemQuery := fmt.Sprintf(`avg(DCGM_FI_DEV_MEM_COPY_UTIL{Hostname="%s"} or DCGM_FI_DEV_MEM_COPY_UTIL{instance=~"%s.*"})`, node, node)
		result.GpuMemoryUtil, _ = s.QueryPrometheus(ctx, gpuMemQuery, start, end, step)

		// Per-GPU SM utilization (multi-series)
		gpuPerDeviceQuery := fmt.Sprintf(`DCGM_FI_DEV_GPU_UTIL{Hostname="%s"} or DCGM_FI_DEV_GPU_UTIL{instance=~"%s.*"}`, node, node)
		result.GpuPerDevice, _ = s.QueryPrometheusMultiSeries(ctx, gpuPerDeviceQuery, start, end, step)
	}

	// --- Conditional: NPU (Huawei Ascend) ---
	if req.HasNpu {
		// NPU utilization (%)
		npuUtilQuery := fmt.Sprintf(`avg(npu_chip_info_utilization{id=~"%s.*"})`, node)
		result.NpuUtilization, _ = s.QueryPrometheus(ctx, npuUtilQuery, start, end, step)

		// NPU HBM usage (%)
		npuMemQuery := fmt.Sprintf(`avg(npu_chip_info_hbm_usage{id=~"%s.*"})`, node)
		result.NpuMemoryUtil, _ = s.QueryPrometheus(ctx, npuMemQuery, start, end, step)

		// NPU temperature (°C)
		npuTempQuery := fmt.Sprintf(`avg(npu_chip_info_temperature{id=~"%s.*"})`, node)
		result.NpuTemperature, _ = s.QueryPrometheus(ctx, npuTempQuery, start, end, step)
	}

	return result, nil
}
