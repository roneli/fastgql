package sql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSQLQuery(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		args    []any
		wantSQL string
	}{
		{
			name:    "simple_query",
			sql:     "SELECT * FROM users",
			args:    nil,
			wantSQL: "SELECT * FROM users",
		},
		{
			name:    "query_with_args",
			sql:     "SELECT * FROM users WHERE id = $1",
			args:    []any{1},
			wantSQL: "SELECT * FROM users WHERE id = $1",
		},
		{
			name:    "query_with_multiple_args",
			sql:     "SELECT * FROM users WHERE id = $1 AND name = $2",
			args:    []any{1, "Alice"},
			wantSQL: "SELECT * FROM users WHERE id = $1 AND name = $2",
		},
		{
			name:    "empty_query",
			sql:     "",
			args:    nil,
			wantSQL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewSQLQuery(tt.sql, tt.args)
			assert.Equal(t, tt.wantSQL, q.SQL())
			assert.Equal(t, tt.args, q.Args())
		})
	}
}

func TestSQLQuery_Native(t *testing.T) {
	q := NewSQLQuery("SELECT 1", []any{})
	native := q.Native()

	// Native should return the SQLQuery itself
	sqlQuery, ok := native.(SQLQuery)
	assert.True(t, ok)
	assert.Equal(t, "SELECT 1", sqlQuery.SQL())
}

func TestSQLQuery_String(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{
			name: "returns_sql",
			sql:  "SELECT * FROM users",
			want: "SELECT * FROM users",
		},
		{
			name: "empty_string",
			sql:  "",
			want: "",
		},
		{
			name: "complex_query",
			sql:  "SELECT u.id, u.name FROM users u JOIN posts p ON u.id = p.user_id WHERE u.active = true",
			want: "SELECT u.id, u.name FROM users u JOIN posts p ON u.id = p.user_id WHERE u.active = true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewSQLQuery(tt.sql, nil)
			assert.Equal(t, tt.want, q.String())
		})
	}
}

func TestSQLQuery_SQL(t *testing.T) {
	sql := "SELECT id, name FROM users WHERE active = $1"
	q := NewSQLQuery(sql, []any{true})

	assert.Equal(t, sql, q.SQL())
}

func TestSQLQuery_Args(t *testing.T) {
	tests := []struct {
		name string
		args []any
	}{
		{
			name: "nil_args",
			args: nil,
		},
		{
			name: "empty_args",
			args: []any{},
		},
		{
			name: "single_arg",
			args: []any{1},
		},
		{
			name: "multiple_args",
			args: []any{1, "test", true, 3.14},
		},
		{
			name: "complex_args",
			args: []any{[]int{1, 2, 3}, map[string]string{"key": "value"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewSQLQuery("SELECT 1", tt.args)
			assert.Equal(t, tt.args, q.Args())
		})
	}
}
