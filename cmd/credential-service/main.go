package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/credential"
	"github.com/openpam/openpam/internal/database"
	"github.com/openpam/openpam/internal/events"
	"github.com/openpam/openpam/internal/middleware"
	"github.com/openpam/openpam/internal/pam/rotation"
	"github.com/openpam/openpam/internal/pam/vault"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configuration
	config := loadConfig()

	// Setup logging
	setupLogger(config.LogLevel)
	logger := log.With().Str("service", "credential-service").Logger()

	// Initialize dependencies
	db, err := database.New(database.Config{
		Host:            config.DBHost,
		Port:            config.DBPort,
		User:            config.DBUser,
		Password:        config.DBPassword,
		Database:        config.DBName,
		SSLMode:         config.DBSSLMode,
		SSLRootCert:     config.DBSSLRootCert,
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 5,
	}, logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}

	redisCache, err := cache.New(cache.Config{
		Host:     config.RedisHost,
		Port:     config.RedisPort,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
		PoolSize: 100,
	}, logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}

	// Initialize master key
	masterKey := deriveMasterKey(config.MasterKey)
	keyID := "current"

	// Initialize envelope encryption
	envelope, err := vault.NewEnvelopeEncryption(masterKey, keyID, db.DB, redisCache, logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize envelope encryption")
	}

	// Initialize services
	eventSigningKey := os.Getenv("EVENT_SIGNING_KEY")
	if len(eventSigningKey) < 32 {
		log.Fatal().Msg("EVENT_SIGNING_KEY environment variable must be set and at least 32 bytes for HMAC-SHA256 security")
	}
	eventBus, err := events.New(events.EventConfig{
		Cache:      redisCache,
		Logger:     logger,
		SigningKey: []byte(eventSigningKey),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize event bus")
	}
	eventPublisher := events.NewPublisher(eventBus)

	secretRepo := vault.NewSecretRepository(db.DB, redisCache, logger)
	vaultSvc := vault.NewVaultService(secretRepo, envelope, logger)
	rotationSvc := rotation.NewService(db.DB, vaultSvc, redisCache, logger)
	credentialSvc := credential.NewService(db.DB, vaultSvc, rotationSvc, redisCache, logger)

	// Setup router
	router := setupRouter(config, db, redisCache, credentialSvc, eventPublisher, logger)

	// Start server
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	go func() {
		logger.Info().Str("port", config.Port).Msg("Starting credential-service server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("Forced shutdown")
	}

	logger.Info().Msg("Goodbye.")
}

// Config holds application configuration
type Config struct {
	Port      string
	LogLevel  string
	MasterKey string

	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	DBSSLRootCert string

	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int
}

func loadConfig() Config {
	return Config{
		Port:      getEnv("PORT", "8503"),
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		MasterKey: getEnv("MASTER_KEY", "development-key-32-bytes-long-!!"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvInt("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", "openpam"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "openpam"),
		DBSSLMode:  getEnv("DB_SSLMODE", "require"), // SECURITY: Default to require SSL
		DBSSLRootCert: getEnv("DB_SSLROOTCERT", "/etc/ssl/certs/postgresql-ca.crt"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnvInt("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),
	}
}

// deriveMasterKey derives a 32-byte key from the master key string
func deriveMasterKey(masterKeyStr string) []byte {
	hash := sha256.Sum256([]byte(masterKeyStr))
	return hash[:]
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var intVal int
		if _, err := fmt.Sscanf(val, "%d", &intVal); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func setupLogger(level string) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func setupRouter(
	config Config,
	db *database.DB,
	cache *cache.Cache,
	credentialSvc *credential.Service,
	publisher *events.Publisher,
	logger zerolog.Logger,
) *gin.Engine {
	if config.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Middleware
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(middleware.Config{
		AllowedOrigins:  []string{"*"},
		AllowedMethods:  []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:  []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"},
		ExposeHeaders:   []string{"Content-Length", "X-Request-ID"},
	}))
	r.Use(middleware.RequestID())

	// Health endpoints
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "up", "service": "credential-service"})
	})
	r.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		ready := true
		details := gin.H{}

		if err := db.Health(ctx); err != nil {
			ready = false
			details["database"] = "unhealthy"
		} else {
			details["database"] = "healthy"
		}

		if err := cache.Health(ctx); err != nil {
			ready = false
			details["cache"] = "unhealthy"
		} else {
			details["cache"] = "healthy"
		}

		if ready {
			c.JSON(http.StatusOK, gin.H{"ready": true, "details": details})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"ready": false, "details": details})
		}
	})

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Protected routes (require auth)
		protected := v1.Group("")
		protected.Use(middleware.Auth())
		{
			// Credential management
			protected.POST("/credentials", handleCreateCredential(credentialSvc, logger))
			protected.GET("/credentials", handleListCredentials(credentialSvc, logger))
			protected.GET("/credentials/:id", handleGetCredential(credentialSvc, logger))
			protected.PUT("/credentials/:id", handleUpdateCredential(credentialSvc, logger))
			protected.DELETE("/credentials/:id", handleDeleteCredential(credentialSvc, logger))

			// Credential data
			protected.GET("/credentials/:id/reveal", handleRevealCredential(credentialSvc, logger))

			// Rotation
			protected.POST("/credentials/:id/rotate", handleRotateCredential(credentialSvc, logger))
			protected.POST("/credentials/:id/test", handleTestCredential(credentialSvc, logger))

			// History
			protected.GET("/credentials/:id/history", handleCredentialHistory(credentialSvc, logger))

			// Usage stats
			protected.GET("/credentials/:id/stats", handleCredentialStats(credentialSvc, logger))

			// Batch operations
			protected.POST("/credentials/batch/rotate", handleBatchRotate(credentialSvc, logger))

			// Admin routes
			admin := protected.Group("/admin")
			admin.Use(middleware.RequireRole("admin", "super_admin"))
			{
				admin.GET("/credentials/pending-rotation", handlePendingRotation(credentialSvc, logger))
			}
		}
	}

	return r
}

// Handler functions

func handleCreateCredential(svc *credential.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name           string                 `json:"name" binding:"required"`
			Description    string                 `json:"description"`
			Type           string                 `json:"type" binding:"required"`
			TargetID       *uuid.UUID            `json:"target_id"`
			Host           string                 `json:"host" binding:"required"`
			Port           int                    `json:"port" binding:"required"`
			Username       string                 `json:"username" binding:"required"`
			Password       string                 `json:"password,omitempty"`
			PrivateKey     string                 `json:"private_key,omitempty"`
			PublicKey      string                 `json:"public_key,omitempty"`
			Token          string                 `json:"token,omitempty"`
			SecretKey      string                 `json:"secret_key,omitempty"`
			Certificate    string                 `json:"certificate,omitempty"`
			RotationPolicy string                 `json:"rotation_policy"`
			FolderID       *uuid.UUID            `json:"folder_id"`
			Tags           []string               `json:"tags"`
			TenantID       uuid.UUID              `json:"tenant_id" binding:"required"`
			CreatedBy      uuid.UUID              `json:"created_by" binding:"required"`
			OwnerID        *uuid.UUID            `json:"owner_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		secret := &vault.Secret{
			Name:           req.Name,
			Description:    req.Description,
			Type:           vault.SecretType(req.Type),
			TargetID:       req.TargetID,
			Host:           req.Host,
			Port:           req.Port,
			Username:       req.Username,
			RotationPolicy: vault.RotationPolicy(req.RotationPolicy),
			FolderID:       req.FolderID,
			Tags:           req.Tags,
			TenantID:       req.TenantID,
			CreatedBy:      req.CreatedBy,
			OwnerID:        req.OwnerID,
		}

		// Build secret data based on type
		secretData := vault.SecretData{
			Type:     vault.SecretType(req.Type),
			Username: req.Username,
		}

		switch vault.SecretType(req.Type) {
		case vault.SecretTypePassword, vault.SecretTypeDatabase:
			secretData.Password = req.Password
		case vault.SecretTypeSSHKey:
			secretData.PrivateKey = req.PrivateKey
			secretData.PublicKey = req.PublicKey
		case vault.SecretTypeAPIToken, vault.SecretTypeAWSKey, vault.SecretTypeAzureKey:
			if req.Token != "" {
				secretData.Token = req.Token
			} else {
				secretData.SecretKey = req.SecretKey
			}
		case vault.SecretTypeCertificate:
			secretData.Certificate = req.Certificate
		}

		if err := svc.CreateCredential(c.Request.Context(), secret, secretData); err != nil {
			logger.Error().Err(err).Msg("Failed to create credential")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create credential"}})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"credential": secret})
	}
}

func handleListCredentials(svc *credential.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, _ := c.Get("tenant_id")
		tenantIDUUID, err := uuid.Parse(tenantID.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		limit := getIntQuery(c, "limit", 50)
		offset := getIntQuery(c, "offset", 0)

		var filter vault.SecretFilter
		if secretType := c.Query("type"); secretType != "" {
			filter.Type = (*vault.SecretType)(&secretType)
		}
		if folderID := c.Query("folder_id"); folderID != "" {
			fid, _ := uuid.Parse(folderID)
			filter.FolderID = &fid
		}

		credentials, total, err := svc.ListCredentials(c.Request.Context(), tenantIDUUID, filter, limit, offset)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to list credentials")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list credentials"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"credentials": credentials,
			"total":       total,
			"limit":       limit,
			"offset":      offset,
		})
	}
}

func handleGetCredential(svc *credential.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid credential ID"}})
			return
		}

		cred, err := svc.GetCredential(c.Request.Context(), id)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get credential")
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Credential not found"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"credential": cred})
	}
}

func handleRevealCredential(svc *credential.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid credential ID"}})
			return
		}

		// This endpoint requires MFA verification in production
		// For now, we'll just return the secret data
		c.JSON(http.StatusNotImplemented, gin.H{"error": gin.H{"code": "NOT_IMPLEMENTED", "message": "MFA verification required"}})
	}
}

func handleUpdateCredential(svc *credential.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid credential ID"}})
			return
		}

		var req struct {
			Password    string `json:"password,omitempty"`
			PrivateKey  string `json:"private_key,omitempty"`
			PublicKey   string `json:"public_key,omitempty"`
			Token       string `json:"token,omitempty"`
			SecretKey   string `json:"secret_key,omitempty"`
			Certificate string `json:"certificate,omitempty"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		// Get current credential to determine type
		secret, err := svc.GetCredential(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Credential not found"}})
			return
		}

		// Build secret data based on type
		secretData := vault.SecretData{
			Type: secret.Type,
		}

		switch secret.Type {
		case vault.SecretTypePassword, vault.SecretTypeDatabase:
			secretData.Password = req.Password
		case vault.SecretTypeSSHKey:
			secretData.PrivateKey = req.PrivateKey
			secretData.PublicKey = req.PublicKey
		case vault.SecretTypeAPIToken, vault.SecretTypeAWSKey, vault.SecretTypeAzureKey:
			if req.Token != "" {
				secretData.Token = req.Token
			} else {
				secretData.SecretKey = req.SecretKey
			}
		case vault.SecretTypeCertificate:
			secretData.Certificate = req.Certificate
		}

		if err := svc.UpdateCredential(c.Request.Context(), id, secretData); err != nil {
			logger.Error().Err(err).Msg("Failed to update credential")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to update credential"}})
			return
		}

		logger.Info().Str("credential_id", id.String()).Msg("Credential updated")
		c.JSON(http.StatusOK, gin.H{"message": "Credential updated"})
	}
}

func handleDeleteCredential(svc *credential.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid credential ID"}})
			return
		}

		if err := svc.DeleteCredential(c.Request.Context(), id); err != nil {
			logger.Error().Err(err).Msg("Failed to delete credential")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to delete credential"}})
			return
		}

		logger.Info().Str("credential_id", id.String()).Msg("Credential deleted")
		c.JSON(http.StatusOK, gin.H{"message": "Credential deleted"})
	}
}

func handleRotateCredential(svc *credential.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid credential ID"}})
			return
		}

		var req struct {
			NewPassword   string `json:"new_password,omitempty"`
			NewToken      string `json:"new_token,omitempty"`
			NewSecretKey  string `json:"new_secret_key,omitempty"`
			NewPrivateKey string `json:"new_private_key,omitempty"`
			NewPublicKey  string `json:"new_public_key,omitempty"`
		}
		_ = c.ShouldBindJSON(&req)

		// Get current credential
		secret, err := svc.GetCredential(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Credential not found"}})
			return
		}

		secretData := vault.SecretData{
			Type:     secret.Type,
			Username: secret.Username,
		}

		if req.NewPassword != "" {
			secretData.Password = req.NewPassword
		}
		if req.NewToken != "" {
			secretData.Token = req.NewToken
		}
		if req.NewSecretKey != "" {
			secretData.SecretKey = req.NewSecretKey
		}
		if req.NewPrivateKey != "" {
			secretData.PrivateKey = req.NewPrivateKey
		}
		if req.NewPublicKey != "" {
			secretData.PublicKey = req.NewPublicKey
		}

		if err := svc.RotateCredential(c.Request.Context(), id, secretData); err != nil {
			logger.Error().Err(err).Msg("Failed to rotate credential")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to rotate credential"}})
			return
		}

		logger.Info().Str("credential_id", id.String()).Msg("Credential rotated")
		c.JSON(http.StatusOK, gin.H{"message": "Credential rotated"})
	}
}

func handleTestCredential(svc *credential.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid credential ID"}})
			return
		}

		if err := svc.TestCredential(c.Request.Context(), id); err != nil {
			logger.Error().Err(err).Msg("Credential test failed")
			c.JSON(http.StatusOK, gin.H{"valid": false, "error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"valid": true})
	}
}

func handleCredentialHistory(svc *credential.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid credential ID"}})
			return
		}

		limit := getIntQuery(c, "limit", 20)
		history, err := svc.GetCredentialHistory(c.Request.Context(), id, limit)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get credential history")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get history"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"history": history})
	}
}

func handleCredentialStats(svc *credential.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid credential ID"}})
			return
		}

		stats, err := svc.GetCredentialUsageStats(c.Request.Context(), id)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get credential stats")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get stats"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"stats": stats})
	}
}

func handleBatchRotate(svc *credential.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			CredentialIDs []uuid.UUID `json:"credential_ids" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		results, rotated := svc.BatchRotateCredentials(c.Request.Context(), req.CredentialIDs)

		c.JSON(http.StatusOK, gin.H{
			"results":  results,
			"rotated":  rotated,
			"total":    len(req.CredentialIDs),
			"success":  len(rotated),
			"failed":   len(results) - len(rotated),
		})
	}
}

func handlePendingRotation(svc *credential.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, _ := c.Get("tenant_id")
		tenantIDUUID, err := uuid.Parse(tenantID.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		credentials, err := svc.GetCredentialsRequiringRotation(c.Request.Context(), tenantIDUUID)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get pending rotations")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get pending rotations"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"credentials": credentials})
	}
}

// Helper functions

func buildSecretData(req struct {
	Name           string
	Description    string
	Type           string
	TargetID       *uuid.UUID
	Host           string
	Port           int
	Username       string
	Password       string
	PrivateKey     string
	PublicKey      string
	Token          string
	SecretKey      string
	Certificate    string
	RotationPolicy string
	FolderID       *uuid.UUID
	Tags           []string
	TenantID       uuid.UUID
	CreatedBy      uuid.UUID
	OwnerID        *uuid.UUID
}) (vault.SecretData, error) {
	data := vault.SecretData{
		Type:     vault.SecretType(req.Type),
		Username: req.Username,
	}

	switch vault.SecretType(req.Type) {
	case vault.SecretTypePassword, vault.SecretTypeDatabase:
		data.Password = req.Password
	case vault.SecretTypeSSHKey:
		data.PrivateKey = req.PrivateKey
		data.PublicKey = req.PublicKey
	case vault.SecretTypeAPIToken, vault.SecretTypeAWSKey, vault.SecretTypeAzureKey:
		if req.Token != "" {
			data.Token = req.Token
		} else {
			data.SecretKey = req.SecretKey
		}
	case vault.SecretTypeCertificate:
		data.Certificate = req.Certificate
	default:
		return data, fmt.Errorf("unsupported secret type: %s", req.Type)
	}

	return data, nil
}

func buildUpdateSecretData(secretType vault.SecretType, req struct {
	Password    string
	PrivateKey  string
	PublicKey   string
	Token       string
	SecretKey   string
	Certificate string
}) vault.SecretData {
	data := vault.SecretData{
		Type: secretType,
	}

	switch secretType {
	case vault.SecretTypePassword, vault.SecretTypeDatabase:
		data.Password = req.Password
	case vault.SecretTypeSSHKey:
		data.PrivateKey = req.PrivateKey
		data.PublicKey = req.PublicKey
	case vault.SecretTypeAPIToken, vault.SecretTypeAWSKey, vault.SecretTypeAzureKey:
		if req.Token != "" {
			data.Token = req.Token
		} else {
			data.SecretKey = req.SecretKey
		}
	case vault.SecretTypeCertificate:
		data.Certificate = req.Certificate
	}

	return data
}

func getIntQuery(c *gin.Context, key string, defaultVal int) int {
	val := c.Query(key)
	if val == "" {
		return defaultVal
	}
	var intVal int
	if _, err := fmt.Sscanf(val, "%d", &intVal); err == nil {
		return intVal
	}
	return defaultVal
}
