package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ngenohkevin/kuiper_admin/internal/cache"
)

type DB struct {
	Pool  *pgxpool.Pool
	Cache *cache.Cache
}

// New creates a new database connection
func New() (*DB, error) {
	// Get database connection string from environment variables
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable not set")
	}

	// Run migrations only if enabled
	if os.Getenv("RUN_MIGRATIONS") == "true" {
		if err := runMigrations(dbURL); err != nil {
			log.Printf("Warning: Migration error: %v", err)
			// Continue anyway, as migrations may have already been applied
		}
	}

	// Set up connection pool
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing database config: %w", err)
	}

	// Configure SSL/TLS
	config.ConnConfig.TLSConfig = nil // Let pgx handle the TLS config automatically

	// Disable prepared statements for PgBouncer/Supabase compatibility
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	// Set connection pool settings
	config.MaxConns = 20
	config.MinConns = 5
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 10 * time.Minute

	// Create the connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	log.Println("Successfully connected to the database")
	return &DB{
		Pool:  pool,
		Cache: cache.New(),
	}, nil
}

// Close closes the database connection
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// RunMigrations runs the database migrations
func runMigrations(dbURL string) error {
	m, err := migrate.New("file://migrations", dbURL)
	if err != nil {
		return fmt.Errorf("error creating migration instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("error applying migrations: %w", err)
	}

	return nil
}
