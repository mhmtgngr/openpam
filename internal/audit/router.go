// Package audit provides router configuration for analytics endpoints
package audit

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/openpam/openpam/internal/audit/middleware"
	"github.com/rs/zerolog"
)

// AnalyticsHandler defines the interface for analytics HTTP handlers
type AnalyticsHandler interface {
	// Dashboard
	GetDashboardSummary(*gin.Context)

	// Compliance Reports
	CreateComplianceReport(*gin.Context)
	GetComplianceReport(*gin.Context)
	ListComplianceReports(*gin.Context)
	UpdateComplianceReportStatus(*gin.Context)
	DeleteComplianceReport(*gin.Context)

	// Compliance Exceptions
	CreateComplianceException(*gin.Context)
	ListComplianceExceptions(*gin.Context)
	UpdateComplianceExceptionStatus(*gin.Context)
	GetExpiringExceptions(*gin.Context)
	GetComplianceException(*gin.Context)
	DeleteComplianceException(*gin.Context)

	// Anomalies
	CreateAnomaly(*gin.Context)
	GetAnomaly(*gin.Context)
	ListAnomalies(*gin.Context)
	UpdateAnomaly(*gin.Context)
	DeleteAnomaly(*gin.Context)
	GetAnomalyStats(*gin.Context)

	// SSH Key Analytics
	GetSSHKeyAnalytics(*gin.Context)
	ListSSHKeyAnalytics(*gin.Context)
	GetSSHKeyUsageSummary(*gin.Context)
	GetMostUsedSSHKeys(*gin.Context)
	GetAnomalousSSHKeys(*gin.Context)

	// Command Blacklist
	CreateCommandBlacklist(*gin.Context)
	GetCommandBlacklist(*gin.Context)
	ListCommandBlacklist(*gin.Context)
	UpdateCommandBlacklist(*gin.Context)
	DeleteCommandBlacklist(*gin.Context)
	GetBlacklistStats(*gin.Context)
	EnableCommandBlacklist(*gin.Context)
	DisableCommandBlacklist(*gin.Context)
	CheckCommandAgainstBlacklist(*gin.Context)

	// Cache
	InvalidateCache(*gin.Context)

	// Export/Admin
	ExportComplianceReport(*gin.Context)
	ExportAnomalies(*gin.Context)
	GetAdminStats(*gin.Context)
	RunAnomalyDetection(*gin.Context)
	GenerateComplianceReport(*gin.Context)
	GetCacheStats(*gin.Context)
}

// RegisterAnalyticsRoutes registers all analytics API routes
func RegisterAnalyticsRoutes(r *gin.RouterGroup, handler AnalyticsHandler, logger zerolog.Logger) {

	// Get middleware configuration
	config := middleware.DefaultAnalyticsConfig()

	// Apply common middleware to all analytics routes
	analyticsGroup := r.Group("/analytics")
	analyticsGroup.Use(
		middleware.RequireTenantID(),
		middleware.CorrelationID(),
		middleware.RequestLogging(logger),
	)

	// ============================================================================
	// Dashboard Routes
	// ============================================================================
	analyticsGroup.GET("/dashboard", handler.GetDashboardSummary)

	// ============================================================================
	// Compliance Report Routes
	// ============================================================================
	complianceGroup := analyticsGroup.Group("/compliance")
	{
		reportsGroup := complianceGroup.Group("/reports")
		reportsGroup.Use(middleware.Pagination(config), middleware.DateRange())

		reportsGroup.GET("", handler.ListComplianceReports)
		reportsGroup.POST("", middleware.RequireAdmin(), handler.CreateComplianceReport)
		reportsGroup.GET("/:id", handler.GetComplianceReport)
		reportsGroup.PATCH("/:id/status", middleware.RequireAdmin(), handler.UpdateComplianceReportStatus)
		reportsGroup.DELETE("/:id", middleware.RequireAdmin(), handler.DeleteComplianceReport)

		// Framework-specific routes
		reportsGroup.GET("/framework/:framework", middleware.ValidateFramework(), handler.ListComplianceReports)

		// Compliance Exception Routes
		exceptionsGroup := complianceGroup.Group("/exceptions")
		exceptionsGroup.Use(middleware.Pagination(config))

		exceptionsGroup.GET("", handler.ListComplianceExceptions)
		exceptionsGroup.POST("", handler.CreateComplianceException)
		exceptionsGroup.GET("/:id", handler.GetComplianceException)
		exceptionsGroup.PATCH("/:id/status", handler.UpdateComplianceExceptionStatus)
		exceptionsGroup.DELETE("/:id", handler.DeleteComplianceException)

		// Expiring exceptions
		exceptionsGroup.GET("/expiring", handler.GetExpiringExceptions)
	}

	// ============================================================================
	// Anomaly Detection Routes
	// ============================================================================
	anomaliesGroup := analyticsGroup.Group("/anomalies")
	anomaliesGroup.Use(
		middleware.Pagination(config),
		middleware.DateRange(),
		middleware.QueryTimeout(config.QueryTimeout),
	)
	{
		anomaliesGroup.GET("", handler.ListAnomalies)
		anomaliesGroup.POST("", handler.CreateAnomaly)
		anomaliesGroup.GET("/:id", handler.GetAnomaly)
		anomaliesGroup.PATCH("/:id", handler.UpdateAnomaly)
		anomaliesGroup.DELETE("/:id", handler.DeleteAnomaly)

		// Anomaly Statistics
		anomaliesGroup.GET("/stats", handler.GetAnomalyStats)

		// Open anomalies (high priority)
		anomaliesGroup.GET("/open", middleware.ValidateSeverity(), handler.ListAnomalies)

		// User-specific anomalies
		anomaliesGroup.GET("/user/:user_id", handler.ListAnomalies)
	}

	// ============================================================================
	// SSH Key Analytics Routes
	// ============================================================================
	sshKeysGroup := analyticsGroup.Group("/ssh-keys")
	sshKeysGroup.Use(middleware.DateRange())
	{
		sshKeysGroup.GET("/analytics", handler.ListSSHKeyAnalytics)
		sshKeysGroup.GET("/:id/analytics", handler.GetSSHKeyAnalytics)
		sshKeysGroup.GET("/:id/summary", handler.GetSSHKeyUsageSummary)
		sshKeysGroup.GET("/most-used", handler.GetMostUsedSSHKeys)
		sshKeysGroup.GET("/anomalous", handler.GetAnomalousSSHKeys)
	}

	// ============================================================================
	// Command Blacklist Routes
	// ============================================================================
	blacklistGroup := analyticsGroup.Group("/blacklist")
	blacklistGroup.Use(
		middleware.Pagination(config),
		middleware.ValidatePatternType(),
	)
	{
		blacklistGroup.GET("", handler.ListCommandBlacklist)
		blacklistGroup.POST("", middleware.RequireAdmin(), handler.CreateCommandBlacklist)
		blacklistGroup.GET("/:id", handler.GetCommandBlacklist)
		blacklistGroup.PATCH("/:id", handler.UpdateCommandBlacklist)
		blacklistGroup.DELETE("/:id", handler.DeleteCommandBlacklist)
		blacklistGroup.GET("/stats", handler.GetBlacklistStats)

		// Enable/Disable
		blacklistGroup.POST("/:id/enable", middleware.RequireAdmin(), handler.EnableCommandBlacklist)
		blacklistGroup.POST("/:id/disable", middleware.RequireAdmin(), handler.DisableCommandBlacklist)

		// Check command against blacklist
		blacklistGroup.POST("/check", handler.CheckCommandAgainstBlacklist)
	}

	// ============================================================================
	// Cache Management Routes
	// ============================================================================
	cacheGroup := analyticsGroup.Group("/cache")
	cacheGroup.Use(middleware.RequireAdmin())
	{
		cacheGroup.POST("/invalidate", handler.InvalidateCache)
		cacheGroup.GET("/stats", handler.GetCacheStats)
	}

	// ============================================================================
	// Summary and Export Routes
	// ============================================================================
	analyticsGroup.GET("/summary", handler.GetDashboardSummary)
	analyticsGroup.GET("/export/compliance", handler.ExportComplianceReport)
	analyticsGroup.GET("/export/anomalies", handler.ExportAnomalies)

	// ============================================================================
	// Admin Routes (require admin role)
	// ============================================================================
	adminGroup := analyticsGroup.Group("/admin")
	adminGroup.Use(middleware.RequireAdmin())
	{
		adminGroup.GET("/stats", handler.GetAdminStats)
		adminGroup.POST("/anomaly-detection/run", handler.RunAnomalyDetection)
		adminGroup.POST("/compliance/generate", handler.GenerateComplianceReport)
	}
}

// =============================================================================
// Router Helper for Complete API Setup
// =============================================================================

// SetupAnalyticsRouter creates and configures the complete analytics router
func SetupAnalyticsRouter(r *gin.Engine, handler AnalyticsHandler, logger zerolog.Logger) {
	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Apply auth middleware to all routes
		// Note: Auth middleware should be applied by the caller
		// Protected routes group
		protected := v1.Group("")
		{
			// Register analytics routes
			RegisterAnalyticsRoutes(protected.Group("/audit"), handler, logger)
		}
	}
}

// =============================================================================
// Analytics-specific Route Groups
// =============================================================================

// RouteGroupConfig holds configuration for a route group
type RouteGroupConfig struct {
	Path         string
	Middlewares  []gin.HandlerFunc
	Handler      string
	Methods      []string
	RequireAuth  bool
	RequireAdmin bool
}

// AnalyticsRouteConfig holds all analytics route configurations
var AnalyticsRoutes = []RouteGroupConfig{
	// Dashboard
	{Path: "/dashboard", Handler: "GetDashboardSummary", Methods: []string{"GET"}},

	// Compliance Reports
	{Path: "/compliance/reports", Handler: "ListComplianceReports", Methods: []string{"GET"}},
	{Path: "/compliance/reports", Handler: "CreateComplianceReport", Methods: []string{"POST"}, RequireAdmin: true},
	{Path: "/compliance/reports/:id", Handler: "GetComplianceReport", Methods: []string{"GET"}},
	{Path: "/compliance/reports/:id/status", Handler: "UpdateComplianceReportStatus", Methods: []string{"PATCH"}, RequireAdmin: true},

	// Compliance Exceptions
	{Path: "/compliance/exceptions", Handler: "ListComplianceExceptions", Methods: []string{"GET"}},
	{Path: "/compliance/exceptions", Handler: "CreateComplianceException", Methods: []string{"POST"}},
	{Path: "/compliance/exceptions/:id", Handler: "GetComplianceException", Methods: []string{"GET"}},
	{Path: "/compliance/exceptions/:id/status", Handler: "UpdateComplianceExceptionStatus", Methods: []string{"PATCH"}, RequireAdmin: true},

	// Anomalies
	{Path: "/anomalies", Handler: "ListAnomalies", Methods: []string{"GET"}},
	{Path: "/anomalies", Handler: "CreateAnomaly", Methods: []string{"POST"}},
	{Path: "/anomalies/:id", Handler: "GetAnomaly", Methods: []string{"GET"}},
	{Path: "/anomalies/:id", Handler: "UpdateAnomaly", Methods: []string{"PATCH"}},
	{Path: "/anomalies/stats", Handler: "GetAnomalyStats", Methods: []string{"GET"}},

	// SSH Key Analytics
	{Path: "/ssh-keys/analytics", Handler: "ListSSHKeyAnalytics", Methods: []string{"GET"}},
	{Path: "/ssh-keys/:id/analytics", Handler: "GetSSHKeyAnalytics", Methods: []string{"GET"}},

	// Command Blacklist
	{Path: "/blacklist", Handler: "ListCommandBlacklist", Methods: []string{"GET"}},
	{Path: "/blacklist", Handler: "CreateCommandBlacklist", Methods: []string{"POST"}, RequireAdmin: true},
	{Path: "/blacklist/:id", Handler: "GetCommandBlacklist", Methods: []string{"GET"}},
	{Path: "/blacklist/:id", Handler: "UpdateCommandBlacklist", Methods: []string{"PATCH"}, RequireAdmin: true},
	{Path: "/blacklist/:id", Handler: "DeleteCommandBlacklist", Methods: []string{"DELETE"}, RequireAdmin: true},
	{Path: "/blacklist/stats", Handler: "GetBlacklistStats", Methods: []string{"GET"}},
}

// GetAnalyticsRoutes returns all analytics route configurations
func GetAnalyticsRoutes() []RouteGroupConfig {
	return AnalyticsRoutes
}

// =============================================================================
// Default Route Configuration
// =============================================================================

// DefaultPaginationLimit is the default limit for paginated queries
const DefaultPaginationLimit = 50

// MaxPaginationLimit is the maximum limit for paginated queries
const MaxPaginationLimit = 1000

// DefaultQueryTimeout is the default timeout for analytics queries
const DefaultQueryTimeout = 30 * time.Second

// DefaultCacheMaxAge is the default cache max age for responses
const DefaultCacheMaxAge = 5 * time.Minute
