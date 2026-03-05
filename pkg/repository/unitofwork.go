package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// DBUnitOfWork implements UnitOfWork using a PostgreSQL transaction
type DBUnitOfWork struct {
	db     *sqlx.DB
	tx     *sqlx.Tx
	logger zerolog.Logger
}

// NewUnitOfWork creates a new unit of work
func NewUnitOfWork(db *sqlx.DB, logger zerolog.Logger) *DBUnitOfWork {
	return &DBUnitOfWork{db: db, logger: logger}
}

// Begin starts a new transaction
func (uow *DBUnitOfWork) Begin(ctx context.Context) error {
	tx, err := uow.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("unitofwork.Begin: %w", err)
	}
	uow.tx = tx
	return nil
}

// Commit commits the transaction
func (uow *DBUnitOfWork) Commit() error {
	if uow.tx == nil {
		return fmt.Errorf("unitofwork.Commit: no active transaction")
	}
	err := uow.tx.Commit()
	uow.tx = nil
	if err != nil {
		return fmt.Errorf("unitofwork.Commit: %w", err)
	}
	return nil
}

// Rollback rolls back the transaction
func (uow *DBUnitOfWork) Rollback() error {
	if uow.tx == nil {
		return nil
	}
	err := uow.tx.Rollback()
	uow.tx = nil
	if err != nil {
		return fmt.Errorf("unitofwork.Rollback: %w", err)
	}
	return nil
}

// Tx returns the underlying transaction
func (uow *DBUnitOfWork) Tx() *sqlx.Tx {
	return uow.tx
}

// Execute runs a function within a transaction, handling commit/rollback
func (uow *DBUnitOfWork) Execute(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	if err := uow.Begin(ctx); err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = uow.Rollback()
			panic(p)
		}
	}()

	if err := fn(uow.tx); err != nil {
		if rbErr := uow.Rollback(); rbErr != nil {
			uow.logger.Error().Err(rbErr).Msg("Failed to rollback transaction")
		}
		return err
	}

	return uow.Commit()
}
