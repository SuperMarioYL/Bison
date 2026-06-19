package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds the API server configuration
type Config struct {
	// Server settings
	Port int
	Mode string // "debug" or "release"

	// Auth settings
	AuthEnabled   bool
	AdminUsername string
	AdminPassword string
	JWTSecret     string

	// External services
	OpenCostURL   string
	PrometheusURL string

	// CORSAllowedOrigins restricts cross-origin requests. Empty means allow all
	// origins ("*"); set a comma-separated allowlist to tighten in production.
	CORSAllowedOrigins []string

	// Feature toggles
	CapsuleEnabled bool

	// LeaderElectionEnabled gates the singleton scheduler behind a Kubernetes
	// lease so it runs on exactly one replica. Disable for single-replica or
	// out-of-cluster development.
	LeaderElectionEnabled bool
}

// Built-in development defaults that MUST NOT be used in production once auth is
// enabled — startup refuses to proceed if they are left unchanged.
const (
	defaultAdminPassword = "admin"
	defaultJWTSecret     = "bison-secret-key-change-in-production"
)

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Port:                  8080,
		Mode:                  "release",
		AuthEnabled:           false,
		AdminUsername:         "admin",
		AdminPassword:         defaultAdminPassword,
		JWTSecret:             defaultJWTSecret,
		OpenCostURL:           "",
		PrometheusURL:         "",
		CapsuleEnabled:        true,
		LeaderElectionEnabled: true,
	}

	if port := os.Getenv("PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("invalid PORT: %v", err)
		}
		cfg.Port = p
	}

	if mode := os.Getenv("GIN_MODE"); mode != "" {
		cfg.Mode = mode
	}

	// Auth settings
	if enabled := os.Getenv("AUTH_ENABLED"); enabled == "true" {
		cfg.AuthEnabled = true
	}
	if username := os.Getenv("ADMIN_USERNAME"); username != "" {
		cfg.AdminUsername = username
	}
	if password := os.Getenv("ADMIN_PASSWORD"); password != "" {
		cfg.AdminPassword = password
	}
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.JWTSecret = secret
	}

	// External services
	if opencostURL := os.Getenv("OPENCOST_URL"); opencostURL != "" {
		cfg.OpenCostURL = opencostURL
	}
	if prometheusURL := os.Getenv("PROMETHEUS_URL"); prometheusURL != "" {
		cfg.PrometheusURL = prometheusURL
	}

	// Feature toggles
	if capsuleEnabled := os.Getenv("CAPSULE_ENABLED"); capsuleEnabled == "false" {
		cfg.CapsuleEnabled = false
	}
	if le := os.Getenv("LEADER_ELECTION_ENABLED"); le == "false" {
		cfg.LeaderElectionEnabled = false
	}

	// CORS allowlist (comma-separated origins). Empty -> allow all.
	if origins := os.Getenv("CORS_ALLOWED_ORIGINS"); origins != "" {
		for _, o := range strings.Split(origins, ",") {
			if o = strings.TrimSpace(o); o != "" {
				cfg.CORSAllowedOrigins = append(cfg.CORSAllowedOrigins, o)
			}
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate refuses to start with insecure defaults once authentication is enabled,
// so a production deployment cannot accidentally run with the public default JWT
// signing key or the well-known "admin" password.
func (c *Config) validate() error {
	if !c.AuthEnabled {
		return nil
	}
	if c.JWTSecret == "" || c.JWTSecret == defaultJWTSecret {
		return fmt.Errorf("refusing to start: JWT_SECRET must be set to a non-default value when AUTH_ENABLED=true")
	}
	if c.AdminPassword == "" || c.AdminPassword == defaultAdminPassword {
		return fmt.Errorf("refusing to start: ADMIN_PASSWORD must be set to a non-default value when AUTH_ENABLED=true")
	}
	return nil
}
