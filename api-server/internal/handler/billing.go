package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/bison/api-server/internal/service"
	"github.com/bison/api-server/pkg/logger"
)

// BillingHandler handles billing-related requests
type BillingHandler struct {
	billingSvc *service.BillingService
	balanceSvc *service.BalanceService
}

// NewBillingHandler creates a new BillingHandler
func NewBillingHandler(billingSvc *service.BillingService, balanceSvc *service.BalanceService) *BillingHandler {
	return &BillingHandler{
		billingSvc: billingSvc,
		balanceSvc: balanceSvc,
	}
}

// GetBillingConfig returns the billing configuration
func (h *BillingHandler) GetBillingConfig(c *gin.Context) {
	config, err := h.billingSvc.GetConfig(c.Request.Context())
	if err != nil {
		logger.Error("Failed to get billing config", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, config)
}

// UpdateBillingConfig updates the billing configuration
func (h *BillingHandler) UpdateBillingConfig(c *gin.Context) {
	var config service.BillingConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.billingSvc.SetConfig(c.Request.Context(), &config); err != nil {
		logger.Error("Failed to update billing config", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "config updated"})
}

// GetTeamBalance returns the balance for a team
func (h *BillingHandler) GetTeamBalance(c *gin.Context) {
	teamName := c.Param("name")

	balance, err := h.balanceSvc.GetBalance(c.Request.Context(), teamName)
	if err != nil {
		logger.Error("Failed to get balance", "team", teamName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, balance)
}

// RechargeTeam recharges a team's balance
func (h *BillingHandler) RechargeTeam(c *gin.Context) {
	teamName := c.Param("name")

	var req struct {
		Amount   float64 `json:"amount" binding:"required,gt=0"`
		Remark   string  `json:"remark"`
		Operator string  `json:"operator"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Operator == "" {
		req.Operator = "admin" // Default operator
	}

	if err := h.balanceSvc.Recharge(c.Request.Context(), teamName, req.Amount, req.Operator, req.Remark); err != nil {
		logger.Error("Failed to recharge", "team", teamName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "recharged successfully"})
}

// GetRechargeHistory returns recharge history for a team
func (h *BillingHandler) GetRechargeHistory(c *gin.Context) {
	teamName := c.Param("name")
	limit := 50 // Default limit

	history, err := h.balanceSvc.GetRechargeHistory(c.Request.Context(), teamName, limit)
	if err != nil {
		logger.Error("Failed to get recharge history", "team", teamName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": history})
}

// GetTeamBill returns a bill for a team
func (h *BillingHandler) GetTeamBill(c *gin.Context) {
	teamName := c.Param("name")
	window := c.DefaultQuery("window", "7d")

	bill, err := h.billingSvc.GetTeamBill(c.Request.Context(), teamName, window)
	if err != nil {
		logger.Error("Failed to get team bill", "team", teamName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, bill)
}

// GetAutoRechargeConfig returns auto-recharge configuration for a team
func (h *BillingHandler) GetAutoRechargeConfig(c *gin.Context) {
	teamName := c.Param("name")

	config, err := h.balanceSvc.GetAutoRechargeConfig(c.Request.Context(), teamName)
	if err != nil {
		logger.Error("Failed to get auto-recharge config", "team", teamName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateAutoRechargeConfig updates auto-recharge configuration for a team
func (h *BillingHandler) UpdateAutoRechargeConfig(c *gin.Context) {
	teamName := c.Param("name")

	var config service.AutoRechargeConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.balanceSvc.SetAutoRechargeConfig(c.Request.Context(), teamName, &config); err != nil {
		logger.Error("Failed to update auto-recharge config", "team", teamName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "config updated"})
}

// SuspendTeam suspends a team
func (h *BillingHandler) SuspendTeam(c *gin.Context) {
	teamName := c.Param("name")

	if err := h.billingSvc.SuspendTeam(c.Request.Context(), teamName); err != nil {
		logger.Error("Failed to suspend team", "team", teamName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "team suspended"})
}

// ResumeTeam resumes a suspended team
func (h *BillingHandler) ResumeTeam(c *gin.Context) {
	teamName := c.Param("name")

	if err := h.billingSvc.ResumeTeam(c.Request.Context(), teamName); err != nil {
		logger.Error("Failed to resume team", "team", teamName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "team resumed"})
}
