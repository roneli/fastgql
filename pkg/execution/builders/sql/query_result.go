package sql

// SQLQuery implements builders.Query for SQL databases.
// It wraps the SQL string and prepared statement arguments.
type SQLQuery struct {
	sql  string
	args []any
}

// NewSQLQuery creates a new SQLQuery with the given SQL and arguments.
func NewSQLQuery(sql string, args []any) SQLQuery {
	return SQLQuery{sql: sql, args: args}
}

// Native returns the SQLQuery itself as the native query object.
// This allows type assertion: query.Native().(SQLQuery)
func (q SQLQuery) Native() any {
	return q
}

// String returns the SQL query string for logging/debugging.
func (q SQLQuery) String() string {
	return q.sql
}

// SQL returns the SQL query string.
func (q SQLQuery) SQL() string {
	return q.sql
}

// Args returns the prepared statement arguments.
func (q SQLQuery) Args() []any {
	return q.args
}
