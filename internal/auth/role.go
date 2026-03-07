// Role-based access control (RBAC) for OpenPAM.
//
// This subsystem is organized into three files by concern:
//
//   - role_models.go:     Data types (Role, Permission, UserRole, AuthorizationContext)
//   - role_repository.go: RoleRepositoryInterface and its PostgreSQL implementation.
//                         Implement the interface to plug in a different storage backend.
//   - role_service.go:    Business logic and authorization enforcement (RoleService)
//
// The RoleService accepts a RoleRepositoryInterface, making the storage layer
// pluggable for testing (mock implementations) or migration to different datastores.
package auth
