package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/bison/api-server/internal/config"
	"github.com/bison/api-server/internal/handler"
	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/internal/middleware"
	"github.com/bison/api-server/internal/opencost"
	"github.com/bison/api-server/internal/scheduler"
	"github.com/bison/api-server/internal/service"
	"github.com/bison/api-server/pkg/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		os.Stderr.WriteString("Failed to load config: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Initialize logger
	debug := cfg.Mode != "release"
	logger.Init(debug)
	defer logger.Sync()

	logger.Info("Starting Bison API Server",
		"port", cfg.Port,
		"mode", cfg.Mode,
		"auth_enabled", cfg.AuthEnabled,
		"opencost_url", cfg.OpenCostURL,
		"prometheus_url", cfg.PrometheusURL,
	)

	// Initialize Kubernetes client
	k8sClient, err := k8s.NewClient()
	if err != nil {
		logger.Fatal("Failed to create k8s client", "error", err)
	}
	logger.Info("Kubernetes client initialized")

	// Initialize OpenCost client
	var opencostClient *opencost.Client
	if cfg.OpenCostURL != "" {
		opencostClient = opencost.NewClient(cfg.OpenCostURL)
		logger.Info("OpenCost client initialized", "url", cfg.OpenCostURL)
	}

	// Initialize services
	resourceConfigSvc := service.NewResourceConfigService(k8sClient)
	resourceSvc := service.NewResourceService(k8sClient, resourceConfigSvc)
	tenantSvc := service.NewTenantService(k8sClient)
	projectSvc := service.NewProjectService(k8sClient)
	costSvc := service.NewCostService(cfg.OpenCostURL, k8sClient)
	settingsSvc := service.NewSettingsService(cfg.PrometheusURL, cfg.OpenCostURL)
	balanceSvc := service.NewBalanceService(k8sClient)
	userSvc := service.NewUserService(k8sClient, opencostClient)
	auditSvc := service.NewAuditService(k8sClient)
	alertSvc := service.NewAlertService(k8sClient, balanceSvc)
	billingSvc := service.NewBillingService(k8sClient, opencostClient, balanceSvc, tenantSvc, projectSvc, resourceConfigSvc)
	reportSvc := service.NewReportService(opencostClient, tenantSvc, projectSvc, billingSvc)
	nodeSvc := service.NewNodeService(k8sClient)
	workloadSvc := service.NewWorkloadService(k8sClient)
	initScriptSvc := service.NewInitScriptService(k8sClient)
	onboardingSvc := service.NewOnboardingService(k8sClient, nodeSvc, initScriptSvc)
	configTransferSvc := service.NewConfigTransferService(billingSvc, alertSvc, resourceConfigSvc, initScriptSvc)

	// Initialize scheduler
	sched := scheduler.NewScheduler(billingSvc, balanceSvc, alertSvc)

	// Initialize status service (needs scheduler)
	statusSvc := service.NewStatusService(
		k8sClient,
		opencostClient,
		sched,
		tenantSvc,
		projectSvc,
		userSvc,
		balanceSvc,
		cfg.PrometheusURL,
	)

	logger.Info("Services initialized")

	// Initialize handlers
	authHandler := handler.NewAuthHandler(cfg.AdminUsername, cfg.AdminPassword, cfg.JWTSecret, cfg.AuthEnabled)
	resourceHandler := handler.NewResourceHandler(resourceSvc)
	resourceConfigHandler := handler.NewResourceConfigHandler(resourceConfigSvc)
	teamHandler := handler.NewTeamHandler(tenantSvc, costSvc, nodeSvc)
	projectHandler := handler.NewProjectHandler(projectSvc, costSvc, resourceConfigSvc)
	statsHandler := handler.NewStatsHandler(k8sClient, tenantSvc, projectSvc, costSvc, resourceSvc, nodeSvc)
	settingsHandler := handler.NewSettingsHandler(settingsSvc)
	clusterHandler := handler.NewClusterHandler(k8sClient)
	billingHandler := handler.NewBillingHandler(billingSvc, balanceSvc)
	userHandler := handler.NewUserHandler(userSvc, tenantSvc, projectSvc)
	auditHandler := handler.NewAuditHandler(auditSvc)
	alertHandler := handler.NewAlertHandler(alertSvc)
	reportHandler := handler.NewReportHandler(reportSvc)
	statusHandler := handler.NewStatusHandler(statusSvc)
	nodeHandler := handler.NewNodeHandler(nodeSvc)
	workloadHandler := handler.NewWorkloadHandler(workloadSvc, projectSvc)
	onboardingHandler := handler.NewOnboardingHandler(onboardingSvc, initScriptSvc)
	configTransferHandler := handler.NewConfigTransferHandler(configTransferSvc)

	// Setup Gin router
	if cfg.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.Logger())
	router.Use(corsMiddleware())

	// Health check endpoints
	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	router.GET("/readyz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Auth endpoints (public)
		api.POST("/auth/login", authHandler.Login)
		api.GET("/auth/status", authHandler.GetAuthStatus)

		// Feature flags (public)
		api.GET("/features", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"costEnabled":      costSvc.IsEnabled(),
				"capsuleEnabled":   cfg.CapsuleEnabled,
				"prometheusEnabled": cfg.PrometheusURL != "",
			})
		})

		// Protected routes
		protected := api.Group("")
		protected.Use(authHandler.AuthMiddleware())
		{
			// Cluster resources (dynamic)
			protected.GET("/cluster/resources", resourceHandler.GetClusterResources)

			// Resource configuration
			protected.GET("/resource-configs", resourceConfigHandler.ListResourceConfigs)
			protected.GET("/resource-configs/enabled", resourceConfigHandler.GetEnabledResourceConfigs)
			protected.GET("/resource-configs/quota", resourceConfigHandler.GetQuotaResourceConfigs)
			protected.GET("/resource-configs/discover", resourceConfigHandler.DiscoverClusterResources)
			protected.POST("/resource-configs", resourceConfigHandler.AddResourceConfig)
			protected.PUT("/resource-configs", resourceConfigHandler.SaveResourceConfigs)
			protected.GET("/resource-configs/:name", resourceConfigHandler.GetResourceConfig)
			protected.PUT("/resource-configs/:name", resourceConfigHandler.UpdateResourceConfig)

			// Team management (Capsule Tenants)
			protected.GET("/teams", teamHandler.ListTeams)
			protected.GET("/teams/:name", teamHandler.GetTeam)
			protected.POST("/teams", teamHandler.CreateTeam)
			protected.PUT("/teams/:name", teamHandler.UpdateTeam)
			protected.DELETE("/teams/:name", teamHandler.DeleteTeam)

			// Team billing
			protected.GET("/teams/:name/balance", billingHandler.GetTeamBalance)
			protected.POST("/teams/:name/recharge", billingHandler.RechargeTeam)
			protected.GET("/teams/:name/balance/history", billingHandler.GetRechargeHistory)
			protected.GET("/teams/:name/bill", billingHandler.GetTeamBill)
			protected.GET("/teams/:name/auto-recharge", billingHandler.GetAutoRechargeConfig)
			protected.PUT("/teams/:name/auto-recharge", billingHandler.UpdateAutoRechargeConfig)
			protected.POST("/teams/:name/suspend", billingHandler.SuspendTeam)
			protected.POST("/teams/:name/resume", billingHandler.ResumeTeam)

			// Project management (Namespaces)
			protected.GET("/projects", projectHandler.ListProjects)
			protected.GET("/projects/:name", projectHandler.GetProject)
			protected.POST("/projects", projectHandler.CreateProject)
			protected.PUT("/projects/:name", projectHandler.UpdateProject)
			protected.DELETE("/projects/:name", projectHandler.DeleteProject)
			protected.GET("/projects/:name/usage", projectHandler.GetProjectUsage)

			// Project workloads
			protected.GET("/projects/:name/workloads", workloadHandler.ListWorkloads)
			protected.GET("/projects/:name/workloads/summary", workloadHandler.GetWorkloadSummary)

			// User management
			protected.GET("/users", userHandler.ListUsers)
			protected.POST("/users", userHandler.CreateUser)
			protected.GET("/users/:email", userHandler.GetUser)
			protected.PUT("/users/:email", userHandler.UpdateUser)
			protected.DELETE("/users/:email", userHandler.DeleteUser)
			protected.PUT("/users/:email/status", userHandler.SetUserStatus)
			protected.GET("/users/:email/usage", userHandler.GetUserUsage)
			protected.POST("/users/:email/teams", userHandler.AddUserToTeam)
			protected.DELETE("/users/:email/teams/:teamName", userHandler.RemoveUserFromTeam)
			protected.POST("/users/:email/projects", userHandler.AddUserToProject)
			protected.DELETE("/users/:email/projects/:projectName", userHandler.RemoveUserFromProject)
			protected.PUT("/users/:email/projects/:projectName/role", userHandler.UpdateUserProjectRole)

			// Statistics (OpenCost)
			protected.GET("/stats/overview", statsHandler.GetOverview)
			protected.GET("/stats/cost-status", statsHandler.GetCostStatus)
			protected.GET("/stats/usage/teams", statsHandler.GetTeamUsage)
			protected.GET("/stats/usage/projects", statsHandler.GetProjectUsage)
			protected.GET("/stats/usage/users", statsHandler.GetUserUsage)
			protected.GET("/stats/quota-alerts", statsHandler.GetQuotaAlerts)
			protected.GET("/stats/cost-trend", statsHandler.GetCostTrend)
			protected.GET("/stats/top-consumers", statsHandler.GetTopConsumers)

			// Reports
			protected.GET("/reports/team/:name", reportHandler.GetTeamReport)
			protected.GET("/reports/team/:name/export", reportHandler.ExportTeamReport)
			protected.GET("/reports/project/:name", reportHandler.GetProjectReport)
			protected.GET("/reports/project/:name/export", reportHandler.ExportProjectReport)
			protected.GET("/reports/summary", reportHandler.GetSummaryReport)
			protected.GET("/reports/summary/export", reportHandler.ExportSummaryReport)

			// Cluster info (legacy)
			protected.GET("/cluster/nodes", clusterHandler.ListNodes)
			protected.GET("/cluster/nodes/:name", clusterHandler.GetNode)
			protected.GET("/cluster/nodes/:name/pods", clusterHandler.GetNodePods)
			protected.PUT("/cluster/nodes/:name/labels", clusterHandler.UpdateNodeLabels)
			protected.PUT("/cluster/nodes/:name/taints", clusterHandler.UpdateNodeTaints)

			// Node management (with Bison status)
			protected.GET("/nodes", nodeHandler.ListNodes)
			protected.GET("/nodes/summary", nodeHandler.GetNodeStatusSummary)
			protected.GET("/nodes/shared", nodeHandler.GetSharedNodes)
			protected.GET("/nodes/team/:team", nodeHandler.GetTeamNodes)
			protected.GET("/nodes/:name", nodeHandler.GetNode)
			protected.POST("/nodes/:name/enable", nodeHandler.EnableNode)
			protected.POST("/nodes/:name/disable", nodeHandler.DisableNode)
			protected.POST("/nodes/:name/assign", nodeHandler.AssignNodeToTeam)
			protected.POST("/nodes/:name/release", nodeHandler.ReleaseNode)

			// Node onboarding
			protected.POST("/nodes/onboard", onboardingHandler.StartOnboarding)
			protected.GET("/nodes/onboard", onboardingHandler.ListOnboardingJobs)
			protected.GET("/nodes/onboard/:jobId", onboardingHandler.GetOnboardingJob)
			protected.DELETE("/nodes/onboard/:jobId", onboardingHandler.CancelOnboardingJob)

			// System settings
			protected.GET("/settings", settingsHandler.GetSettings)
			protected.GET("/settings/billing", billingHandler.GetBillingConfig)
			protected.PUT("/settings/billing", billingHandler.UpdateBillingConfig)
			protected.GET("/settings/alerts", alertHandler.GetAlertConfig)
			protected.PUT("/settings/alerts", alertHandler.UpdateAlertConfig)
			protected.POST("/settings/alerts/test", alertHandler.TestChannel)

			// Control plane settings
			protected.GET("/settings/control-plane", onboardingHandler.GetControlPlaneConfig)
			protected.PUT("/settings/control-plane", onboardingHandler.UpdateControlPlaneConfig)
			protected.POST("/settings/control-plane/test", onboardingHandler.TestControlPlaneConnection)

			// Init scripts settings
			protected.GET("/settings/init-scripts", onboardingHandler.ListInitScripts)
			protected.POST("/settings/init-scripts", onboardingHandler.CreateInitScript)
			protected.GET("/settings/init-scripts/:id", onboardingHandler.GetInitScript)
			protected.PUT("/settings/init-scripts/:id", onboardingHandler.UpdateInitScript)
			protected.DELETE("/settings/init-scripts/:id", onboardingHandler.DeleteInitScript)
			protected.PUT("/settings/init-scripts/:id/toggle", onboardingHandler.ToggleInitScript)
			protected.PUT("/settings/init-scripts/reorder", onboardingHandler.ReorderInitScripts)

			// Configuration import/export
			protected.GET("/settings/export", configTransferHandler.ExportConfig)
			protected.POST("/settings/import/preview", configTransferHandler.PreviewImport)
			protected.POST("/settings/import/apply", configTransferHandler.ApplyImport)

			// Node metrics (from Prometheus)
			protected.GET("/metrics/node/:name", settingsHandler.GetNodeMetrics)

			// Audit logs
			protected.GET("/audit/logs", auditHandler.ListLogs)
			protected.GET("/audit/recent", auditHandler.GetRecentLogs)

			// Alerts
			protected.GET("/alerts/history", alertHandler.GetAlertHistory)

			// System status
			protected.GET("/system/status", statusHandler.GetStatus)
			protected.GET("/system/tasks", statusHandler.GetTaskHistory)
		}
	}

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start scheduler
	ctx, cancel := context.WithCancel(context.Background())
	sched.Start(ctx)

	// Start server in goroutine
	go func() {
		logger.Info("API server started", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server error", "error", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("Shutting down server", "signal", sig.String())

	// Stop scheduler
	cancel()
	sched.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("Server stopped gracefully")
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
