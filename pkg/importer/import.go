package importer

import (
	"context"
)

// Source represents a database source that can be introspected
type Source interface {
	// Connect establishes a connection to the database
	Connect(ctx context.Context, connStr string) error
	// Introspect analyzes the database schema and returns a Schema structure
	Introspect(ctx context.Context, options IntrospectOptions) (*Schema, error)
	// Close closes the database connection
	Close() error
}

// IntrospectOptions contains options for schema introspection
type IntrospectOptions struct {
	// SchemaName is the database schema name to introspect (e.g., "public", "app")
	// If empty, uses the default schema for the database
	SchemaName string
	// Tables is a list of table names to include. If empty, all tables are included
	Tables []string
	// GenerateQueries indicates if Query fields should be generated
	GenerateQueries bool
	// GenerateFilters indicates if @generateFilterInput directives should be added
	GenerateFilters bool
}

