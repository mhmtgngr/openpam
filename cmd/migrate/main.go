package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	dbHost     = flag.String("host", "localhost", "Database host")
	dbPort     = flag.Int("port", 5432, "Database port")
	dbUser     = flag.String("user", "openpam", "Database user")
	dbPassword = flag.String("password", "", "Database password")
	dbName     = flag.String("dbname", "openpam", "Database name")
	sslMode    = flag.String("sslmode", "disable", "SSL mode")
	migrationsDir = flag.String("dir", "./migrations", "Migrations directory")
	dryRun    = flag.Bool("dry-run", false, "Print SQL without executing")
)

func main() {
	flag.Parse()

	// Setup logging
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Info().Msg("OpenPAM Migration Runner")

	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		*dbHost, *dbPort, *dbUser, *dbPassword, *dbName, *sslMode)

	// Connect to database
	var db *sql.DB
	var err error
	if *dryRun {
		log.Warn().Msg("Dry run mode - no changes will be made")
	} else {
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to connect to database")
		}
		defer db.Close()

		if err := db.Ping(); err != nil {
			log.Fatal().Err(err).Msg("Failed to ping database")
		}
		log.Info().Msg("Database connection established")
	}

	// Create schema_migrations table if not exists
	if !*dryRun {
		if err := createMigrationsTable(db); err != nil {
			log.Fatal().Err(err).Msg("Failed to create migrations table")
		}
	}

	// Find all migration files
	migrations, err := findMigrations(*migrationsDir)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to find migrations")
	}

	if len(migrations) == 0 {
		log.Info().Msg("No migrations found")
		return
	}

	log.Info().Int("count", len(migrations)).Msg("Found migrations")

	// Get applied migrations
	var applied map[string]bool
	if !*dryRun {
		applied, err = getAppliedMigrations(db)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get applied migrations")
		}
	} else {
		applied = make(map[string]bool)
	}

	// Run pending migrations
	runCount := 0
	for _, m := range migrations {
		if applied[m.Name] {
			log.Debug().Str("name", m.Name).Msg("Already applied, skipping")
			continue
		}

		log.Info().Str("name", m.Name).Str("path", m.Path).Msg("Running migration")

		content, err := os.ReadFile(m.Path)
		if err != nil {
			log.Error().Err(err).Str("name", m.Name).Msg("Failed to read migration file")
			continue
		}

		if *dryRun {
			fmt.Printf("\n--- Migration: %s ---\n%s\n\n", m.Name, string(content))
			continue
		}

		// Execute migration
		if err := executeMigration(db, content); err != nil {
			log.Error().Err(err).Str("name", m.Name).Msg("Failed to execute migration")
			continue
		}

		// Record migration
		if err := recordMigration(db, m.Name); err != nil {
			log.Error().Err(err).Str("name", m.Name).Msg("Failed to record migration")
			continue
		}

		log.Info().Str("name", m.Name).Msg("Migration completed")
		runCount++
	}

	log.Info().Int("count", runCount).Msg("Migrations completed")
}

type Migration struct {
	Name string
	Path string
}

func findMigrations(dir string) ([]Migration, error) {
	var migrations []Migration

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Only process .up.sql files
		if !strings.HasSuffix(info.Name(), ".up.sql") {
			return nil
		}

		// Skip down migrations
		if strings.Contains(info.Name(), ".down.sql") {
			return nil
		}

		// Create a consistent name for the migration
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		name := strings.ReplaceAll(relPath, "/", "_")

		migrations = append(migrations, Migration{
			Name: name,
			Path: path,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort migrations by name for consistent order
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})

	return migrations, nil
}

func createMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) UNIQUE NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`
	_, err := db.Exec(query)
	return err
}

func getAppliedMigrations(db *sql.DB) (map[string]bool, error) {
	query := `SELECT name FROM schema_migrations`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}

	return applied, rows.Err()
}

func executeMigration(db *sql.DB, content []byte) error {
	// Split by semicolon and execute each statement
	// This is a simple approach - for production, consider using a proper migration library
	statements := splitStatements(string(content))
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			// Check if it's a "already exists" error, which is okay
			if strings.Contains(err.Error(), "already exists") {
				log.Debug().Err(err).Msg("Statement failed (likely already exists), continuing...")
				continue
			}
			return fmt.Errorf("failed to execute statement: %w\nStatement: %s", err, stmt)
		}
	}
	return nil
}

func splitStatements(sql string) []string {
	// Split by semicolon, but handle PL/pgSQL blocks which contain semicolons
	var statements []string
	var current strings.Builder
	inBlock := false

	lines := strings.Split(sql, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for function/trigger body start
		if strings.HasPrefix(trimmed, "CREATE") &&
			(strings.Contains(trimmed, "FUNCTION") || strings.Contains(trimmed, "TRIGGER") || strings.Contains(trimmed, "PROCEDURE")) {
			inBlock = true
		}

		current.WriteString(line)
		current.WriteString("\n")

		// Check for end of function/trigger
		if inBlock && strings.HasPrefix(trimmed, "$") {
			// Toggle block state on dollar quotes
			if strings.Count(trimmed, "$") >= 2 {
				inBlock = false
			}
		}

		// Split on semicolon if not in a block
		if strings.HasSuffix(trimmed, ";") && !inBlock {
			statements = append(statements, current.String())
			current.Reset()
		}
	}

	// Add remaining content
	if current.Len() > 0 {
		statements = append(statements, current.String())
	}

	return statements
}

func recordMigration(db *sql.DB, name string) error {
	query := `INSERT INTO schema_migrations (name) VALUES ($1) ON CONFLICT (name) DO NOTHING`
	_, err := db.Exec(query, name)
	return err
}

// MigrateDatabase is a convenience function for programmatic use
func MigrateDatabase(db *sql.DB, migrationsDir string) error {
	logger := zerolog.New(io.Discard).With().Timestamp().Logger()

	migrations, err := findMigrations(migrationsDir)
	if err != nil {
		return fmt.Errorf("find migrations: %w", err)
	}

	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	applied, err := getAppliedMigrations(db)
	if err != nil {
		return fmt.Errorf("get applied migrations: %w", err)
	}

	for _, m := range migrations {
		if applied[m.Name] {
			continue
		}

		logger.Info().Str("name", m.Name).Msg("Running migration")

		content, err := os.ReadFile(m.Path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", m.Name, err)
		}

		if err := executeMigration(db, content); err != nil {
			return fmt.Errorf("execute migration %s: %w", m.Name, err)
		}

		if err := recordMigration(db, m.Name); err != nil {
			return fmt.Errorf("record migration %s: %w", m.Name, err)
		}
	}

	return nil
}
