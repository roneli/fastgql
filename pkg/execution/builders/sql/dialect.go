package sql

import (
	"encoding/json"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

// Dialect provides dialect-specific SQL functions.
// Different databases have different syntax for JSON operations.
type Dialect interface {
	// JSONBuildObject creates a JSON object from key-value pairs
	JSONBuildObject(args ...any) exp.SQLFunctionExpression
	// JSONAgg aggregates rows into a JSON array
	JSONAgg(expr exp.Expression) exp.SQLFunctionExpression
	// CoalesceJSON returns a fallback value if the expression is null
	CoalesceJSON(expr exp.Expression, fallback string) exp.SQLFunctionExpression

	// JSON filtering methods
	// JSONPathExists checks if a JSONPath expression matches
	JSONPathExists(col exp.Expression, path string, vars map[string]any) exp.Expression
	// JSONContains checks if JSON contains a value (@> operator)
	JSONContains(col exp.Expression, value string) exp.Expression
	// JSONExtract extracts a value from JSON using a path
	JSONExtract(col exp.Expression, path string) exp.Expression
}

// PostgresDialect implements Dialect for PostgreSQL.
type PostgresDialect struct{}

func (PostgresDialect) JSONBuildObject(args ...any) exp.SQLFunctionExpression {
	return goqu.Func("jsonb_build_object", args...)
}

func (PostgresDialect) JSONAgg(expr exp.Expression) exp.SQLFunctionExpression {
	return goqu.Func("jsonb_agg", expr)
}

func (PostgresDialect) CoalesceJSON(expr exp.Expression, fallback string) exp.SQLFunctionExpression {
	return goqu.COALESCE(expr, goqu.L(fallback))
}

func (PostgresDialect) JSONPathExists(col exp.Expression, path string, vars map[string]any) exp.Expression {
	if len(vars) == 0 {
		// No variables - simple form
		return goqu.L("jsonb_path_exists(?, ?::jsonpath)", col, path)
	}

	// Marshal variables to JSON
	varsJSON, err := json.Marshal(vars)
	if err != nil {
		// Fallback to empty vars if marshal fails
		return goqu.L("jsonb_path_exists(?, ?::jsonpath)", col, path)
	}

	return goqu.L("jsonb_path_exists(?, ?::jsonpath, ?::jsonb)", col, path, string(varsJSON))
}

func (PostgresDialect) JSONContains(col exp.Expression, value string) exp.Expression {
	return goqu.L("? @> ?::jsonb", col, value)
}

func (PostgresDialect) JSONExtract(col exp.Expression, path string) exp.Expression {
	return goqu.L("?->?", col, path)
}

// dialectRegistry maps dialect names to their implementations
var dialectRegistry = map[string]Dialect{
	"postgres": PostgresDialect{},
}

// GetSQLDialect returns the Dialect for a given dialect name.
// Returns PostgresDialect as the default if the dialect is not found.
func GetSQLDialect(name string) Dialect {
	if d, ok := dialectRegistry[name]; ok {
		return d
	}
	return PostgresDialect{} // Default to PostgreSQL
}

// RegisterDialect allows registering custom SQL dialects.
func RegisterDialect(name string, dialect Dialect) {
	dialectRegistry[name] = dialect
}
