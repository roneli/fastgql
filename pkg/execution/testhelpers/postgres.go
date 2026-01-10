package testhelpers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// GetTestPostgresPool returns a pgxpool.Pool for testing.
// It either connects to the database specified by TEST_DATABASE_URL
// or spins up a new Postgres container using testcontainers.
func GetTestPostgresPool(ctx context.Context) (*pgxpool.Pool, func(), error) {
	connStr := os.Getenv("TEST_DATABASE_URL")
	cleanup := func() {}

	if connStr == "" {
		// No connection string provided, spin up a test container
		_, b, _, _ := runtime.Caller(0)
		basepath := filepath.Dir(b)
		// Navigate up from pkg/execution/testhelpers to root
		rootDir := filepath.Join(basepath, "..", "..", "..")
		initScriptPath := filepath.Join(rootDir, ".github", "workflows", "data", "init.sql")

		// Check for Ryuk disabled env var or try to auto-detect if we should disable it
		// This is often needed in environments with SELinux or restricted Docker socket access
		// In many CI/local setups (like Fedora), Ryuk might fail due to socket permissions.
		// Users can set TESTCONTAINERS_RYUK_DISABLED=true to skip it.
		// Alternatively, setting TESTCONTAINERS_RYUK_CONTAINER_PRIVILEGED=true often fixes it by running Ryuk as privileged.
		if err := os.Setenv("TESTCONTAINERS_RYUK_CONTAINER_PRIVILEGED", "true"); err != nil {
			// Log but don't fail - this is a best-effort optimization
			fmt.Printf("warning: failed to set TESTCONTAINERS_RYUK_CONTAINER_PRIVILEGED: %v\n", err)
		}
		pgContainer, err := postgres.Run(ctx,
			"postgres:16",
			postgres.WithInitScripts(initScriptPath),
			postgres.WithDatabase("testdb"),
			postgres.WithUsername("testuser"),
			postgres.WithPassword("testpass"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(30*time.Second)),
		)
		if err != nil {
			return nil, cleanup, fmt.Errorf("failed to start postgres container: %w", err)
		}

		cleanup = func() {
			if err := pgContainer.Terminate(ctx); err != nil {
				fmt.Printf("failed to terminate container: %s\n", err)
			}
		}

		connStr, err = pgContainer.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			cleanup() // Ensure container is cleaned up if getting connection string fails
			return nil, nil, fmt.Errorf("failed to get connection string: %w", err)
		}
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to create pool: %w", err)
	}

	return pool, func() {
		pool.Close()
		cleanup()
	}, nil
}
