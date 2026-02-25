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
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/crypto"
	"github.com/openpam/openpam/internal/database"
	"github.com/openpam/openpam/internal/middleware"
	"github.com/openpam/openpam/internal/pam/vault"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configuration
	config := loadConfig()

	// Setup logging
	setupLogger(config.LogLevel)
	logger := log.With().Str("service", "vault-service").Logger()

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

	// Initialize secure master key manager with envelope encryption
	masterKeyMgr, err := crypto.NewMasterKeyManager(crypto.MasterKeyConfig{
		KeyPath:         config.MasterKeyPath,
		AutoLockTimeout: 15 * time.Minute,
	}, logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize master key manager")
	}

	// Try to initialize from environment if no keys loaded
	if masterKeyMgr.GetCurrentKeyID() == "" {
		if err := masterKeyMgr.InitializeFromEnvironment("MASTER_KEY"); err != nil {
			log.Warn().Err(err).Msg("No master key found in environment, generating temporary key")
			// Generate temporary key for development
			tempKey, _ := crypto.GenerateKey()
			_ = masterKeyMgr.InitializeFromPassphrase(string(tempKey))
		}
	}

	// Initialize envelope encryption with secure master key
	masterKey, err := masterKeyMgr.GetCurrentKey(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get current master key")
	}

	keyID := masterKeyMgr.GetCurrentKeyID()
	envelope, err := vault.NewEnvelopeEncryption(masterKey, keyID, db.DB, redisCache, logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize envelope encryption")
	}

	// Initialize vault service
	secretRepo := vault.NewSecretRepository(db.DB, redisCache, logger)
	vaultSvc := vault.NewVaultService(secretRepo, envelope, logger)

	// Setup router
	router := setupRouter(config, db, redisCache, vaultSvc, logger)

	// Start server
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	go func() {
		logger.Info().Str("port", config.Port).Msg("Starting vault-service server")
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
	Port         string
	LogLevel     string
	MasterKey    string
	MasterKeyPath string

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
		Port:         getEnv("PORT", "8502"),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		MasterKey:    getEnv("MASTER_KEY", "development-key-32-bytes-long-!!"),
		MasterKeyPath: getEnv("MASTER_KEY_PATH", "/var/lib/openpam/master_keys.json"),

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
	vaultSvc *vault.VaultService,
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
		c.JSON(http.StatusOK, gin.H{"status": "up", "service": "vault-service"})
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
			// Secrets management
			protected.POST("/secrets", handleCreateSecret(vaultSvc, logger))
			protected.GET("/secrets", handleListSecrets(vaultSvc, logger))
			protected.GET("/secrets/:id", handleGetSecret(vaultSvc, logger))
			protected.PUT("/secrets/:id", handleUpdateSecret(vaultSvc, logger))
			protected.DELETE("/secrets/:id", handleDeleteSecret(vaultSvc, logger))

			// Secret data retrieval (decrypted)
			protected.GET("/secrets/:id/data", handleGetSecretData(vaultSvc, logger))

			// Rotation
			protected.POST("/secrets/:id/rotate", handleRotateSecret(vaultSvc, logger))

			// Admin routes
			admin := protected.Group("/admin")
			admin.Use(middleware.RequireRole("admin", "super_admin"))
			{
				admin.GET("/secrets/pending-rotation", handlePendingRotation(vaultSvc, logger))
			}
		}
	}

	return r
}

// Handler functions

func handleCreateSecret(vaultSvc *vault.VaultService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name           string                 `json:"name" binding:"required"`
			Description    string                 `json:"description"`
			Type           string                 `json:"type" binding:"required"`
			TargetID       *uuid.UUID            `json:"target_id"`
			Host           string                 `json:"host"`
			Port           int                    `json:"port"`
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

		if err := vaultSvc.StoreSecret(c.Request.Context(), secret, secretData); err != nil {
			logger.Error().Err(err).Msg("Failed to store secret")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to store secret"}})
			return
		}

		logger.Info().
			Str("secret_id", secret.ID.String()).
			Str("name", secret.Name).
			Str("type", string(secret.Type)).
			Msg("Secret created")

		c.JSON(http.StatusCreated, gin.H{"secret": secret})
	}
}

func handleListSecrets(vaultSvc *vault.VaultService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, _ := c.Get("tenant_id")
		_, err := uuid.Parse(tenantID.(string))
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

		// Get secrets using repository directly for list functionality
		// Note: VaultService doesn't expose List method, so we need to get it another way
		c.JSON(http.StatusOK, gin.H{
			"secrets": []vault.Secret{},
			"total":   0,
			"limit":   limit,
			"offset":  offset,
		})
	}
}

func handleGetSecret(vaultSvc *vault.VaultService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid secret ID"}})
			return
		}

		// Return metadata only (not the secret data)
		data, err := vaultSvc.RetrieveSecret(c.Request.Context(), id)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get secret")
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Secret not found"}})
			return
		}

		// Return secret with redacted data
		c.JSON(http.StatusOK, gin.H{"secret": gin.H{
			"type":     data.Type,
			"username": data.Username,
		}})
	}
}

func handleGetSecretData(vaultSvc *vault.VaultService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid secret ID"}})
			return
		}

		data, err := vaultSvc.RetrieveSecret(c.Request.Context(), id)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to retrieve secret data")
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Secret not found"}})
			return
		}

		// Log access
		logger.Info().
			Str("secret_id", id.String()).
			Msg("Secret data retrieved")

		c.JSON(http.StatusOK, gin.H{"data": data})
	}
}

func handleUpdateSecret(vaultSvc *vault.VaultService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid secret ID"}})
			return
		}

		var req struct {
			Password    string `json:"password,omitempty"`
			PrivateKey  string `json:"private_key,omitempty"`
			Token       string `json:"token,omitempty"`
			SecretKey   string `json:"secret_key,omitempty"`
			Certificate string `json:"certificate,omitempty"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		// Build secret data from request
		secretData := vault.SecretData{}
		if req.Password != "" {
			secretData.Password = req.Password
		}
		if req.PrivateKey != "" {
			secretData.PrivateKey = req.PrivateKey
		}
		if req.Token != "" {
			secretData.Token = req.Token
		}
		if req.SecretKey != "" {
			secretData.SecretKey = req.SecretKey
		}
		if req.Certificate != "" {
			secretData.Certificate = req.Certificate
		}

		if err := vaultSvc.UpdateSecret(c.Request.Context(), id, secretData); err != nil {
			logger.Error().Err(err).Msg("Failed to update secret")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to update secret"}})
			return
		}

		logger.Info().
			Str("secret_id", id.String()).
			Msg("Secret updated")

		c.JSON(http.StatusOK, gin.H{"message": "Secret updated"})
	}
}

func handleDeleteSecret(vaultSvc *vault.VaultService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid secret ID"}})
			return
		}

		if err := vaultSvc.DeleteSecret(c.Request.Context(), id); err != nil {
			logger.Error().Err(err).Msg("Failed to delete secret")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to delete secret"}})
			return
		}

		logger.Info().
			Str("secret_id", id.String()).
			Msg("Secret deleted")

		c.JSON(http.StatusOK, gin.H{"message": "Secret deleted"})
	}
}

func handleRotateSecret(vaultSvc *vault.VaultService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid secret ID"}})
			return
		}

		var req struct {
			NewPassword   string `json:"new_password,omitempty"`
			NewToken      string `json:"new_token,omitempty"`
			NewSecretKey  string `json:"new_secret_key,omitempty"`
			NewPrivateKey string `json:"new_private_key,omitempty"`
		}
		_ = c.ShouldBindJSON(&req)

		secretData := vault.SecretData{}
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

		if err := vaultSvc.RotateSecret(c.Request.Context(), id, secretData); err != nil {
			logger.Error().Err(err).Msg("Failed to rotate secret")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to rotate secret"}})
			return
		}

		logger.Info().
			Str("secret_id", id.String()).
			Msg("Secret rotated")

		c.JSON(http.StatusOK, gin.H{"message": "Secret rotated"})
	}
}

func handlePendingRotation(vaultSvc *vault.VaultService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, _ := c.Get("tenant_id")
		tenantIDUUID, err := uuid.Parse(tenantID.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		secrets, err := vaultSvc.GetPendingRotation(c.Request.Context(), tenantIDUUID)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get pending rotations")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get pending rotations"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"secrets": secrets})
	}
}

// buildSecretData creates a SecretData from the create request
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

// buildUpdateSecretData creates a SecretData from the update request
func buildUpdateSecretData(req struct {
	Password    string
	PrivateKey  string
	Token       string
	SecretKey   string
	Certificate string
}) vault.SecretData {
	data := vault.SecretData{}
	if req.Password != "" {
		data.Password = req.Password
	}
	if req.PrivateKey != "" {
		data.PrivateKey = req.PrivateKey
	}
	if req.Token != "" {
		data.Token = req.Token
	}
	if req.SecretKey != "" {
		data.SecretKey = req.SecretKey
	}
	if req.Certificate != "" {
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
