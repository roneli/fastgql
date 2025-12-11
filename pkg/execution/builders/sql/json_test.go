package sql

import (
	"testing"

	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		// Valid paths
		{name: "simple field", path: "price", wantErr: false},
		{name: "nested field", path: "nested.field", wantErr: false},
		{name: "array index", path: "items[0]", wantErr: false},
		{name: "array with nested", path: "items[0].name", wantErr: false},
		{name: "deep nesting", path: "a.b.c.d", wantErr: false},
		{name: "underscore field", path: "my_field", wantErr: false},
		{name: "mixed", path: "items[0].sub_items[1].value", wantErr: false},

		// Invalid paths
		{name: "empty", path: "", wantErr: true},
		{name: "starts with number", path: "0field", wantErr: true},
		{name: "special chars", path: "field;DROP TABLE", wantErr: true},
		{name: "sql injection attempt", path: "x' OR '1'='1", wantErr: true},
		{name: "jsonpath operators", path: "$.field", wantErr: true},
		{name: "quotes", path: "field\"name", wantErr: true},
		{name: "parentheses", path: "field()", wantErr: true},
		{name: "negative index", path: "items[-1]", wantErr: true},
		{name: "star wildcard", path: "items[*]", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildJsonPathExpression(t *testing.T) {
	tests := []struct {
		name           string
		conditions     []ConditionExpr
		logic          string
		wantPath       string
		wantVarsKeys   []string
		wantVarsValues []any
		wantErr        bool
	}{
		{
			name: "single eq condition",
			conditions: []ConditionExpr{
				{Path: "color", Op: "eq", Value: "red"},
			},
			logic:          "AND",
			wantPath:       "$ ? (@.color == $v0)",
			wantVarsKeys:   []string{"v0"},
			wantVarsValues: []any{"red"},
		},
		{
			name: "multiple AND conditions",
			conditions: []ConditionExpr{
				{Path: "price", Op: "gt", Value: 100},
				{Path: "active", Op: "eq", Value: true},
			},
			logic:          "AND",
			wantPath:       "$ ? (@.price > $v0 && @.active == $v1)",
			wantVarsKeys:   []string{"v0", "v1"},
			wantVarsValues: []any{100, true},
		},
		{
			name: "multiple OR conditions",
			conditions: []ConditionExpr{
				{Path: "status", Op: "eq", Value: "active"},
				{Path: "status", Op: "eq", Value: "pending"},
			},
			logic:          "OR",
			wantPath:       "$ ? (@.status == $v0 || @.status == $v1)",
			wantVarsKeys:   []string{"v0", "v1"},
			wantVarsValues: []any{"active", "pending"},
		},
		{
			name: "nested path",
			conditions: []ConditionExpr{
				{Path: "items[0].name", Op: "eq", Value: "widget"},
			},
			logic:          "AND",
			wantPath:       "$ ? (@.items[0].name == $v0)",
			wantVarsKeys:   []string{"v0"},
			wantVarsValues: []any{"widget"},
		},
		{
			name: "isNull true",
			conditions: []ConditionExpr{
				{Path: "deleted", Op: "isNull", Value: true},
			},
			logic:          "AND",
			wantPath:       "$ ? (@.deleted == null)",
			wantVarsKeys:   []string{},
			wantVarsValues: []any{},
		},
		{
			name: "isNull false",
			conditions: []ConditionExpr{
				{Path: "email", Op: "isNull", Value: false},
			},
			logic:          "AND",
			wantPath:       "$ ? (@.email != null)",
			wantVarsKeys:   []string{},
			wantVarsValues: []any{},
		},
		{
			name: "all operators",
			conditions: []ConditionExpr{
				{Path: "a", Op: "eq", Value: 1},
				{Path: "b", Op: "neq", Value: 2},
				{Path: "c", Op: "gt", Value: 3},
				{Path: "d", Op: "gte", Value: 4},
				{Path: "e", Op: "lt", Value: 5},
				{Path: "f", Op: "lte", Value: 6},
			},
			logic:          "AND",
			wantPath:       "$ ? (@.a == $v0 && @.b != $v1 && @.c > $v2 && @.d >= $v3 && @.e < $v4 && @.f <= $v5)",
			wantVarsKeys:   []string{"v0", "v1", "v2", "v3", "v4", "v5"},
			wantVarsValues: []any{1, 2, 3, 4, 5, 6},
		},
		{
			name: "like operator",
			conditions: []ConditionExpr{
				{Path: "name", Op: "like", Value: "^test.*"},
			},
			logic:          "AND",
			wantPath:       "$ ? (@.name like_regex $v0)",
			wantVarsKeys:   []string{"v0"},
			wantVarsValues: []any{"^test.*"},
		},
		{
			name:       "empty conditions",
			conditions: []ConditionExpr{},
			logic:      "AND",
			wantErr:    true,
		},
		{
			name: "invalid path",
			conditions: []ConditionExpr{
				{Path: "invalid;path", Op: "eq", Value: "x"},
			},
			logic:   "AND",
			wantErr: true,
		},
		{
			name: "unsupported operator",
			conditions: []ConditionExpr{
				{Path: "field", Op: "unsupported", Value: "x"},
			},
			logic:   "AND",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotVars, err := BuildJsonPathExpression(tt.conditions, tt.logic)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantPath, gotPath)

			// Check vars
			assert.Len(t, gotVars, len(tt.wantVarsKeys))
			for i, key := range tt.wantVarsKeys {
				assert.Equal(t, tt.wantVarsValues[i], gotVars[key])
			}
		})
	}
}

func TestBuildJsonFilterFromOperatorMap(t *testing.T) {
	tests := []struct {
		name         string
		filterMap    map[string]any
		wantPath     string
		wantVarsLen  int
		wantErr      bool
		wantContains string // substring to check in path
	}{
		{
			name: "single field single operator",
			filterMap: map[string]any{
				"color": map[string]any{"eq": "red"},
			},
			wantPath:    "$ ? (@.color == $v0)",
			wantVarsLen: 1,
		},
		{
			name: "single field multiple operators",
			filterMap: map[string]any{
				"price": map[string]any{"gt": 10, "lt": 100},
			},
			wantVarsLen:  2,
			wantContains: "@.price",
		},
		{
			name: "multiple fields",
			filterMap: map[string]any{
				"color": map[string]any{"eq": "red"},
				"size":  map[string]any{"gt": 10},
			},
			wantVarsLen:  2,
			wantContains: "@.color",
		},
		{
			name: "isNull operator",
			filterMap: map[string]any{
				"deleted": map[string]any{"isNull": true},
			},
			wantPath:    "$ ? (@.deleted == null)",
			wantVarsLen: 0,
		},
		{
			name: "AND logical operator",
			filterMap: map[string]any{
				"AND": []any{
					map[string]any{"price": map[string]any{"gt": 50}},
					map[string]any{"active": map[string]any{"eq": true}},
				},
			},
			wantVarsLen:  2,
			wantContains: "@.price",
		},
		{
			name: "OR logical operator",
			filterMap: map[string]any{
				"OR": []any{
					map[string]any{"status": map[string]any{"eq": "active"}},
					map[string]any{"status": map[string]any{"eq": "pending"}},
				},
			},
			wantVarsLen:  2,
			wantContains: "||",
		},
		{
			name: "NOT logical operator",
			filterMap: map[string]any{
				"NOT": map[string]any{
					"deleted": map[string]any{"eq": true},
				},
			},
			wantVarsLen:  1,
			wantContains: "!(",
		},
		{
			name: "nested field path",
			filterMap: map[string]any{
				"address.city": map[string]any{"eq": "NYC"},
			},
			wantPath:    "$ ? (@.address.city == $v0)",
			wantVarsLen: 1,
		},
		{
			name:      "empty filter",
			filterMap: map[string]any{},
			wantErr:   true,
		},
		{
			name: "invalid field path",
			filterMap: map[string]any{
				"invalid;path": map[string]any{"eq": "x"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotVars, err := BuildJsonFilterFromOperatorMap(tt.filterMap)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.wantPath != "" {
				assert.Equal(t, tt.wantPath, gotPath)
			}

			if tt.wantContains != "" {
				assert.Contains(t, gotPath, tt.wantContains)
			}

			assert.Len(t, gotVars, tt.wantVarsLen)
		})
	}
}

func TestBuildContainsExpression(t *testing.T) {
	tests := []struct {
		name    string
		value   map[string]any
		wantSQL string
		wantErr bool
	}{
		{
			name:    "simple key-value",
			value:   map[string]any{"color": "red"},
			wantSQL: `"col" @> '{"color":"red"}'::jsonb`,
		},
		{
			name:    "nested object",
			value:   map[string]any{"address": map[string]any{"city": "NYC"}},
			wantSQL: `"col" @> '{"address":{"city":"NYC"}}'::jsonb`,
		},
		{
			name:    "multiple keys",
			value:   map[string]any{"a": 1, "b": 2},
			wantSQL: `@>`, // Just check it contains @>
		},
		{
			name:    "boolean value",
			value:   map[string]any{"active": true},
			wantSQL: `"col" @> '{"active":true}'::jsonb`,
		},
		{
			name:    "empty map",
			value:   map[string]any{},
			wantErr: true,
		},
		{
			name:    "nil map",
			value:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := goqu.C("col")
			expr, err := BuildContainsExpression(col, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, expr)

			// Convert to SQL to verify
			sql, _, err := goqu.Dialect("postgres").Select().Where(expr).ToSQL()
			require.NoError(t, err)
			assert.Contains(t, sql, tt.wantSQL)
		})
	}
}

func TestBuildJsonPathExistsExpression(t *testing.T) {
	tests := []struct {
		name     string
		jsonPath string
		vars     map[string]any
		wantSQL  string
		wantErr  bool
	}{
		{
			name:     "simple path no vars",
			jsonPath: "$ ? (@.color == \"red\")",
			vars:     nil,
			wantSQL:  "jsonb_path_exists",
		},
		{
			name:     "path with vars",
			jsonPath: "$ ? (@.price > $v0)",
			vars:     map[string]any{"v0": 100},
			wantSQL:  `jsonb_path_exists("col", '$ ? (@.price > $v0)'::jsonpath, '{"v0":100}'::jsonb)`,
		},
		{
			name:     "empty path",
			jsonPath: "",
			vars:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := goqu.C("col")
			expr, err := BuildJsonPathExistsExpression(col, tt.jsonPath, tt.vars)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, expr)

			sql, _, err := goqu.Dialect("postgres").Select().Where(expr).ToSQL()
			require.NoError(t, err)
			assert.Contains(t, sql, tt.wantSQL)
		})
	}
}

func TestBuildMapFilter(t *testing.T) {
	tests := []struct {
		name    string
		filter  JsonFilter
		wantSQL []string // substrings that should be in SQL
		wantErr bool
	}{
		{
			name: "isNull true",
			filter: JsonFilter{
				IsNull: boolPtr(true),
			},
			wantSQL: []string{"IS NULL"},
		},
		{
			name: "isNull false",
			filter: JsonFilter{
				IsNull: boolPtr(false),
			},
			wantSQL: []string{"IS NOT NULL"},
		},
		{
			name: "contains only",
			filter: JsonFilter{
				Contains: map[string]any{"type": "premium"},
			},
			wantSQL: []string{"@>", `"type":"premium"`},
		},
		{
			name: "where conditions",
			filter: JsonFilter{
				Where: []JsonPathCondition{
					{Path: "price", Op: "gt", Value: 100},
				},
			},
			wantSQL: []string{"jsonb_path_exists", "@.price > $v0"},
		},
		{
			name: "whereAny conditions",
			filter: JsonFilter{
				WhereAny: []JsonPathCondition{
					{Path: "status", Op: "eq", Value: "a"},
					{Path: "status", Op: "eq", Value: "b"},
				},
			},
			wantSQL: []string{"jsonb_path_exists", "||"},
		},
		{
			name: "combined filters",
			filter: JsonFilter{
				Contains: map[string]any{"type": "special"},
				Where: []JsonPathCondition{
					{Path: "price", Op: "gt", Value: 50},
				},
				IsNull: boolPtr(false),
			},
			wantSQL: []string{"@>", "jsonb_path_exists", "IS NOT NULL"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := goqu.C("col")
			expr, err := BuildMapFilter(col, tt.filter)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, expr)

			sql, _, err := goqu.Dialect("postgres").Select().Where(expr).ToSQL()
			require.NoError(t, err)

			for _, want := range tt.wantSQL {
				assert.Contains(t, sql, want)
			}
		})
	}
}

func TestParseMapComparator(t *testing.T) {
	tests := []struct {
		name      string
		filterMap map[string]any
		wantErr   bool
		validate  func(t *testing.T, f JsonFilter)
	}{
		{
			name: "contains",
			filterMap: map[string]any{
				"contains": map[string]any{"key": "value"},
			},
			validate: func(t *testing.T, f JsonFilter) {
				assert.Equal(t, map[string]any{"key": "value"}, f.Contains)
			},
		},
		{
			name: "isNull",
			filterMap: map[string]any{
				"isNull": true,
			},
			validate: func(t *testing.T, f JsonFilter) {
				require.NotNil(t, f.IsNull)
				assert.True(t, *f.IsNull)
			},
		},
		{
			name: "where conditions",
			filterMap: map[string]any{
				"where": []any{
					map[string]any{"path": "price", "gt": 100},
					map[string]any{"path": "active", "eq": true},
				},
			},
			validate: func(t *testing.T, f JsonFilter) {
				assert.Len(t, f.Where, 2)
				assert.Equal(t, "price", f.Where[0].Path)
				assert.Equal(t, "gt", f.Where[0].Op)
				assert.Equal(t, 100, f.Where[0].Value)
			},
		},
		{
			name: "whereAny conditions",
			filterMap: map[string]any{
				"whereAny": []any{
					map[string]any{"path": "status", "eq": "active"},
				},
			},
			validate: func(t *testing.T, f JsonFilter) {
				assert.Len(t, f.WhereAny, 1)
			},
		},
		{
			name: "combined",
			filterMap: map[string]any{
				"contains": map[string]any{"type": "x"},
				"isNull":   false,
				"where": []any{
					map[string]any{"path": "a", "eq": 1},
				},
			},
			validate: func(t *testing.T, f JsonFilter) {
				assert.NotEmpty(t, f.Contains)
				require.NotNil(t, f.IsNull)
				assert.False(t, *f.IsNull)
				assert.Len(t, f.Where, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := ParseMapComparator(tt.filterMap)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, filter)
			}
		})
	}
}

func TestLogicalOperatorsCombined(t *testing.T) {
	// Test complex nested logical operators
	filterMap := map[string]any{
		"active": map[string]any{"eq": true},
		"OR": []any{
			map[string]any{"status": map[string]any{"eq": "published"}},
			map[string]any{
				"AND": []any{
					map[string]any{"draft": map[string]any{"eq": true}},
					map[string]any{"reviewed": map[string]any{"eq": true}},
				},
			},
		},
	}

	path, vars, err := BuildJsonFilterFromOperatorMap(filterMap)
	require.NoError(t, err)

	// Should contain the active condition
	assert.Contains(t, path, "@.active == $")

	// Should contain OR with ||
	assert.Contains(t, path, "||")

	// Should have all the variable values
	assert.GreaterOrEqual(t, len(vars), 3)
}

func TestBuildJsonFilterFromOperatorMap_NestedObjects(t *testing.T) {
	tests := []struct {
		name         string
		filterMap    map[string]any
		wantContains []string
		wantVarsLen  int
		wantErr      bool
	}{
		{
			name: "simple nested object",
			filterMap: map[string]any{
				"details": map[string]any{
					"manufacturer": map[string]any{"eq": "Acme"},
				},
			},
			wantContains: []string{"@.details.manufacturer == $v0"},
			wantVarsLen:  1,
		},
		{
			name: "deeply nested object",
			filterMap: map[string]any{
				"details": map[string]any{
					"specs": map[string]any{
						"dimensions": map[string]any{
							"width": map[string]any{"gt": 10},
						},
					},
				},
			},
			wantContains: []string{"@.details.specs.dimensions.width > $v0"},
			wantVarsLen:  1,
		},
		{
			name: "nested object with multiple fields",
			filterMap: map[string]any{
				"details": map[string]any{
					"manufacturer": map[string]any{"eq": "Acme"},
					"model":        map[string]any{"like": "^Pro"},
				},
			},
			wantContains: []string{"@.details.manufacturer", "@.details.model"},
			wantVarsLen:  2,
		},
		{
			name: "mixed flat and nested",
			filterMap: map[string]any{
				"name": map[string]any{"eq": "Widget"},
				"details": map[string]any{
					"price": map[string]any{"gt": 100},
				},
			},
			wantContains: []string{"@.name == $", "@.details.price > $"},
			wantVarsLen:  2,
		},
		{
			name: "nested with logical operators",
			filterMap: map[string]any{
				"details": map[string]any{
					"OR": []any{
						map[string]any{"color": map[string]any{"eq": "red"}},
						map[string]any{"color": map[string]any{"eq": "blue"}},
					},
				},
			},
			wantContains: []string{"@.details.color", "||"},
			wantVarsLen:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, vars, err := BuildJsonFilterFromOperatorMap(tt.filterMap)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			for _, want := range tt.wantContains {
				assert.Contains(t, path, want)
			}

			assert.Len(t, vars, tt.wantVarsLen)
		})
	}
}

func TestBuildJsonFilterFromOperatorMap_Arrays(t *testing.T) {
	tests := []struct {
		name         string
		filterMap    map[string]any
		wantContains []string
		wantVarsLen  int
		wantErr      bool
	}{
		{
			name: "array any with simple condition",
			filterMap: map[string]any{
				"items": map[string]any{
					"any": map[string]any{
						"name": map[string]any{"eq": "widget"},
					},
				},
			},
			wantContains: []string{"@.items[*].name == $v0"},
			wantVarsLen:  1,
		},
		{
			name: "array any with multiple conditions",
			filterMap: map[string]any{
				"items": map[string]any{
					"any": map[string]any{
						"name":  map[string]any{"eq": "widget"},
						"price": map[string]any{"lt": 100},
					},
				},
			},
			wantContains: []string{"@.items[*].name", "@.items[*].price"},
			wantVarsLen:  2,
		},
		{
			name: "array any with nested object",
			filterMap: map[string]any{
				"items": map[string]any{
					"any": map[string]any{
						"details": map[string]any{
							"category": map[string]any{"eq": "electronics"},
						},
					},
				},
			},
			wantContains: []string{"@.items[*].details.category == $v0"},
			wantVarsLen:  1,
		},
		{
			name: "combined array and regular field",
			filterMap: map[string]any{
				"name": map[string]any{"eq": "Order"},
				"items": map[string]any{
					"any": map[string]any{
						"qty": map[string]any{"gt": 0},
					},
				},
			},
			wantContains: []string{"@.name == $", "@.items[*].qty > $"},
			wantVarsLen:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, vars, err := BuildJsonFilterFromOperatorMap(tt.filterMap)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			for _, want := range tt.wantContains {
				assert.Contains(t, path, want)
			}

			assert.Len(t, vars, tt.wantVarsLen)
		})
	}
}

func TestIsOperatorMap(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]any
		want bool
	}{
		{
			name: "operator map with eq",
			m:    map[string]any{"eq": "value"},
			want: true,
		},
		{
			name: "operator map with multiple",
			m:    map[string]any{"gt": 10, "lt": 100},
			want: true,
		},
		{
			name: "operator map with any",
			m:    map[string]any{"any": map[string]any{}},
			want: true,
		},
		{
			name: "nested object (not operator)",
			m:    map[string]any{"manufacturer": map[string]any{"eq": "Acme"}},
			want: false,
		},
		{
			name: "mixed - still detected as operator",
			m:    map[string]any{"eq": "val", "nested": map[string]any{}},
			want: true, // Has at least one operator
		},
		{
			name: "empty map",
			m:    map[string]any{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isOperatorMap(tt.m)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}
