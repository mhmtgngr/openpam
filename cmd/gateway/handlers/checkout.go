package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/pam/checkout"
	"github.com/openpam/openpam/pkg/validation"
	"github.com/rs/zerolog"
)

// CheckoutHandler handles credential checkout requests
type CheckoutHandler struct {
	service *checkout.Service
	logger  zerolog.Logger
}

// NewCheckoutHandler creates a new checkout handler
func NewCheckoutHandler(service *checkout.Service, logger zerolog.Logger) *CheckoutHandler {
	return &CheckoutHandler{service: service, logger: logger}
}

// List returns all active checkouts
func (h *CheckoutHandler) List(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	checkouts, err := h.service.ListActiveCheckouts(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list checkouts")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list checkouts"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": checkouts, "total": len(checkouts)})
}

// Create creates a new checkout request
func (h *CheckoutHandler) Create(c *gin.Context) {
	var req struct {
		CredentialID     string `json:"credential_id" binding:"required"`
		Justification    string `json:"justification" binding:"required"`
		Duration         int    `json:"duration_minutes" binding:"required"`
		IsBreakGlass     bool   `json:"is_break_glass"`
		BreakGlassReason string `json:"break_glass_reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	// Validate using Chain of Responsibility pattern
	chain := validation.NewChain().
		Add(validation.ValidUUID{Field: "credential_id", Value: req.CredentialID}).
		Add(validation.RequiredString{Field: "justification", Value: req.Justification}).
		Add(validation.MinLength{Field: "justification", Value: req.Justification, Min: 10}).
		Add(validation.IntRange{Field: "duration_minutes", Value: req.Duration, Min: 1, Max: 480})

	if req.IsBreakGlass {
		chain.Add(validation.RequiredString{Field: "break_glass_reason", Value: req.BreakGlassReason})
		chain.Add(validation.MinLength{Field: "break_glass_reason", Value: req.BreakGlassReason, Min: 20})
	}

	if errs := chain.Run(); errs.HasErrors() {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": errs.Error(), "details": errs}})
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))
	credentialID, _ := uuid.Parse(req.CredentialID)

	result, err := h.service.RequestCheckout(c.Request.Context(), userID, credentialID, req.Justification, req.Duration, req.IsBreakGlass, req.BreakGlassReason)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create checkout")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create checkout request"}})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// Checkout checks out a credential (returns the actual secret)
func (h *CheckoutHandler) Checkout(c *gin.Context) {
	checkoutID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid checkout ID"}})
		return
	}

	secret, err := h.service.CheckoutCredential(c.Request.Context(), checkoutID)
	if err != nil {
		h.logger.Error().Err(err).Str("checkout_id", checkoutID.String()).Msg("Failed to checkout credential")
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "CHECKOUT_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"credential": secret})
}

// Checkin checks in a credential
func (h *CheckoutHandler) Checkin(c *gin.Context) {
	checkoutID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid checkout ID"}})
		return
	}

	if err := h.service.CheckinCredential(c.Request.Context(), checkoutID); err != nil {
		h.logger.Error().Err(err).Str("checkout_id", checkoutID.String()).Msg("Failed to checkin credential")
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "CHECKIN_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Credential checked in successfully"})
}

func getIntParam(c *gin.Context, key string, defaultVal int) int {
	if v := c.Query(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}
