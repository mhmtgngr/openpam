package main

import (
	"context"
	"fmt"
	"net/http"
	"net/mail"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/admin"
	"github.com/openpam/openpam/internal/auth"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/database"
	"github.com/openpam/openpam/internal/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configuration
	config := loadConfig()

	// Setup logging
	setupLogger(config.LogLevel)
	logger := log.With().Str("service", "admin-service").Logger()

	// Initialize dependencies
	db, err := database.New(database.Config{
		Host:            config.DBHost,
		Port:            config.DBPort,
		User:            config.DBUser,
		Password:        config.DBPassword,
		Database:        config.DBName,
		SSLMode:         config.DBSSLMode,
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

	// Initialize services
	adminSvc := admin.NewService(db.DB, redisCache, logger)
	roleRepo := auth.NewRoleRepository(db.DB, logger)
	roleSvc := auth.NewRoleService(roleRepo, logger)

	// Setup router
	router := setupRouter(config, db, redisCache, adminSvc, roleSvc, logger)

	// Start server
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	go func() {
		logger.Info().Str("port", config.Port).Msg("Starting admin-service server")
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
	Port     string
	LogLevel string

	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int
}

func loadConfig() Config {
	return Config{
		Port:     getEnv("PORT", "8505"),
		LogLevel: getEnv("LOG_LEVEL", "info"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvInt("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", "openpam"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "openpam"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

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
	adminSvc *admin.Service,
	roleSvc *auth.RoleService,
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
		c.JSON(http.StatusOK, gin.H{"status": "up", "service": "admin-service"})
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
			// Tenant management
			protected.GET("/tenants", handleListTenants(adminSvc, logger))
			protected.POST("/tenants", handleCreateTenant(adminSvc, logger))
			protected.GET("/tenants/:id", handleGetTenant(adminSvc, logger))
			protected.PUT("/tenants/:id", handleUpdateTenant(adminSvc, logger))
			protected.DELETE("/tenants/:id", handleDeleteTenant(adminSvc, logger))
			protected.POST("/tenants/:id/suspend", handleSuspendTenant(adminSvc, logger))

			// Tenant configuration
			protected.GET("/tenants/:id/config", handleGetTenantConfig(adminSvc, logger))
			protected.PUT("/tenants/:id/config", handleUpdateTenantConfig(adminSvc, logger))
			protected.GET("/tenants/:id/usage", handleGetTenantUsage(adminSvc, logger))

			// Role management
			protected.GET("/roles", handleListRoles(roleSvc, logger))
			protected.POST("/roles", handleCreateRole(roleSvc, logger))
			protected.GET("/roles/:id", handleGetRole(roleSvc, logger))
			protected.PUT("/roles/:id", handleUpdateRole(roleSvc, logger))
			protected.DELETE("/roles/:id", handleDeleteRole(roleSvc, logger))
			protected.GET("/roles/:id/permissions", handleGetRolePermissions(roleSvc, logger))
			protected.POST("/roles/:id/permissions", handleGrantPermission(roleSvc, logger))
			protected.DELETE("/roles/:id/permissions/:permission_id", handleRevokePermission(roleSvc, logger))

			// User role assignments
			protected.GET("/users/:user_id/roles", handleGetUserRoles(roleSvc, logger))
			protected.POST("/users/:user_id/roles/:role_id", handleAssignUserRole(roleSvc, logger))
			protected.DELETE("/users/:user_id/roles/:role_id", handleRevokeUserRole(roleSvc, logger))

			// System stats (admin only)
			admin := protected.Group("/admin")
			admin.Use(middleware.RequireRole("super_admin"))
			{
				admin.GET("/stats", handleSystemStats(adminSvc, logger))
				admin.GET("/permissions", handleListPermissions(roleSvc, logger))
			}
		}
	}

	return r
}

// validateUUID validates a UUID string
func validateUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// validateEmail validates an email address
func validateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// validateUsername validates a username (alphanumeric, underscore, hyphen)
func validateUsername(username string) bool {
	if len(username) < 3 || len(username) > 64 {
		return false
	}
	for _, r := range username {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

// sanitizeInput removes potentially dangerous characters
func sanitizeInput(s string) string {
	// Basic sanitization - remove null bytes and control characters
	result := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= 32 && r != 127 { // Keep printable ASCII, remove DEL
			result = append(result, r)
		}
	}
	return string(result)
}

// validateTenantRequest validates tenant creation/update requests
func validateTenantRequest(req admin.Tenant) error {
	if req.Name == "" {
		return fmt.Errorf("name: required")
	}
	if len(req.Name) > 255 {
		return fmt.Errorf("name: too long (max 255 characters)")
	}
	if req.Status != "" && req.Status != "active" && req.Status != "suspended" && req.Status != "deleted" {
		return fmt.Errorf("status: invalid value")
	}
	return nil
}

// validateRoleRequest validates role creation/update requests
func validateRoleRequest(req auth.Role) error {
	if req.Name == "" {
		return fmt.Errorf("name: required")
	}
	if !validateUsername(req.Name) {
		return fmt.Errorf("name: must be alphanumeric with underscore/hyphen, 3-64 characters")
	}
	if req.DisplayName == "" {
		return fmt.Errorf("display_name: required")
	}
	if len(req.DisplayName) > 255 {
		return fmt.Errorf("display_name: too long (max 255 characters)")
	}
	return nil
}

// Handler functions - Tenants

func handleListTenants(svc *admin.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := getIntQuery(c, "limit", 50)
		offset := getIntQuery(c, "offset", 0)

		tenants, total, err := svc.ListTenants(c.Request.Context(), limit, offset)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to list tenants")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list tenants"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"tenants": tenants,
			"total":   total,
			"limit":   limit,
			"offset":  offset,
		})
	}
}

func handleCreateTenant(svc *admin.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req admin.Tenant
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		// Additional validation
		if err := validateTenantRequest(req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
			return
		}

		// Sanitize input
		req.Name = sanitizeInput(req.Name)

		// Ensure tenant_id matches authenticated user's tenant (for multi-tenancy)
		userTenantID, exists := c.Get("tenant_id")
		if exists {
			if userTenantIDStr, ok := userTenantID.(string); ok {
				if req.ID != uuid.Nil && req.ID.String() != userTenantIDStr {
					c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Cannot create tenant for different organization"}})
					return
				}
			}
		}

		if err := svc.CreateTenant(c.Request.Context(), &req); err != nil {
			logger.Error().Err(err).Msg("Failed to create tenant")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create tenant"}})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"tenant": req})
	}
}

func handleGetTenant(svc *admin.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid tenant ID"}})
			return
		}

		tenant, err := svc.GetTenant(c.Request.Context(), id)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get tenant")
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Tenant not found"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"tenant": tenant})
	}
}

func handleUpdateTenant(svc *admin.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid tenant ID"}})
			return
		}

		var req admin.Tenant
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		req.ID = id
		if err := svc.UpdateTenant(c.Request.Context(), &req); err != nil {
			logger.Error().Err(err).Msg("Failed to update tenant")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to update tenant"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"tenant": req})
	}
}

func handleDeleteTenant(svc *admin.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid tenant ID"}})
			return
		}

		if err := svc.DeleteTenant(c.Request.Context(), id); err != nil {
			logger.Error().Err(err).Msg("Failed to delete tenant")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to delete tenant"}})
			return
		}

		logger.Info().Str("tenant_id", id.String()).Msg("Tenant deleted")
		c.JSON(http.StatusOK, gin.H{"message": "Tenant deleted"})
	}
}

func handleSuspendTenant(svc *admin.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid tenant ID"}})
			return
		}

		var req struct {
			Reason string `json:"reason"`
		}
		c.ShouldBindJSON(&req)

		if err := svc.SuspendTenant(c.Request.Context(), id, req.Reason); err != nil {
			logger.Error().Err(err).Msg("Failed to suspend tenant")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to suspend tenant"}})
			return
		}

		logger.Info().Str("tenant_id", id.String()).Str("reason", req.Reason).Msg("Tenant suspended")
		c.JSON(http.StatusOK, gin.H{"message": "Tenant suspended"})
	}
}

func handleGetTenantConfig(svc *admin.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid tenant ID"}})
			return
		}

		config, err := svc.GetTenantConfig(c.Request.Context(), id)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get tenant config")
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Tenant not found"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"config": config})
	}
}

func handleUpdateTenantConfig(svc *admin.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid tenant ID"}})
			return
		}

		var req admin.TenantConfig
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		if err := svc.UpdateTenantConfig(c.Request.Context(), id, &req); err != nil {
			logger.Error().Err(err).Msg("Failed to update tenant config")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to update config"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Config updated"})
	}
}

func handleGetTenantUsage(svc *admin.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid tenant ID"}})
			return
		}

		usage, err := svc.GetTenantUsage(c.Request.Context(), id)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get tenant usage")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get usage"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"usage": usage})
	}
}

func handleSystemStats(svc *admin.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats, err := svc.GetSystemStats(c.Request.Context())
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get system stats")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get stats"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"stats": stats})
	}
}

// Handler functions - Roles

func handleListRoles(svc *auth.RoleService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, _ := c.Get("tenant_id")
		tenantIDUUID, err := uuid.Parse(tenantID.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		limit := getIntQuery(c, "limit", 50)
		offset := getIntQuery(c, "offset", 0)

		// Use repository directly since RoleService doesn't expose List
		roles, err := svc.GetRepo().List(c.Request.Context(), tenantIDUUID, limit, offset)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to list roles")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list roles"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"roles": roles})
	}
}

func handleCreateRole(svc *auth.RoleService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req auth.Role
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		// Validate request
		if err := validateRoleRequest(req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
			return
		}

		// Sanitize input
		req.Name = sanitizeInput(req.Name)
		req.DisplayName = sanitizeInput(req.DisplayName)
		req.Description = sanitizeInput(req.Description)

		// Get authorization context
		userIDStr, _ := c.Get("user_id")
		tenantIDStr, _ := c.Get("tenant_id")

		userID, _ := uuid.Parse(userIDStr.(string))
		tenantID, _ := uuid.Parse(tenantIDStr.(string))

		// Ensure tenant_id matches authenticated user
		if req.TenantID != tenantID {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Cannot create role for different tenant"}})
			return
		}

		authCtx := auth.AuthorizationContext{
			UserID:   userID,
			TenantID: tenantID,
		}

		if err := svc.CreateRole(c.Request.Context(), &req, authCtx); err != nil {
			logger.Error().Err(err).Msg("Failed to create role")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create role"}})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"role": req})
	}
}

func handleGetRole(svc *auth.RoleService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid role ID"}})
			return
		}

		role, err := svc.GetRepo().GetByID(c.Request.Context(), id)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get role")
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Role not found"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"role": role})
	}
}

func handleUpdateRole(svc *auth.RoleService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid role ID"}})
			return
		}

		var req auth.Role
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		// Get authorization context
		userIDStr, _ := c.Get("user_id")
		tenantIDStr, _ := c.Get("tenant_id")

		userID, _ := uuid.Parse(userIDStr.(string))
		tenantID, _ := uuid.Parse(tenantIDStr.(string))

		authCtx := auth.AuthorizationContext{
			UserID:   userID,
			TenantID: tenantID,
		}

		req.ID = id
		if err := svc.UpdateRole(c.Request.Context(), &req, authCtx); err != nil {
			logger.Error().Err(err).Msg("Failed to update role")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to update role"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"role": req})
	}
}

func handleDeleteRole(svc *auth.RoleService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid role ID"}})
			return
		}

		// Get authorization context
		userIDStr, _ := c.Get("user_id")
		tenantIDStr, _ := c.Get("tenant_id")

		userID, _ := uuid.Parse(userIDStr.(string))
		tenantID, _ := uuid.Parse(tenantIDStr.(string))

		authCtx := auth.AuthorizationContext{
			UserID:   userID,
			TenantID: tenantID,
		}

		if err := svc.DeleteRole(c.Request.Context(), id, authCtx); err != nil {
			logger.Error().Err(err).Msg("Failed to delete role")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to delete role"}})
			return
		}

		logger.Info().Str("role_id", id.String()).Msg("Role deleted")
		c.JSON(http.StatusOK, gin.H{"message": "Role deleted"})
	}
}

func handleGetRolePermissions(svc *auth.RoleService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid role ID"}})
			return
		}

		permissions, err := svc.GetRepo().GetPermissions(c.Request.Context(), id)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get role permissions")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get permissions"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"permissions": permissions})
	}
}

func handleGrantPermission(svc *auth.RoleService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid role ID"}})
			return
		}

		var req struct {
			Resource string `json:"resource" binding:"required"`
			Action   string `json:"action" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		// Get authorization context
		userIDStr, _ := c.Get("user_id")
		tenantIDStr, _ := c.Get("tenant_id")

		userID, _ := uuid.Parse(userIDStr.(string))
		tenantID, _ := uuid.Parse(tenantIDStr.(string))

		authCtx := auth.AuthorizationContext{
			UserID:   userID,
			TenantID: tenantID,
		}

		if err := svc.AssignPermission(c.Request.Context(), id, req.Resource, req.Action, authCtx); err != nil {
			logger.Error().Err(err).Msg("Failed to grant permission")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to grant permission"}})
			return
		}

		logger.Info().
			Str("role_id", id.String()).
			Str("resource", req.Resource).
			Str("action", req.Action).
			Msg("Permission granted")

		c.JSON(http.StatusOK, gin.H{"message": "Permission granted"})
	}
}

func handleRevokePermission(svc *auth.RoleService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid role ID"}})
			return
		}

		permissionID, err := uuid.Parse(c.Param("permission_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid permission ID"}})
			return
		}

		if err := svc.GetRepo().RevokePermission(c.Request.Context(), roleID, permissionID); err != nil {
			logger.Error().Err(err).Msg("Failed to revoke permission")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to revoke permission"}})
			return
		}

		logger.Info().
			Str("role_id", roleID.String()).
			Str("permission_id", permissionID.String()).
			Msg("Permission revoked")

		c.JSON(http.StatusOK, gin.H{"message": "Permission revoked"})
	}
}

func handleGetUserRoles(svc *auth.RoleService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := uuid.Parse(c.Param("user_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid user ID"}})
			return
		}

		roles, err := svc.GetRepo().GetUserRoles(c.Request.Context(), userID)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get user roles")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get roles"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"roles": roles})
	}
}

func handleAssignUserRole(svc *auth.RoleService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := uuid.Parse(c.Param("user_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid user ID"}})
			return
		}

		roleID, err := uuid.Parse(c.Param("role_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid role ID"}})
			return
		}

		assignedBy, _ := c.Get("user_id")
		assignedByID, _ := uuid.Parse(assignedBy.(string))

		if err := svc.AssignUserToRole(c.Request.Context(), userID, roleID, assignedByID); err != nil {
			logger.Error().Err(err).Msg("Failed to assign role")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to assign role"}})
			return
		}

		logger.Info().
			Str("user_id", userID.String()).
			Str("role_id", roleID.String()).
			Msg("Role assigned to user")

		c.JSON(http.StatusOK, gin.H{"message": "Role assigned"})
	}
}

func handleRevokeUserRole(svc *auth.RoleService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := uuid.Parse(c.Param("user_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid user ID"}})
			return
		}

		roleID, err := uuid.Parse(c.Param("role_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid role ID"}})
			return
		}

		if err := svc.RevokeRoleFromUser(c.Request.Context(), userID, roleID); err != nil {
			logger.Error().Err(err).Msg("Failed to revoke role")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to revoke role"}})
			return
		}

		logger.Info().
			Str("user_id", userID.String()).
			Str("role_id", roleID.String()).
			Msg("Role revoked from user")

		c.JSON(http.StatusOK, gin.H{"message": "Role revoked"})
	}
}

func handleListPermissions(svc *auth.RoleService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		permissions, err := svc.GetRepo().ListAllPermissions(c.Request.Context())
		if err != nil {
			logger.Error().Err(err).Msg("Failed to list permissions")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list permissions"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"permissions": permissions})
	}
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
