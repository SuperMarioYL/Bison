package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bison/api-server/pkg/logger"
)

const (
	ExportVersion    = "1.0"
	RedactedValue    = "***REDACTED***"
	SectionBilling   = "billing"
	SectionAlerts    = "alerts"
	SectionResources = "resources"
	SectionCP        = "controlPlane"
	SectionScripts   = "initScripts"
)

var AllSections = []string{SectionBilling, SectionAlerts, SectionResources, SectionCP, SectionScripts}

// ExportConfig represents the full export file structure
type ExportConfig struct {
	Version    string                 `json:"version"`
	ExportedAt time.Time             `json:"exportedAt"`
	ExportedBy string                 `json:"exportedBy"`
	Sections   map[string]json.RawMessage `json:"sections"`
}

// SectionPreview holds diff info for one config section
type SectionPreview struct {
	Present          bool                              `json:"present"`
	Valid            bool                              `json:"valid"`
	HasSensitiveData bool                              `json:"hasSensitiveData"`
	Changes          map[string]*FieldChange           `json:"changes,omitempty"`
	Summary          *ResourceSummary                  `json:"summary,omitempty"`
	Warnings         []string                          `json:"warnings,omitempty"`
	Errors           []string                          `json:"errors,omitempty"`
}

// FieldChange represents a single field change
type FieldChange struct {
	Current  interface{} `json:"current"`
	Imported interface{} `json:"imported"`
}

// ResourceSummary for array-based configs
type ResourceSummary struct {
	Added     []string `json:"added,omitempty"`
	Modified  []string `json:"modified,omitempty"`
	Removed   []string `json:"removed,omitempty"`
	Unchanged []string `json:"unchanged,omitempty"`
}

// ImportPreviewResult holds the preview/diff analysis
type ImportPreviewResult struct {
	Valid      bool                       `json:"valid"`
	Version    string                     `json:"version"`
	ExportedAt string                     `json:"exportedAt,omitempty"`
	Sections   map[string]*SectionPreview `json:"sections"`
	Errors     []string                   `json:"errors"`
	Warnings   []string                   `json:"warnings"`
}

// ImportRequest holds the import apply request
type ImportRequest struct {
	Config            ExportConfig `json:"config"`
	Sections          []string     `json:"sections"`
	PreserveSensitive bool         `json:"preserveSensitive"`
}

// ImportResult holds the import apply result
type ImportResult struct {
	Message  string   `json:"message"`
	Applied  []string `json:"applied"`
	Skipped  []string `json:"skipped"`
	Warnings []string `json:"warnings"`
}

// ConfigTransferService handles configuration export and import
type ConfigTransferService struct {
	billingSvc        *BillingService
	alertSvc          *AlertService
	resourceConfigSvc *ResourceConfigService
	initScriptSvc     *InitScriptService
}

// NewConfigTransferService creates a new ConfigTransferService
func NewConfigTransferService(
	billingSvc *BillingService,
	alertSvc *AlertService,
	resourceConfigSvc *ResourceConfigService,
	initScriptSvc *InitScriptService,
) *ConfigTransferService {
	return &ConfigTransferService{
		billingSvc:        billingSvc,
		alertSvc:          alertSvc,
		resourceConfigSvc: resourceConfigSvc,
		initScriptSvc:     initScriptSvc,
	}
}

// Export exports selected configuration sections
func (s *ConfigTransferService) Export(ctx context.Context, sections []string, includeSensitive bool, operator string) (*ExportConfig, error) {
	logger.Info("Exporting configuration", "sections", sections, "includeSensitive", includeSensitive, "operator", operator)

	sectionSet := make(map[string]bool)
	for _, sec := range sections {
		sectionSet[sec] = true
	}

	result := &ExportConfig{
		Version:    ExportVersion,
		ExportedAt: time.Now(),
		ExportedBy: operator,
		Sections:   make(map[string]json.RawMessage),
	}

	if sectionSet[SectionBilling] {
		config, err := s.billingSvc.GetConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to export billing config: %w", err)
		}
		data, _ := json.Marshal(config)
		result.Sections[SectionBilling] = data
	}

	if sectionSet[SectionAlerts] {
		config, err := s.alertSvc.GetConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to export alert config: %w", err)
		}
		if !includeSensitive {
			s.redactAlertChannels(config)
		}
		data, _ := json.Marshal(config)
		result.Sections[SectionAlerts] = data
	}

	if sectionSet[SectionResources] {
		configs, err := s.resourceConfigSvc.GetResourceConfigs(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to export resource configs: %w", err)
		}
		data, _ := json.Marshal(configs)
		result.Sections[SectionResources] = data
	}

	if sectionSet[SectionCP] {
		config, err := s.initScriptSvc.GetControlPlaneConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to export control plane config: %w", err)
		}
		if !includeSensitive {
			if config.Password != "" {
				config.Password = RedactedValue
			}
			if config.PrivateKey != "" {
				config.PrivateKey = RedactedValue
			}
		}
		data, _ := json.Marshal(config)
		result.Sections[SectionCP] = data
	}

	if sectionSet[SectionScripts] {
		groups, err := s.initScriptSvc.GetAllScriptGroups(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to export init scripts: %w", err)
		}
		data, _ := json.Marshal(groups)
		result.Sections[SectionScripts] = data
	}

	return result, nil
}

// redactAlertChannels masks sensitive webhook URLs in alert channels
func (s *ConfigTransferService) redactAlertChannels(config *AlertConfig) {
	sensitiveKeys := map[string]bool{
		"url":     true,
		"webhook": true,
		"smtp":    true,
	}
	for i := range config.Channels {
		for key := range config.Channels[i].Config {
			if sensitiveKeys[key] {
				val := config.Channels[i].Config[key]
				if len(val) > 20 {
					config.Channels[i].Config[key] = val[:10] + "***" + val[len(val)-5:]
				} else if val != "" {
					config.Channels[i].Config[key] = RedactedValue
				}
			}
		}
	}
}

// Preview validates and previews an import configuration
func (s *ConfigTransferService) Preview(ctx context.Context, config *ExportConfig) (*ImportPreviewResult, error) {
	logger.Info("Previewing configuration import")

	result := &ImportPreviewResult{
		Valid:    true,
		Version:  config.Version,
		Sections: make(map[string]*SectionPreview),
		Errors:   []string{},
		Warnings: []string{},
	}

	if config.Version != ExportVersion {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("不支持的版本: %s (期望 %s)", config.Version, ExportVersion))
		return result, nil
	}

	if !config.ExportedAt.IsZero() {
		result.ExportedAt = config.ExportedAt.Format(time.RFC3339)
	}

	for section, raw := range config.Sections {
		switch section {
		case SectionBilling:
			preview := s.previewBilling(ctx, raw)
			result.Sections[section] = preview
			if !preview.Valid {
				result.Valid = false
			}
		case SectionAlerts:
			preview := s.previewAlerts(ctx, raw)
			result.Sections[section] = preview
			if !preview.Valid {
				result.Valid = false
			}
		case SectionResources:
			preview := s.previewResources(ctx, raw)
			result.Sections[section] = preview
			if !preview.Valid {
				result.Valid = false
			}
		case SectionCP:
			preview := s.previewControlPlane(ctx, raw)
			result.Sections[section] = preview
			if !preview.Valid {
				result.Valid = false
			}
		case SectionScripts:
			preview := s.previewInitScripts(ctx, raw)
			result.Sections[section] = preview
			if !preview.Valid {
				result.Valid = false
			}
		default:
			result.Warnings = append(result.Warnings, fmt.Sprintf("未知的配置模块: %s (将被忽略)", section))
		}
	}

	return result, nil
}

func (s *ConfigTransferService) previewBilling(ctx context.Context, raw json.RawMessage) *SectionPreview {
	preview := &SectionPreview{Present: true, Valid: true}

	var imported BillingConfig
	if err := json.Unmarshal(raw, &imported); err != nil {
		preview.Valid = false
		preview.Errors = append(preview.Errors, "计费配置格式无效: "+err.Error())
		return preview
	}

	if imported.Interval <= 0 || imported.Interval > 24 {
		preview.Errors = append(preview.Errors, "计费间隔必须在 1-24 小时之间")
		preview.Valid = false
	}
	if imported.Currency == "" {
		preview.Errors = append(preview.Errors, "货币代码不能为空")
		preview.Valid = false
	}

	current, err := s.billingSvc.GetConfig(ctx)
	if err != nil {
		preview.Warnings = append(preview.Warnings, "无法获取当前计费配置进行对比")
		return preview
	}

	preview.Changes = make(map[string]*FieldChange)
	if current.Enabled != imported.Enabled {
		preview.Changes["enabled"] = &FieldChange{Current: current.Enabled, Imported: imported.Enabled}
	}
	if current.Interval != imported.Interval {
		preview.Changes["interval"] = &FieldChange{Current: current.Interval, Imported: imported.Interval}
	}
	if current.Currency != imported.Currency {
		preview.Changes["currency"] = &FieldChange{Current: current.Currency, Imported: imported.Currency}
	}
	if current.CurrencySymbol != imported.CurrencySymbol {
		preview.Changes["currencySymbol"] = &FieldChange{Current: current.CurrencySymbol, Imported: imported.CurrencySymbol}
	}
	if current.GracePeriodValue != imported.GracePeriodValue {
		preview.Changes["gracePeriodValue"] = &FieldChange{Current: current.GracePeriodValue, Imported: imported.GracePeriodValue}
	}
	if current.GracePeriodUnit != imported.GracePeriodUnit {
		preview.Changes["gracePeriodUnit"] = &FieldChange{Current: current.GracePeriodUnit, Imported: imported.GracePeriodUnit}
	}

	return preview
}

func (s *ConfigTransferService) previewAlerts(ctx context.Context, raw json.RawMessage) *SectionPreview {
	preview := &SectionPreview{Present: true, Valid: true}

	var imported AlertConfig
	if err := json.Unmarshal(raw, &imported); err != nil {
		preview.Valid = false
		preview.Errors = append(preview.Errors, "告警配置格式无效: "+err.Error())
		return preview
	}

	if imported.BalanceThreshold < 0 {
		preview.Errors = append(preview.Errors, "告警阈值不能为负数")
		preview.Valid = false
	}

	for _, ch := range imported.Channels {
		if ch.ID == "" || ch.Type == "" || ch.Name == "" {
			preview.Errors = append(preview.Errors, fmt.Sprintf("告警通道 '%s' 缺少必填字段 (id/type/name)", ch.Name))
			preview.Valid = false
		}
		for _, val := range ch.Config {
			if val == RedactedValue {
				preview.HasSensitiveData = true
				preview.Warnings = append(preview.Warnings, "告警通道包含已脱敏的敏感数据，导入时将保留当前值")
				break
			}
		}
	}

	current, err := s.alertSvc.GetConfig(ctx)
	if err != nil {
		preview.Warnings = append(preview.Warnings, "无法获取当前告警配置进行对比")
		return preview
	}

	preview.Changes = make(map[string]*FieldChange)
	if current.BalanceThreshold != imported.BalanceThreshold {
		preview.Changes["balanceThreshold"] = &FieldChange{Current: current.BalanceThreshold, Imported: imported.BalanceThreshold}
	}
	if len(current.Channels) != len(imported.Channels) {
		preview.Changes["channels"] = &FieldChange{
			Current:  fmt.Sprintf("%d 个通道", len(current.Channels)),
			Imported: fmt.Sprintf("%d 个通道", len(imported.Channels)),
		}
	}

	return preview
}

func (s *ConfigTransferService) previewResources(ctx context.Context, raw json.RawMessage) *SectionPreview {
	preview := &SectionPreview{Present: true, Valid: true}

	var imported []ResourceDefinition
	if err := json.Unmarshal(raw, &imported); err != nil {
		preview.Valid = false
		preview.Errors = append(preview.Errors, "资源配置格式无效: "+err.Error())
		return preview
	}

	for _, r := range imported {
		if r.Name == "" {
			preview.Errors = append(preview.Errors, "资源名称不能为空")
			preview.Valid = false
		}
		if r.Divisor <= 0 {
			preview.Errors = append(preview.Errors, fmt.Sprintf("资源 '%s' 的 divisor 必须大于 0", r.Name))
			preview.Valid = false
		}
	}

	current, err := s.resourceConfigSvc.GetResourceConfigs(ctx)
	if err != nil {
		preview.Warnings = append(preview.Warnings, "无法获取当前资源配置进行对比")
		return preview
	}

	currentMap := make(map[string]ResourceDefinition)
	for _, r := range current {
		currentMap[r.Name] = r
	}
	importedMap := make(map[string]ResourceDefinition)
	for _, r := range imported {
		importedMap[r.Name] = r
	}

	summary := &ResourceSummary{}
	for _, r := range imported {
		if _, exists := currentMap[r.Name]; exists {
			curR := currentMap[r.Name]
			if curR.DisplayName != r.DisplayName || curR.Unit != r.Unit || curR.Divisor != r.Divisor ||
				curR.Category != r.Category || curR.Enabled != r.Enabled || curR.Price != r.Price ||
				curR.SortOrder != r.SortOrder || curR.ShowInQuota != r.ShowInQuota {
				summary.Modified = append(summary.Modified, r.Name)
			} else {
				summary.Unchanged = append(summary.Unchanged, r.Name)
			}
		} else {
			summary.Added = append(summary.Added, r.Name)
		}
	}
	for _, r := range current {
		if _, exists := importedMap[r.Name]; !exists {
			summary.Removed = append(summary.Removed, r.Name)
		}
	}

	if len(summary.Removed) > 0 {
		preview.Warnings = append(preview.Warnings, fmt.Sprintf("以下资源将被移除: %v", summary.Removed))
	}

	preview.Summary = summary
	return preview
}

func (s *ConfigTransferService) previewControlPlane(ctx context.Context, raw json.RawMessage) *SectionPreview {
	preview := &SectionPreview{Present: true, Valid: true}

	var imported ControlPlaneConfig
	if err := json.Unmarshal(raw, &imported); err != nil {
		preview.Valid = false
		preview.Errors = append(preview.Errors, "控制面配置格式无效: "+err.Error())
		return preview
	}

	if imported.SSHPort < 1 || imported.SSHPort > 65535 {
		preview.Errors = append(preview.Errors, "SSH 端口必须在 1-65535 之间")
		preview.Valid = false
	}
	if imported.AuthMethod != "" && imported.AuthMethod != "password" && imported.AuthMethod != "privateKey" {
		preview.Errors = append(preview.Errors, "认证方式必须为 password 或 privateKey")
		preview.Valid = false
	}

	if imported.Password == RedactedValue || imported.PrivateKey == RedactedValue {
		preview.HasSensitiveData = true
		preview.Warnings = append(preview.Warnings, "敏感数据 (密码/私钥) 已被排除，导入时将保留当前值")
	}

	current, err := s.initScriptSvc.GetControlPlaneConfig(ctx)
	if err != nil {
		preview.Warnings = append(preview.Warnings, "无法获取当前控制面配置进行对比")
		return preview
	}

	preview.Changes = make(map[string]*FieldChange)
	if current.Host != imported.Host {
		preview.Changes["host"] = &FieldChange{Current: current.Host, Imported: imported.Host}
	}
	if current.SSHPort != imported.SSHPort {
		preview.Changes["sshPort"] = &FieldChange{Current: current.SSHPort, Imported: imported.SSHPort}
	}
	if current.SSHUser != imported.SSHUser {
		preview.Changes["sshUser"] = &FieldChange{Current: current.SSHUser, Imported: imported.SSHUser}
	}
	if current.AuthMethod != imported.AuthMethod {
		preview.Changes["authMethod"] = &FieldChange{Current: current.AuthMethod, Imported: imported.AuthMethod}
	}

	return preview
}

func (s *ConfigTransferService) previewInitScripts(ctx context.Context, raw json.RawMessage) *SectionPreview {
	preview := &SectionPreview{Present: true, Valid: true}

	var imported []ScriptGroup
	if err := json.Unmarshal(raw, &imported); err != nil {
		preview.Valid = false
		preview.Errors = append(preview.Errors, "初始化脚本配置格式无效: "+err.Error())
		return preview
	}

	for _, g := range imported {
		if g.ID == "" || g.Name == "" {
			preview.Errors = append(preview.Errors, fmt.Sprintf("脚本组 '%s' 缺少必填字段 (id/name)", g.Name))
			preview.Valid = false
		}
		if g.Phase != PhasePreJoin && g.Phase != PhasePostJoin {
			preview.Errors = append(preview.Errors, fmt.Sprintf("脚本组 '%s' 的 phase 必须为 pre-join 或 post-join", g.Name))
			preview.Valid = false
		}
	}

	current, err := s.initScriptSvc.GetAllScriptGroups(ctx)
	if err != nil {
		preview.Warnings = append(preview.Warnings, "无法获取当前初始化脚本进行对比")
		return preview
	}

	currentMap := make(map[string]ScriptGroup)
	for _, g := range current {
		currentMap[g.ID] = g
	}

	summary := &ResourceSummary{}
	for _, g := range imported {
		if _, exists := currentMap[g.ID]; exists {
			summary.Modified = append(summary.Modified, g.Name)
		} else {
			summary.Added = append(summary.Added, g.Name)
		}
	}
	importedMap := make(map[string]bool)
	for _, g := range imported {
		importedMap[g.ID] = true
	}
	for _, g := range current {
		if !importedMap[g.ID] {
			summary.Removed = append(summary.Removed, g.Name)
		}
	}

	builtinOverwrite := 0
	for _, g := range imported {
		if cur, exists := currentMap[g.ID]; exists && cur.Builtin {
			builtinOverwrite++
		}
	}
	if builtinOverwrite > 0 {
		preview.Warnings = append(preview.Warnings, fmt.Sprintf("将覆盖 %d 个内置脚本组", builtinOverwrite))
	}

	preview.Summary = summary
	return preview
}

// Apply applies the imported configuration
func (s *ConfigTransferService) Apply(ctx context.Context, req *ImportRequest) (*ImportResult, error) {
	logger.Info("Applying imported configuration", "sections", req.Sections)

	result := &ImportResult{
		Applied:  []string{},
		Skipped:  []string{},
		Warnings: []string{},
	}

	sectionSet := make(map[string]bool)
	for _, sec := range req.Sections {
		sectionSet[sec] = true
	}

	for _, section := range AllSections {
		raw, exists := req.Config.Sections[section]
		if !exists || !sectionSet[section] {
			if sectionSet[section] {
				result.Skipped = append(result.Skipped, section)
			}
			continue
		}

		var err error
		switch section {
		case SectionBilling:
			err = s.applyBilling(ctx, raw)
		case SectionAlerts:
			err = s.applyAlerts(ctx, raw, req.PreserveSensitive)
		case SectionResources:
			err = s.applyResources(ctx, raw)
		case SectionCP:
			err = s.applyControlPlane(ctx, raw, req.PreserveSensitive)
		case SectionScripts:
			err = s.applyInitScripts(ctx, raw)
		}

		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s 导入失败: %s", section, err.Error()))
			result.Skipped = append(result.Skipped, section)
		} else {
			result.Applied = append(result.Applied, section)
		}
	}

	if len(result.Applied) > 0 {
		result.Message = fmt.Sprintf("成功导入 %d 个配置模块", len(result.Applied))
	} else {
		result.Message = "未成功导入任何配置模块"
	}

	return result, nil
}

func (s *ConfigTransferService) applyBilling(ctx context.Context, raw json.RawMessage) error {
	var config BillingConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return fmt.Errorf("解析计费配置失败: %w", err)
	}
	return s.billingSvc.SetConfig(ctx, &config)
}

func (s *ConfigTransferService) applyAlerts(ctx context.Context, raw json.RawMessage, preserveSensitive bool) error {
	var config AlertConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return fmt.Errorf("解析告警配置失败: %w", err)
	}

	if preserveSensitive {
		current, err := s.alertSvc.GetConfig(ctx)
		if err == nil {
			currentChannelMap := make(map[string]NotifyChannel)
			for _, ch := range current.Channels {
				currentChannelMap[ch.ID] = ch
			}
			for i, ch := range config.Channels {
				if curCh, exists := currentChannelMap[ch.ID]; exists {
					for key, val := range ch.Config {
						if val == RedactedValue || (len(val) > 8 && val[len(val)-3:] == "***") {
							if curVal, ok := curCh.Config[key]; ok {
								config.Channels[i].Config[key] = curVal
							}
						}
					}
				}
			}
		}
	}

	return s.alertSvc.SetConfig(ctx, &config)
}

func (s *ConfigTransferService) applyResources(ctx context.Context, raw json.RawMessage) error {
	var configs []ResourceDefinition
	if err := json.Unmarshal(raw, &configs); err != nil {
		return fmt.Errorf("解析资源配置失败: %w", err)
	}
	return s.resourceConfigSvc.SaveResourceConfigs(ctx, configs)
}

func (s *ConfigTransferService) applyControlPlane(ctx context.Context, raw json.RawMessage, preserveSensitive bool) error {
	var config ControlPlaneConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return fmt.Errorf("解析控制面配置失败: %w", err)
	}

	if preserveSensitive {
		current, err := s.initScriptSvc.GetControlPlaneConfig(ctx)
		if err == nil {
			if config.Password == RedactedValue {
				config.Password = current.Password
			}
			if config.PrivateKey == RedactedValue {
				config.PrivateKey = current.PrivateKey
			}
		}
	}

	return s.initScriptSvc.SaveControlPlaneConfig(ctx, &config)
}

func (s *ConfigTransferService) applyInitScripts(ctx context.Context, raw json.RawMessage) error {
	var groups []ScriptGroup
	if err := json.Unmarshal(raw, &groups); err != nil {
		return fmt.Errorf("解析初始化脚本配置失败: %w", err)
	}
	return s.initScriptSvc.SaveAllScriptGroups(ctx, groups)
}
