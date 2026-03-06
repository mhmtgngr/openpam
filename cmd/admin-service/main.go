package main

import (
	"github.com/openpam/openpam/cmd/admin-service/handlers"
	"github.com/openpam/openpam/internal/admin"
	"github.com/openpam/openpam/internal/auth"
	"github.com/openpam/openpam/internal/middleware"
	"github.com/openpam/openpam/internal/server"
)

func main() {
	deps := server.Bootstrap("admin-service")

	// Initialize domain services
	adminSvc := admin.NewService(deps.DB.DB, deps.Cache, deps.Logger)
	roleRepo := auth.NewRoleRepository(deps.DB.DB, deps.Logger)
	roleSvc := auth.NewRoleService(roleRepo, deps.Logger)

	// Create handlers
	tenantHandler := handlers.NewTenantHandler(adminSvc, deps.Logger)
	roleHandler := handlers.NewRoleHandler(roleSvc, deps.Logger)

	// Setup router with standard middleware
	router := server.NewRouter(deps)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		protected := v1.Group("")
		protected.Use(middleware.Auth())
		{
			// Pluggable handler registration — each handler owns its routes
			tenantHandler.RegisterRoutes(protected)
			roleHandler.RegisterRoutes(protected)

			// System admin endpoints
			adminGroup := protected.Group("/admin")
			adminGroup.Use(middleware.RequireRole("super_admin"))
			{
				adminGroup.GET("/stats", handlers.SystemStats(adminSvc, deps.Logger))
				adminGroup.GET("/permissions", handlers.ListPermissions(roleSvc, deps.Logger))
			}
		}
	}

	server.ListenAndServe(router, deps)
}
