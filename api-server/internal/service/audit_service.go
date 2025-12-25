package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/pkg/logger"
)

const (
	AuditLogsConfigMap = "bison-audit-logs"
	MaxAuditLogs       = 10000
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Operator  string                 `json:"operator"`
	Action    string                 `json:"action"`   // create, update, delete, recharge, suspend, resume, etc.
	Resource  string                 `json:"resource"` // team, project, user, config, etc.
	Target    string                 `json:"target"`   // Resource name
	Detail    map[string]interface{} `json:"detail,omitempty"`
	IP        string                 `json:"ip,omitempty"`
	UserAgent string                 `json:"userAgent,omitempty"`
}

// AuditFilter represents filter options for audit logs
type AuditFilter struct {
	Action   string    `json:"action,omitempty"`
	Resource string    `json:"resource,omitempty"`
	Operator string    `json:"operator,omitempty"`
	Target   string    `json:"target,omitempty"`
	From     time.Time `json:"from,omitempty"`
	To       time.Time `json:"to,omitempty"`
}

// AuditPage represents a paginated list of audit logs
type AuditPage struct {
	Items      []*AuditLog `json:"items"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalPages int         `json:"totalPages"`
}

// AuditService handles audit logging
type AuditService struct {
	k8sClient *k8s.Client
}

// NewAuditService creates a new AuditService
func NewAuditService(k8sClient *k8s.Client) *AuditService {
	return &AuditService{
		k8sClient: k8sClient,
	}
}

// Log records an audit log entry
func (s *AuditService) Log(ctx context.Context, log *AuditLog) error {
	logger.Debug("Recording audit log", "action", log.Action, "resource", log.Resource, "target", log.Target)

	if log.ID == "" {
		log.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	cm, err := s.getOrCreateConfigMap(ctx)
	if err != nil {
		return err
	}

	// Get existing logs
	var logs []*AuditLog
	if data, ok := cm.Data["logs"]; ok {
		if err := json.Unmarshal([]byte(data), &logs); err != nil {
			logger.Warn("Failed to unmarshal existing audit logs, starting fresh")
			logs = []*AuditLog{}
		}
	}

	// Add new log
	logs = append(logs, log)

	// Keep only last MaxAuditLogs
	if len(logs) > MaxAuditLogs {
		logs = logs[len(logs)-MaxAuditLogs:]
	}

	// Save back
	data, err := json.Marshal(logs)
	if err != nil {
		return fmt.Errorf("failed to marshal logs: %w", err)
	}

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data["logs"] = string(data)

	return s.k8sClient.UpdateConfigMap(ctx, BisonNamespace, cm)
}

// Query queries audit logs with filters and pagination
func (s *AuditService) Query(ctx context.Context, filter *AuditFilter, page, pageSize int) (*AuditPage, error) {
	logger.Debug("Querying audit logs", "filter", filter, "page", page, "pageSize", pageSize)

	cm, err := s.getOrCreateConfigMap(ctx)
	if err != nil {
		return nil, err
	}

	var logs []*AuditLog
	if data, ok := cm.Data["logs"]; ok {
		if err := json.Unmarshal([]byte(data), &logs); err != nil {
			logger.Error("Failed to unmarshal audit logs", "error", err)
			return &AuditPage{Items: []*AuditLog{}, Total: 0}, nil
		}
	}

	// Apply filters
	var filtered []*AuditLog
	for _, log := range logs {
		if s.matchesFilter(log, filter) {
			filtered = append(filtered, log)
		}
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.After(filtered[j].Timestamp)
	})

	// Apply pagination
	total := len(filtered)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	totalPages := (total + pageSize - 1) / pageSize

	return &AuditPage{
		Items:      filtered[start:end],
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetRecent returns the most recent audit logs
func (s *AuditService) GetRecent(ctx context.Context, limit int) ([]*AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}

	page, err := s.Query(ctx, nil, 1, limit)
	if err != nil {
		return nil, err
	}

	return page.Items, nil
}

// LogAction is a convenience method to log an action
func (s *AuditService) LogAction(ctx context.Context, operator, action, resource, target string, detail map[string]interface{}) {
	log := &AuditLog{
		Operator: operator,
		Action:   action,
		Resource: resource,
		Target:   target,
		Detail:   detail,
	}

	if err := s.Log(ctx, log); err != nil {
		logger.Error("Failed to record audit log", "error", err)
	}
}

// Helper methods

func (s *AuditService) matchesFilter(log *AuditLog, filter *AuditFilter) bool {
	if filter == nil {
		return true
	}

	if filter.Action != "" && log.Action != filter.Action {
		return false
	}
	if filter.Resource != "" && log.Resource != filter.Resource {
		return false
	}
	if filter.Operator != "" && log.Operator != filter.Operator {
		return false
	}
	if filter.Target != "" && log.Target != filter.Target {
		return false
	}
	if !filter.From.IsZero() && log.Timestamp.Before(filter.From) {
		return false
	}
	if !filter.To.IsZero() && log.Timestamp.After(filter.To) {
		return false
	}

	return true
}

func (s *AuditService) getOrCreateConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	cm, err := s.k8sClient.GetConfigMap(ctx, BisonNamespace, AuditLogsConfigMap)
	if err != nil {
		// Create if not exists
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      AuditLogsConfigMap,
				Namespace: BisonNamespace,
				Labels: map[string]string{
					"app.kubernetes.io/name":      "bison",
					"app.kubernetes.io/component": "audit",
				},
			},
			Data: map[string]string{
				"logs": "[]",
			},
		}
		if err := s.k8sClient.CreateConfigMap(ctx, BisonNamespace, cm); err != nil {
			return nil, fmt.Errorf("failed to create configmap: %w", err)
		}
	}

	return cm, nil
}

