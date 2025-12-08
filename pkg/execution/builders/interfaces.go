package builders

import "context"

// Builder generates queries for a specific database dialect.
// Each database implementation (SQL, MongoDB, Cypher, etc.) implements this interface.
type Builder interface {
	// Query generates a read query from the GraphQL field
	Query(field Field) (Query, error)
	// Create generates an insert/create mutation query
	Create(field Field) (Query, error)
	// Update generates an update mutation query
	Update(field Field) (Query, error)
	// Delete generates a delete mutation query
	Delete(field Field) (Query, error)
	// Capabilities returns what this database supports
	Capabilities() Capabilities
}

// Query is the output of a builder - wraps database-specific query objects.
// Implementations include SQLQuery (string + args), MongoDB Pipeline, Cypher string, etc.
type Query interface {
	// Native returns the dialect-specific query object.
	// For SQL: SQLQuery{sql, args}
	// For MongoDB: mongo.Pipeline
	// For Cypher: string
	Native() any
	// String returns a human-readable representation for logging/debugging
	String() string
}

// Capabilities declares what features a database supports.
// This allows the framework to adapt behavior based on database limitations.
type Capabilities struct {
	// SupportsJoins indicates if the database can perform JOIN operations.
	// False for Cassandra, true for SQL databases.
	SupportsJoins bool

	// SupportsReturning indicates if mutations can return affected rows.
	// PostgreSQL supports INSERT...RETURNING, some databases don't.
	SupportsReturning bool

	// SupportsTransactions indicates if the database supports ACID transactions.
	SupportsTransactions bool

	// MaxRelationDepth limits nested relation queries.
	// -1 = unlimited, 0 = no relations (use DataLoader pattern), N = max depth
	MaxRelationDepth int
}

// Driver executes queries against a database connection.
type Driver interface {
	// Execute runs the query and scans results into dest
	Execute(ctx context.Context, query Query, dest any) error
	// Close closes the database connection
	Close() error
	// Dialect returns the database dialect name (e.g., "postgres", "mongodb")
	Dialect() string
}
