package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository is the base interface for all repositories
type Repository interface {
	// WithTx returns a repository scoped to a transaction
	WithTx(tx *sqlx.Tx) Repository
}

// UnitOfWork coordinates multiple repository operations within a single transaction
type UnitOfWork interface {
	// Begin starts a new unit of work (transaction)
	Begin(ctx context.Context) error
	// Commit commits all changes
	Commit() error
	// Rollback rolls back all changes
	Rollback() error
	// Tx returns the underlying transaction
	Tx() *sqlx.Tx
}

// CRUDRepository provides basic CRUD operations for any entity
type CRUDRepository[T any] interface {
	Create(ctx context.Context, entity *T) error
	GetByID(ctx context.Context, id uuid.UUID) (*T, error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// TenantScopedRepository adds tenant isolation to CRUD operations
type TenantScopedRepository[T any, F any] interface {
	CRUDRepository[T]
	List(ctx context.Context, tenantID uuid.UUID, filter F, limit, offset int) ([]T, int, error)
}

// Pagination holds pagination parameters
type Pagination struct {
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	SortBy string `json:"sort_by,omitempty"`
	Order  string `json:"order,omitempty"` // "asc" or "desc"
}

// PaginatedResult wraps a list result with total count
type PaginatedResult[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
	Pagination
}

// DefaultPagination returns safe defaults
func DefaultPagination() Pagination {
	return Pagination{
		Limit:  20,
		Offset: 0,
		SortBy: "created_at",
		Order:  "desc",
	}
}

// ClampPagination ensures pagination values are within safe bounds
func ClampPagination(p Pagination) Pagination {
	if p.Limit <= 0 {
		p.Limit = 20
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	if p.Order != "asc" && p.Order != "desc" {
		p.Order = "desc"
	}
	return p
}
