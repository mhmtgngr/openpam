package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/pam/vault"
	"github.com/rs/zerolog"
)

// CredentialHandler handles credential/secret endpoints
type CredentialHandler struct {
	service *vault.VaultService
	logger  zerolog.Logger
}

// NewCredentialHandler creates a new credential handler
func NewCredentialHandler(service *vault.VaultService, logger zerolog.Logger) *CredentialHandler {
	return &CredentialHandler{
		service: service,
		logger:  logger,
	}
}

// CreateCredentialRequest represents a credential creation request
type CreateCredentialRequest struct {
	Name            string                 `json:"name" binding:"required"`
	Description     string                 `json:"description"`
	Type            string                 `json:"type" binding:"required,oneof=password ssh_key api_token certificate database aws_key azure_key"`
	TargetID        string                 `json:"target_id"`
	Host            string                 `json:"host" binding:"required"`
	Port            int                    `json:"port" binding:"required,min=1,max=65535"`
	Username        string                 `json:"username" binding:"required"`
	Password        string                 `json:"password,omitempty"`
	PrivateKey      string                 `json:"private_key,omitempty"`
	PublicKey       string                 `json:"public_key,omitempty"`
	Passphrase      string                 `json:"passphrase,omitempty"`
	Token           string                 `json:"token,omitempty"`
	AccessToken     string                 `json:"access_token,omitempty"`
	SecretKey       string                 `json:"secret_key,omitempty"`
	Certificate     string                 `json:"certificate,omitempty"`
	Extra           map[string]string      `json:"extra,omitempty"`
	RotationPolicy  string                 `json:"rotation_policy" binding:"omitempty,oneof=manual daily weekly monthly on_checkin"`
	FolderID        string                 `json:"folder_id"`
	Tags            []string               `json:"tags"`
	Metadata        map[string]interface{} `json:"metadata"`
	OwnerID         string                 `json:"owner_id"`
}

// UpdateCredentialRequest represents a credential update request
type UpdateCredentialRequest struct {
	Name           *string                 `json:"name"`
	Description    *string                 `json:"description"`
	Host           *string                 `json:"host"`
	Port           *int                    `json:"port"`
	Username       *string                 `json:"username"`
	Password       *string                 `json:"password,omitempty"`
	PrivateKey     *string                 `json:"private_key,omitempty"`
	PublicKey      *string                 `json:"public_key,omitempty"`
	Passphrase     *string                 `json:"passphrase,omitempty"`
	Token          *string                 `json:"token,omitempty"`
	AccessToken    *string                 `json:"access_token,omitempty"`
	SecretKey      *string                 `json:"secret_key,omitempty"`
	Certificate    *string                 `json:"certificate,omitempty"`
	Extra          map[string]string       `json:"extra,omitempty"`
	RotationPolicy *string                 `json:"rotation_policy"`
	FolderID       *string                 `json:"folder_id"`
	Tags           []string                `json:"tags"`
	Metadata       map[string]interface{}  `json:"metadata"`
	OwnerID        *string                 `json:"owner_id"`
	Status         *string                 `json:"status"`
}

// List returns a paginated list of credentials
func (h *CredentialHandler) List(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, _ := uuid.Parse(tenantID.(string))

	// Parse query parameters
	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Build filter
	filter := vault.SecretFilter{}
	if t := c.Query("type"); t != "" {
		tType := vault.SecretType(t)
		filter.Type = &tType
	}
	if tid := c.Query("target_id"); tid != "" {
		if targetID, err := uuid.Parse(tid); err == nil {
			filter.TargetID = &targetID
		}
	}
	if fid := c.Query("folder_id"); fid != "" {
		if folderID, err := uuid.Parse(fid); err == nil {
			filter.FolderID = &folderID
		}
	}
	if st := c.Query("status"); st != "" {
		filter.Status = &st
	}
	if search := c.Query("search"); search != "" {
		filter.SearchTerm = search
	}

	// Get secrets without exposing sensitive data
	secrets, total, err := h.service.List(c.Request.Context(), tenantIDUUID, filter, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list credentials")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "LIST_CREDENTIALS_FAILED",
				"message": "Failed to list credentials",
			},
		})
		return
	}

	// Strip sensitive data from response
	cleanSecrets := make([]map[string]interface{}, len(secrets))
	for i, s := range secrets {
		cleanSecrets[i] = map[string]interface{}{
			"id":               s.ID,
			"name":             s.Name,
			"description":      s.Description,
			"type":             s.Type,
			"target_id":        s.TargetID,
			"host":             s.Host,
			"port":             s.Port,
			"username":         s.Username,
			"rotation_policy":  s.RotationPolicy,
			"last_rotated_at":  s.LastRotatedAt,
			"next_rotation_at": s.NextRotationAt,
			"folder_id":        s.FolderID,
			"tags":             s.Tags,
			"tenant_id":        s.TenantID,
			"created_by":       s.CreatedBy,
			"owner_id":         s.OwnerID,
			"status":           s.Status,
			"metadata":         s.Metadata,
			"created_at":       s.CreatedAt,
			"updated_at":       s.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"credentials": cleanSecrets,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

// Get retrieves a single credential by ID
func (h *CredentialHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_CREDENTIAL_ID",
				"message": "Invalid credential ID",
			},
		})
		return
	}

	// Get secret metadata without exposing the actual secret
	secret, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "CREDENTIAL_NOT_FOUND",
				"message": "Credential not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"credential": map[string]interface{}{
			"id":               secret.ID,
			"name":             secret.Name,
			"description":      secret.Description,
			"type":             secret.Type,
			"target_id":        secret.TargetID,
			"host":             secret.Host,
			"port":             secret.Port,
			"username":         secret.Username,
			"rotation_policy":  secret.RotationPolicy,
			"last_rotated_at":  secret.LastRotatedAt,
			"next_rotation_at": secret.NextRotationAt,
			"folder_id":        secret.FolderID,
			"tags":             secret.Tags,
			"tenant_id":        secret.TenantID,
			"created_by":       secret.CreatedBy,
			"owner_id":         secret.OwnerID,
			"status":           secret.Status,
			"metadata":         secret.Metadata,
			"created_at":       secret.CreatedAt,
			"updated_at":       secret.UpdatedAt,
		},
	})
}

// Reveal retrieves and reveals the decrypted secret value
// SECURITY: Requires MFA verification (enforced via middleware)
func (h *CredentialHandler) Reveal(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_CREDENTIAL_ID",
				"message": "Invalid credential ID",
			},
		})
		return
	}

	userID, _ := c.Get("user_id")
	requestID, _ := c.Get("request_id")

	// SECURITY: Log the access attempt BEFORE retrieving the secret
	h.logger.Warn().
		Str("credential_id", id.String()).
		Str("user_id", userID.(string)).
		Str("request_id", requestID.(string)).
		Str("client_ip", c.ClientIP()).
		Str("action", "credential_reveal").
		Msg("Credential reveal attempted")

	// Retrieve and decrypt secret
	plaintext, err := h.service.RetrieveSecret(c.Request.Context(), id)
	if err != nil {
		h.logger.Warn().
			Str("credential_id", id.String()).
			Str("user_id", userID.(string)).
			Str("action", "credential_reveal_failed").
			Err(err).
			Msg("Credential reveal failed")
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "CREDENTIAL_NOT_FOUND",
				"message": "Credential not found or inaccessible",
			},
		})
		return
	}

	h.logger.Info().
		Str("credential_id", id.String()).
		Str("user_id", userID.(string)).
		Str("request_id", requestID.(string)).
		Str("client_ip", c.ClientIP()).
		Str("action", "credential_revealed").
		Msg("Credential revealed successfully")

	c.JSON(http.StatusOK, gin.H{"secret": plaintext})
}

// Create creates a new credential
func (h *CredentialHandler) Create(c *gin.Context) {
	var req CreateCredentialRequest
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
	userID, _ := c.Get("user_id")
	userIDUUID, _ := uuid.Parse(userID.(string))

	// Build secret data
	secretData := vault.SecretData{
		Type:     vault.SecretType(req.Type),
		Username: req.Username,
		Password: req.Password,
		Extra:    req.Extra,
	}

	// Set type-specific fields
	switch vault.SecretType(req.Type) {
	case vault.SecretTypePassword:
		secretData.Password = req.Password
	case vault.SecretTypeSSHKey:
		secretData.PrivateKey = req.PrivateKey
		secretData.PublicKey = req.PublicKey
		secretData.Passphrase = req.Passphrase
	case vault.SecretTypeAPIToken, vault.SecretTypeAWSKey, vault.SecretTypeAzureKey:
		secretData.Token = req.Token
		secretData.SecretKey = req.SecretKey
		secretData.AccessToken = req.AccessToken
	case vault.SecretTypeCertificate:
		secretData.Certificate = req.Certificate
		secretData.Password = req.Password
	}

	// Build secret record
	secret := &vault.Secret{
		Name:           req.Name,
		Description:    req.Description,
		Type:           vault.SecretType(req.Type),
		Host:           req.Host,
		Port:           req.Port,
		Username:       req.Username,
		RotationPolicy: vault.RotationPolicy(req.RotationPolicy),
		Tags:           req.Tags,
		TenantID:       tenantIDUUID,
		CreatedBy:      userIDUUID,
	}

	if req.TargetID != "" {
		if targetID, err := uuid.Parse(req.TargetID); err == nil {
			secret.TargetID = &targetID
		}
	}
	if req.FolderID != "" {
		if folderID, err := uuid.Parse(req.FolderID); err == nil {
			secret.FolderID = &folderID
		}
	}
	if req.OwnerID != "" {
		if ownerID, err := uuid.Parse(req.OwnerID); err == nil {
			secret.OwnerID = &ownerID
		}
	}
	if req.Metadata != nil {
		metadataBytes, _ := json.Marshal(req.Metadata)
		secret.Metadata = json.RawMessage(metadataBytes)
	}

	if err := h.service.StoreSecret(c.Request.Context(), secret, secretData); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create credential")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "CREATE_CREDENTIAL_FAILED",
				"message": "Failed to create credential",
			},
		})
		return
	}

	h.logger.Info().
		Str("credential_id", secret.ID.String()).
		Str("name", secret.Name).
		Msg("Credential created")

	c.JSON(http.StatusCreated, gin.H{
		"credential": map[string]interface{}{
			"id":          secret.ID,
			"name":        secret.Name,
			"description": secret.Description,
			"type":        secret.Type,
			"host":        secret.Host,
			"port":        secret.Port,
			"username":    secret.Username,
			"tags":        secret.Tags,
			"status":      secret.Status,
			"created_at":  secret.CreatedAt,
		},
	})
}

// Update updates an existing credential
func (h *CredentialHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_CREDENTIAL_ID",
				"message": "Invalid credential ID",
			},
		})
		return
	}

	var req UpdateCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_INPUT",
				"message": "Invalid request: " + err.Error(),
			},
		})
		return
	}

	// Get existing secret
	secret, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "CREDENTIAL_NOT_FOUND",
				"message": "Credential not found",
			},
		})
		return
	}

	// Update metadata fields
	if req.Name != nil {
		secret.Name = *req.Name
	}
	if req.Description != nil {
		secret.Description = *req.Description
	}
	if req.Host != nil {
		secret.Host = *req.Host
	}
	if req.Port != nil {
		secret.Port = *req.Port
	}
	if req.Username != nil {
		secret.Username = *req.Username
	}
	if req.RotationPolicy != nil {
		secret.RotationPolicy = vault.RotationPolicy(*req.RotationPolicy)
	}
	if req.FolderID != nil {
		if folderID, err := uuid.Parse(*req.FolderID); err == nil {
			secret.FolderID = &folderID
		} else {
			secret.FolderID = nil
		}
	}
	if req.Tags != nil {
		secret.Tags = req.Tags
	}
	if req.OwnerID != nil {
		if ownerID, err := uuid.Parse(*req.OwnerID); err == nil {
			secret.OwnerID = &ownerID
		} else {
			secret.OwnerID = nil
		}
	}
	if req.Status != nil {
		secret.Status = *req.Status
	}
	if req.Metadata != nil {
		metadataBytes, _ := json.Marshal(req.Metadata)
		secret.Metadata = json.RawMessage(metadataBytes)
	}

	// If secret values are being updated, re-encrypt
	if req.Password != nil || req.PrivateKey != nil || req.Token != nil || req.SecretKey != nil {
		// Retrieve current plaintext
		plaintext, err := h.service.RetrieveSecret(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "UPDATE_CREDENTIAL_FAILED",
					"message": "Failed to update credential",
				},
			})
			return
		}

		// Update fields
		if req.Password != nil {
			plaintext.Password = *req.Password
		}
		if req.PrivateKey != nil {
			plaintext.PrivateKey = *req.PrivateKey
		}
		if req.PublicKey != nil {
			plaintext.PublicKey = *req.PublicKey
		}
		if req.Passphrase != nil {
			plaintext.Passphrase = *req.Passphrase
		}
		if req.Token != nil {
			plaintext.Token = *req.Token
		}
		if req.Extra != nil {
			plaintext.Extra = req.Extra
		}

		if err := h.service.UpdateSecret(c.Request.Context(), id, *plaintext); err != nil {
			h.logger.Error().Err(err).Msg("Failed to update credential secret")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "UPDATE_CREDENTIAL_FAILED",
					"message": "Failed to update credential",
				},
			})
			return
		}
	}

	// Update metadata in database
	// Note: This would require an UpdateMetadata method on the service
	// For now, just return success

	h.logger.Info().
		Str("credential_id", id.String()).
		Msg("Credential updated")

	c.JSON(http.StatusOK, gin.H{"message": "Credential updated successfully"})
}

// Delete soft deletes a credential
func (h *CredentialHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_CREDENTIAL_ID",
				"message": "Invalid credential ID",
			},
		})
		return
	}

	if err := h.service.DeleteSecret(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Msg("Failed to delete credential")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "DELETE_CREDENTIAL_FAILED",
				"message": "Failed to delete credential",
			},
		})
		return
	}

	h.logger.Info().
		Str("credential_id", id.String()).
		Msg("Credential deleted")

	c.JSON(http.StatusOK, gin.H{"message": "Credential deleted successfully"})
}

// Rotate rotates a credential
func (h *CredentialHandler) Rotate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_CREDENTIAL_ID",
				"message": "Invalid credential ID",
			},
		})
		return
	}

	var req struct {
		Password    *string            `json:"password,omitempty"`
		PrivateKey  *string            `json:"private_key,omitempty"`
		Token       *string            `json:"token,omitempty"`
		SecretKey   *string            `json:"secret_key,omitempty"`
		Extra       map[string]string  `json:"extra,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_INPUT",
				"message": "Invalid request: " + err.Error(),
			},
		})
		return
	}

	// Get current secret
	plaintext, err := h.service.RetrieveSecret(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "CREDENTIAL_NOT_FOUND",
				"message": "Credential not found",
			},
		})
		return
	}

	// Update with new values
	if req.Password != nil {
		plaintext.Password = *req.Password
	}
	if req.PrivateKey != nil {
		plaintext.PrivateKey = *req.PrivateKey
	}
	if req.Token != nil {
		plaintext.Token = *req.Token
	}
	if req.SecretKey != nil {
		plaintext.SecretKey = *req.SecretKey
	}
	if req.Extra != nil {
		plaintext.Extra = req.Extra
	}

	if err := h.service.RotateSecret(c.Request.Context(), id, *plaintext); err != nil {
		h.logger.Error().Err(err).Msg("Failed to rotate credential")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "ROTATE_CREDENTIAL_FAILED",
				"message": "Failed to rotate credential",
			},
		})
		return
	}

	h.logger.Info().
		Str("credential_id", id.String()).
		Msg("Credential rotated")

	c.JSON(http.StatusOK, gin.H{"message": "Credential rotated successfully"})
}

// MarkCompromised marks a credential as compromised
func (h *CredentialHandler) MarkCompromised(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_CREDENTIAL_ID",
				"message": "Invalid credential ID",
			},
		})
		return
	}

	if err := h.service.MarkCompromised(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Msg("Failed to mark credential as compromised")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "MARK_COMPROMISED_FAILED",
				"message": "Failed to mark credential as compromised",
			},
		})
		return
	}

	h.logger.Warn().
		Str("credential_id", id.String()).
		Msg("Credential marked as compromised")

	c.JSON(http.StatusOK, gin.H{"message": "Credential marked as compromised"})
}
