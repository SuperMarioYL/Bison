package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/pkg/logger"
)

const (
	AlertConfigConfigMap  = "bison-alert-config"
	AlertHistoryConfigMap = "bison-alert-history"
	MaxAlertHistory       = 1000
)

// AlertConfig represents alert configuration
type AlertConfig struct {
	BalanceThreshold float64          `json:"balanceThreshold"` // Alert when balance below this
	Channels         []NotifyChannel  `json:"channels"`
}

// NotifyChannel represents a notification channel
type NotifyChannel struct {
	ID      string            `json:"id"`
	Type    string            `json:"type"`    // email, webhook, dingtalk, wechat
	Name    string            `json:"name"`
	Config  map[string]string `json:"config"`  // Channel-specific config
	Enabled bool              `json:"enabled"`
}

// Alert represents an alert instance
type Alert struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`    // low_balance, suspended, etc.
	Severity  string    `json:"severity"` // warning, critical
	Target    string    `json:"target"`   // Team name
	Message   string    `json:"message"`
	Sent      bool      `json:"sent"`
	SentAt    time.Time `json:"sentAt,omitempty"`
	Channels  []string  `json:"channels,omitempty"` // Channels alert was sent to
}

// AlertService handles alert operations
type AlertService struct {
	k8sClient  *k8s.Client
	balanceSvc *BalanceService
	httpClient *http.Client
}

// NewAlertService creates a new AlertService
func NewAlertService(k8sClient *k8s.Client, balanceSvc *BalanceService) *AlertService {
	return &AlertService{
		k8sClient:  k8sClient,
		balanceSvc: balanceSvc,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetConfig returns the alert configuration
func (s *AlertService) GetConfig(ctx context.Context) (*AlertConfig, error) {
	logger.Debug("Getting alert config")

	cm, err := s.k8sClient.GetConfigMap(ctx, BisonNamespace, AlertConfigConfigMap)
	if err != nil {
		return s.getDefaultConfig(), nil
	}

	data, ok := cm.Data["config"]
	if !ok {
		return s.getDefaultConfig(), nil
	}

	var config AlertConfig
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		logger.Error("Failed to unmarshal alert config", "error", err)
		return s.getDefaultConfig(), nil
	}

	return &config, nil
}

// SetConfig sets the alert configuration
func (s *AlertService) SetConfig(ctx context.Context, config *AlertConfig) error {
	logger.Info("Setting alert config")

	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	cm, err := s.k8sClient.GetConfigMap(ctx, BisonNamespace, AlertConfigConfigMap)
	if err != nil {
		// Create if not exists
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      AlertConfigConfigMap,
				Namespace: BisonNamespace,
				Labels: map[string]string{
					"app.kubernetes.io/name":      "bison",
					"app.kubernetes.io/component": "alert",
				},
			},
			Data: map[string]string{
				"config": string(data),
			},
		}
		return s.k8sClient.CreateConfigMap(ctx, BisonNamespace, cm)
	}

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data["config"] = string(data)

	return s.k8sClient.UpdateConfigMap(ctx, BisonNamespace, cm)
}

// CheckAndNotify checks for alert conditions and sends notifications
func (s *AlertService) CheckAndNotify(ctx context.Context) error {
	logger.Debug("Checking alert conditions")

	config, err := s.GetConfig(ctx)
	if err != nil {
		return err
	}

	if s.balanceSvc == nil {
		return nil
	}

	// Check for low balance teams
	lowBalanceTeams, err := s.balanceSvc.GetLowBalanceTeams(ctx, config.BalanceThreshold)
	if err != nil {
		logger.Error("Failed to get low balance teams", "error", err)
		return err
	}

	for _, balance := range lowBalanceTeams {
		alert := &Alert{
			ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
			Timestamp: time.Now(),
			Type:      "low_balance",
			Severity:  "warning",
			Target:    balance.TeamName,
			Message:   fmt.Sprintf("Team %s balance is low: %.2f", balance.TeamName, balance.Amount),
		}

		if balance.Amount < 0 {
			alert.Severity = "critical"
			alert.Type = "negative_balance"
			alert.Message = fmt.Sprintf("Team %s has negative balance: %.2f", balance.TeamName, balance.Amount)
		}

		if err := s.SendAlert(ctx, config, alert); err != nil {
			logger.Error("Failed to send alert", "team", balance.TeamName, "error", err)
		}
	}

	return nil
}

// SendAlert sends an alert through configured channels
func (s *AlertService) SendAlert(ctx context.Context, config *AlertConfig, alert *Alert) error {
	logger.Info("Sending alert", "type", alert.Type, "target", alert.Target)

	var sentChannels []string
	for _, channel := range config.Channels {
		if !channel.Enabled {
			continue
		}

		if err := s.sendToChannel(ctx, &channel, alert); err != nil {
			logger.Error("Failed to send alert to channel", "channel", channel.Name, "error", err)
		} else {
			sentChannels = append(sentChannels, channel.Name)
		}
	}

	alert.Sent = len(sentChannels) > 0
	alert.SentAt = time.Now()
	alert.Channels = sentChannels

	// Record alert history
	return s.recordAlert(ctx, alert)
}

// TestChannel tests a notification channel
func (s *AlertService) TestChannel(ctx context.Context, channel *NotifyChannel) error {
	logger.Info("Testing notification channel", "type", channel.Type, "name", channel.Name)

	alert := &Alert{
		ID:        "test",
		Timestamp: time.Now(),
		Type:      "test",
		Severity:  "info",
		Target:    "test",
		Message:   "This is a test notification from Bison",
	}

	return s.sendToChannel(ctx, channel, alert)
}

// GetHistory returns alert history
func (s *AlertService) GetHistory(ctx context.Context, limit int) ([]*Alert, error) {
	logger.Debug("Getting alert history", "limit", limit)

	cm, err := s.k8sClient.GetConfigMap(ctx, BisonNamespace, AlertHistoryConfigMap)
	if err != nil {
		return []*Alert{}, nil
	}

	data, ok := cm.Data["history"]
	if !ok {
		return []*Alert{}, nil
	}

	var alerts []*Alert
	if err := json.Unmarshal([]byte(data), &alerts); err != nil {
		logger.Error("Failed to unmarshal alert history", "error", err)
		return []*Alert{}, nil
	}

	// Sort by timestamp descending
	sort.Slice(alerts, func(i, j int) bool {
		return alerts[i].Timestamp.After(alerts[j].Timestamp)
	})

	if limit > 0 && len(alerts) > limit {
		alerts = alerts[:limit]
	}

	return alerts, nil
}

// Helper methods

func (s *AlertService) getDefaultConfig() *AlertConfig {
	return &AlertConfig{
		BalanceThreshold: 100,
		Channels:         []NotifyChannel{},
	}
}

func (s *AlertService) sendToChannel(ctx context.Context, channel *NotifyChannel, alert *Alert) error {
	switch channel.Type {
	case "webhook":
		return s.sendWebhook(ctx, channel, alert)
	case "dingtalk":
		return s.sendDingtalk(ctx, channel, alert)
	case "wechat":
		return s.sendWechat(ctx, channel, alert)
	case "email":
		return s.sendEmail(ctx, channel, alert)
	default:
		return fmt.Errorf("unknown channel type: %s", channel.Type)
	}
}

func (s *AlertService) sendWebhook(ctx context.Context, channel *NotifyChannel, alert *Alert) error {
	url := channel.Config["url"]
	if url == "" {
		return fmt.Errorf("webhook url not configured")
	}

	payload := map[string]interface{}{
		"type":      alert.Type,
		"severity":  alert.Severity,
		"target":    alert.Target,
		"message":   alert.Message,
		"timestamp": alert.Timestamp,
	}

	data, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (s *AlertService) sendDingtalk(ctx context.Context, channel *NotifyChannel, alert *Alert) error {
	url := channel.Config["webhook"]
	if url == "" {
		return fmt.Errorf("dingtalk webhook not configured")
	}

	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": fmt.Sprintf("[%s] %s\n%s", alert.Severity, alert.Type, alert.Message),
		},
	}

	data, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dingtalk returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (s *AlertService) sendWechat(ctx context.Context, channel *NotifyChannel, alert *Alert) error {
	url := channel.Config["webhook"]
	if url == "" {
		return fmt.Errorf("wechat webhook not configured")
	}

	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": fmt.Sprintf("[%s] %s\n%s", alert.Severity, alert.Type, alert.Message),
		},
	}

	data, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("wechat returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (s *AlertService) sendEmail(ctx context.Context, channel *NotifyChannel, alert *Alert) error {
	// Email sending requires SMTP configuration
	// For now, just log
	logger.Info("Email alert would be sent", "to", channel.Config["to"], "message", alert.Message)
	return nil
}

func (s *AlertService) recordAlert(ctx context.Context, alert *Alert) error {
	cm, err := s.k8sClient.GetConfigMap(ctx, BisonNamespace, AlertHistoryConfigMap)
	if err != nil {
		// Create if not exists
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      AlertHistoryConfigMap,
				Namespace: BisonNamespace,
				Labels: map[string]string{
					"app.kubernetes.io/name":      "bison",
					"app.kubernetes.io/component": "alert",
				},
			},
			Data: map[string]string{
				"history": "[]",
			},
		}
		if err := s.k8sClient.CreateConfigMap(ctx, BisonNamespace, cm); err != nil {
			return err
		}
	}

	var alerts []*Alert
	if data, ok := cm.Data["history"]; ok {
		json.Unmarshal([]byte(data), &alerts)
	}

	alerts = append(alerts, alert)
	if len(alerts) > MaxAlertHistory {
		alerts = alerts[len(alerts)-MaxAlertHistory:]
	}

	data, _ := json.Marshal(alerts)
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data["history"] = string(data)

	return s.k8sClient.UpdateConfigMap(ctx, BisonNamespace, cm)
}

