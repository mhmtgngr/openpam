package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/auth"
	"github.com/rs/zerolog"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	auth   *auth.Service
	logger zerolog.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *auth.Service, logger zerolog.Logger) *AuthHandler {
	return &AuthHandler{
		auth:   authService,
		logger: logger,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	MFACode  string `json:"mfa_code,omitempty"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	AccessToken      string       `json:"access_token"`
	RefreshToken     string       `json:"refresh_token"`
	TokenType        string       `json:"token_type"`
	ExpiresIn        int64        `json:"expires_in"`
	User             *auth.User   `json:"user"`
	MFARequired      bool         `json:"mfa_required"`
	MFASetupRequired bool         `json:"mfa_setup_required"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// MFASetupRequest represents MFA setup request
type MFASetupRequest struct {
	Code string `json:"code" binding:"required"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=12"`
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_INPUT",
				"message": "Invalid request: " + err.Error(),
			},
		})
		return
	}

	// Get tenant from context (set by middleware)
	tenantID, _ := c.Get("tenant_id")
	if tenantID == nil {
		// Try to get from domain or use default
		tenantID = "default"
	}

	// Authenticate
	resp, err := h.auth.Authenticate(c.Request.Context(), req.Email, req.Password, tenantID.(string), c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		requestID, _ := c.Get("request_id")
		h.logger.Warn().
			Str("request_id", requestID.(string)).
			Str("email", req.Email).
			Str("ip", c.ClientIP()).
			Err(err).
			Msg("Failed login attempt")

		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "AUTH_FAILED",
				"message": "Invalid email or password",
			},
		})
		return
	}

	// If MFA is required but not provided, return 202 indicating MFA verification needed
	if resp.MFARequired && req.MFACode == "" {
		c.JSON(http.StatusAccepted, gin.H{
			"message":          "MFA verification required",
			"mfa_required":     true,
			"mfa_setup_required": resp.MFASetupRequired,
			"temp_token":       resp.AccessToken, // Temporary token for MFA verification
		})
		return
	}

	// If MFA is required and code is provided, verify it
	if resp.MFARequired && req.MFACode != "" {
		resp, err = h.auth.AuthenticateMFA(c.Request.Context(), resp.User.ID, req.MFACode, resp.User.TenantID.String())
		if err != nil {
			h.logger.Warn().
				Str("user_id", resp.User.ID.String()).
				Msg("Failed MFA verification")

			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "MFA_FAILED",
					"message": "Invalid MFA code",
				},
			})
			return
		}
	}

	// Log successful login
	requestID, _ := c.Get("request_id")
	h.logger.Info().
		Str("request_id", requestID.(string)).
		Str("user_id", resp.User.ID.String()).
		Str("email", resp.User.Email).
		Str("ip", c.ClientIP()).
		Msg("User logged in successfully")

	c.JSON(http.StatusOK, resp)
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = c.ShouldBindJSON(&req)

	userID, _ := c.Get("user_id")

	if req.RefreshToken != "" {
		if err := h.auth.Logout(c.Request.Context(), req.RefreshToken, userID.(string)); err != nil {
			h.logger.Error().Err(err).Msg("Failed to logout")
		}
	}

	// Clear session from cache
	// In production, you'd invalidate the access token in Redis

	h.logger.Info().
		Str("user_id", userID.(string)).
		Msg("User logged out")

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// Refresh handles token refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_INPUT",
				"message": "Refresh token is required",
			},
		})
		return
	}

	resp, err := h.auth.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "INVALID_TOKEN",
				"message": "Invalid or expired refresh token",
			},
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Me returns the current user info
func (h *AuthHandler) Me(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDUUID, _ := uuid.Parse(userID.(string))

	user, err := h.auth.GetUser(c.Request.Context(), userIDUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "User not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// SetupMFA initiates MFA setup
func (h *AuthHandler) SetupMFA(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDUUID, _ := uuid.Parse(userID.(string))

	secret, qrCode, err := h.auth.SetupMFA(c.Request.Context(), userIDUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "MFA_SETUP_FAILED",
				"message": "Failed to initiate MFA setup",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"secret": secret,
		"qr_code": qrCode,
	})
}

// VerifyAndEnableMFA verifies MFA code and enables it
func (h *AuthHandler) VerifyAndEnableMFA(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDUUID, _ := uuid.Parse(userID.(string))

	var req MFASetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_INPUT",
				"message": "Invalid request",
			},
		})
		return
	}

	if err := h.auth.VerifyAndEnableMFA(c.Request.Context(), userIDUUID, req.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_MFA_CODE",
				"message": "Invalid MFA verification code",
			},
		})
		return
	}

	h.logger.Info().
		Str("user_id", userIDUUID.String()).
		Msg("MFA enabled for user")

	c.JSON(http.StatusOK, gin.H{"message": "MFA enabled successfully"})
}

// ChangePassword handles password change
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDUUID, _ := uuid.Parse(userID.(string))

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_INPUT",
				"message": "Invalid request",
			},
		})
		return
	}

	if err := h.auth.ChangePassword(c.Request.Context(), userIDUUID, req.CurrentPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "PASSWORD_CHANGE_FAILED",
				"message": "Failed to change password. Please verify your current password.",
			},
		})
		return
	}

	h.logger.Info().
		Str("user_id", userIDUUID.String()).
		Msg("Password changed successfully")

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// VerifyMFA handles standalone MFA verification
func (h *AuthHandler) VerifyMFA(c *gin.Context) {
	var req struct {
		TempToken string `json:"temp_token" binding:"required"`
		MFACode   string `json:"mfa_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_INPUT",
				"message": "Invalid request",
			},
		})
		return
	}

	// Validate temp token and get user ID
	claims, err := h.auth.ValidateToken(req.TempToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "INVALID_TOKEN",
				"message": "Invalid temporary token",
			},
		})
		return
	}

	userIDUUID, _ := uuid.Parse(claims.UserID)

	// Verify MFA
	resp, err := h.auth.AuthenticateMFA(c.Request.Context(), userIDUUID, req.MFACode, claims.TenantID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "MFA_FAILED",
				"message": "Invalid MFA code",
			},
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Password  string `json:"password" binding:"required,min=12"`
	Role      string `json:"role"`
}

// Register handles new user registration (tenant admins only)
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_INPUT",
				"message": "Invalid request: " + err.Error(),
			},
		})
		return
	}

	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, _ := uuid.Parse(tenantID.(string))

	// Set default role
	if req.Role == "" {
		req.Role = "user"
	}

	user := &auth.User{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      req.Role,
		Status:    "active",
		TenantID:  tenantIDUUID,
	}

	if err := h.auth.CreateUser(c.Request.Context(), user, req.Password); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "USER_CREATE_FAILED",
				"message": "Failed to create user",
			},
		})
		return
	}

	h.logger.Info().
		Str("user_id", user.ID.String()).
		Str("email", user.Email).
		Str("role", user.Role).
		Msg("User created")

	c.JSON(http.StatusCreated, gin.H{"user": user})
}

// ListUsers returns a list of users in the tenant
func (h *AuthHandler) ListUsers(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, _ := uuid.Parse(tenantID.(string))

	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	users, total, err := h.auth.ListUsers(c.Request.Context(), tenantIDUUID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "LIST_USERS_FAILED",
				"message": "Failed to list users",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": total,
		"limit": limit,
		"offset": offset,
	})
}

// UpdateUser updates a user
func (h *AuthHandler) UpdateUser(c *gin.Context) {
	targetUserID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_USER_ID",
				"message": "Invalid user ID",
			},
		})
		return
	}

	var req struct {
		FirstName *string `json:"first_name"`
		LastName  *string `json:"last_name"`
		Role      *string `json:"role"`
		Status    *string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_INPUT",
				"message": "Invalid request",
			},
		})
		return
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.FirstName != nil {
		updates["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		updates["last_name"] = *req.LastName
	}
	if req.Role != nil {
		updates["role"] = *req.Role
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if err := h.auth.UpdateUser(c.Request.Context(), targetUserID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "UPDATE_USER_FAILED",
				"message": "Failed to update user",
			},
		})
		return
	}

	h.logger.Info().
		Str("target_user_id", targetUserID.String()).
		Msg("User updated")

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// DeleteUser deletes (soft deletes) a user
func (h *AuthHandler) DeleteUser(c *gin.Context) {
	targetUserID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_USER_ID",
				"message": "Invalid user ID",
			},
		})
		return
	}

	requestorUserID, _ := c.Get("user_id")
	requestorID, _ := uuid.Parse(requestorUserID.(string))

	// Prevent self-deletion
	if targetUserID == requestorID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "CANNOT_DELETE_SELF",
				"message": "Cannot delete your own account",
			},
		})
		return
	}

	if err := h.auth.DeleteUser(c.Request.Context(), targetUserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "DELETE_USER_FAILED",
				"message": "Failed to delete user",
			},
		})
		return
	}

	h.logger.Info().
		Str("target_user_id", targetUserID.String()).
		Str("deleted_by", requestorID.String()).
		Msg("User deleted")

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// GetBackupCodes generates new backup codes for MFA
func (h *AuthHandler) GetBackupCodes(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDUUID, _ := uuid.Parse(userID.(string))

	_, err := h.auth.GetUser(c.Request.Context(), userIDUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "User not found",
			},
		})
		return
	}

	// Return backup codes (one-time operation)
	// In production, these would be generated during MFA setup
	c.JSON(http.StatusOK, gin.H{
		"message": "Backup codes are only shown during MFA setup. Please contact administrator if lost.",
	})
}
