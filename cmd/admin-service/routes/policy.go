package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/openpam/openpam/cmd/admin-service/handlers"
	"github.com/openpam/openpam/internal/middleware"
)

// RegisterPolicyRoutes registers policy-related routes
func RegisterPolicyRoutes(r *gin.RouterGroup, policyHandler *handlers.PolicyHandler) {
	// All policy routes require authentication
	policies := r.Group("/policies")
	policies.Use(middleware.Auth())
	{
		// Policy CRUD
		policies.GET("", policyHandler.ListPolicies())
		policies.POST("", policyHandler.CreatePolicy())
		policies.GET("/:id", policyHandler.GetPolicy())
		policies.PUT("/:id", policyHandler.UpdatePolicy())
		policies.DELETE("/:id", policyHandler.DeletePolicy())

		// Policy state management
		policies.POST("/:id/enable", policyHandler.EnablePolicy())
		policies.POST("/:id/disable", policyHandler.DisablePolicy())

		// Policy evaluation
		policies.POST("/evaluate", policyHandler.EvaluatePolicy())

		// Policy evaluation logs
		policies.GET("/evaluations/logs", policyHandler.GetEvaluationLogs())

		// Policy templates
		policies.GET("/templates", policyHandler.GetPolicyTemplates())

		// Command filters
		policies.POST("/command-filters", policyHandler.AddCommandFilter())
	}

	// Approval workflow routes
	approvals := r.Group("/approvals")
	approvals.Use(middleware.Auth())
	{
		approvals.POST("", policyHandler.CreateApprovalRequest())
		approvals.GET("/pending", policyHandler.GetPendingApprovals())
		approvals.POST("/:id/process", policyHandler.ProcessApprovalRequest())
	}
}
