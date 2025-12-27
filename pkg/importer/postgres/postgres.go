package postgres

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jinzhu/inflection"
	"github.com/roneli/fastgql/pkg/importer"
	schemapkg "github.com/roneli/fastgql/pkg/schema"
)

//go:embed introspect.sql
var introspectionQuery string

// Source implements importer.Source for PostgreSQL databases
type Source struct {
	pool *pgxpool.Pool
}

// NewSource creates a new PostgreSQL source
func NewSource(pool *pgxpool.Pool) *Source {
	return &Source{pool: pool}
}

// Connect establishes a connection to the database
func (p *Source) Connect(ctx context.Context, connStr string) error {
	var err error
	p.pool, err = pgxpool.New(ctx, connStr)
	return err
}

// Close closes the database connection
func (p *Source) Close() error {
	if p.pool != nil {
		p.pool.Close()
	}
	return nil
}

// Introspect analyzes the database schema and returns a Schema structure
func (p *Source) Introspect(ctx context.Context, options importer.IntrospectOptions) (*importer.Schema, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("database connection not established")
	}

	schemaName := options.SchemaName
	if schemaName == "" {
		schemaName = "public"
	}

	// Build the introspection query with table filter
	query, args := buildQuery(schemaName, options.Tables)

	// Execute the query
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute introspection query: %w", err)
	}
	defer rows.Close()

	// Parse results
	return parseIntrospectionResults(rows, options)
}

// buildQuery creates the query string with optional table filter
func buildQuery(schemaName string, tables []string) (string, []interface{}) {
	query := introspectionQuery
	args := []interface{}{schemaName}

	// Add table filter if specified
	if len(tables) > 0 {
		placeholders := make([]string, len(tables))
		for i := range tables {
			placeholders[i] = fmt.Sprintf("$%d", len(args)+i+1)
		}
		tableFilter := fmt.Sprintf("AND t.table_name IN (%s)", strings.Join(placeholders, ", "))
		query = strings.Replace(query, "%s", tableFilter, 1)
		for _, t := range tables {
			args = append(args, t)
		}
	} else {
		query = strings.Replace(query, "%s", "", 1)
	}

	return query, args
}

// parseIntrospectionResults parses the query results into a Schema structure
func parseIntrospectionResults(rows pgx.Rows, options importer.IntrospectOptions) (*importer.Schema, error) {
	schema := &importer.Schema{
		ObjectTypes: []*importer.ObjectType{},
		QueryFields: []*importer.QueryField{},
	}

	// Map to track tables and their foreign keys for relation building
	tableMap := make(map[string]*importer.ObjectType)
	foreignKeyMap := make(map[string][]foreignKeyInfo)
	pkMap := make(map[string][]string)
	uniqueColumnMap := make(map[string]map[string]bool)

	for rows.Next() {
		var tableSchema, tableName string
		var columnsJSON, uniqueConstraintsJSON, foreignKeysJSON json.RawMessage
		var pkColumns []string
		var isJunction bool

		err := rows.Scan(&tableSchema, &tableName, &columnsJSON, &pkColumns, &uniqueConstraintsJSON, &foreignKeysJSON, &isJunction)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Parse columns
		var columns []columnInfo
		if err := json.Unmarshal(columnsJSON, &columns); err != nil {
			return nil, fmt.Errorf("failed to parse columns: %w", err)
		}

		// Create ObjectType
		objType := &importer.ObjectType{
			Name:      strcase.ToCamel(tableName),
			TableName: tableName,
			Schema:    tableSchema,
			Dialect:   "postgres",
			Fields:    []*importer.Field{},
			Relations: []*importer.Relation{},
		}

		if options.GenerateFilters {
			objType.GenerateFilterInput = true
		}

		// Parse unique constraints to determine which columns are unique
		var uniqueConstraints []uniqueConstraintInfo
		if len(uniqueConstraintsJSON) > 0 {
			if err := json.Unmarshal(uniqueConstraintsJSON, &uniqueConstraints); err != nil {
				return nil, fmt.Errorf("failed to parse unique constraints: %w", err)
			}
		}

		uniqueColumnSet := make(map[string]bool)
		for _, uc := range uniqueConstraints {
			for _, col := range uc.Columns {
				uniqueColumnSet[col] = true
			}
		}
		for _, pkCol := range pkColumns {
			uniqueColumnSet[pkCol] = true
		}

		// Convert columns to fields
		for _, col := range columns {
			field := &importer.Field{
				Name:           strcase.ToLowerCamel(col.Name),
				ColumnName:     col.Name,
				IsNonNull:      !col.IsNullable,
				IsList:         col.IsArray,
				IsJSON:         col.IsJSON,
				JSONColumnName: col.Name,
			}

			// Map PostgreSQL types to GraphQL types
			field.Type = mapPostgresTypeToGraphQL(col.Type, col.UDTName, col.IsArray)

			objType.Fields = append(objType.Fields, field)
		}

		// Parse foreign keys
		var foreignKeys []foreignKeyInfo
		if len(foreignKeysJSON) > 0 {
			if err := json.Unmarshal(foreignKeysJSON, &foreignKeys); err != nil {
				return nil, fmt.Errorf("failed to parse foreign keys: %w", err)
			}
		}

		// Store foreign keys for later relation building
		tableKey := fmt.Sprintf("%s.%s", tableSchema, tableName)
		foreignKeyMap[tableKey] = foreignKeys
		pkMap[tableKey] = pkColumns
		uniqueColumnMap[tableKey] = uniqueColumnSet

		tableMap[tableKey] = objType
		schema.ObjectTypes = append(schema.ObjectTypes, objType)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Build relations
	// First, identify junction tables
	junctionTableSet := make(map[string]bool)
	junctionTableInfo := make(map[string]junctionInfo) // tableKey -> junction info

	for tableKey, fks := range foreignKeyMap {
		// Check if this is a junction table by counting distinct referenced tables
		refTables := make(map[string]bool)
		for _, fk := range fks {
			if fk.ReferencedTable != "" {
				refTables[fk.ReferencedTable] = true
			}
		}
		if len(refTables) == 2 && len(fks) == 2 {
			junctionTableSet[tableKey] = true
			// Store junction table info
			junctionTableInfo[tableKey] = junctionInfo{
				TableName: tableMap[tableKey].TableName,
				FKs:       fks,
			}
		}
	}

	if err := buildRelations(tableMap, foreignKeyMap, junctionTableSet, pkMap, uniqueColumnMap, junctionTableInfo); err != nil {
		return nil, fmt.Errorf("failed to build relations: %w", err)
	}

	// Generate query fields if requested
	if options.GenerateQueries {
		for _, objType := range schema.ObjectTypes {
			pluralTableName := inflection.Plural(objType.TableName)
			queryField := &importer.QueryField{
				Name:     strcase.ToLowerCamel(pluralTableName),
				Type:     objType.Name,
				IsList:   true,
				Generate: true,
			}
			schema.QueryFields = append(schema.QueryFields, queryField)
		}
	}

	return schema, nil
}

// Helper types for parsing JSON
type columnInfo struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	UDTName         string `json:"udt_name"`
	IsNullable      bool   `json:"is_nullable"`
	IsJSON          bool   `json:"is_json"`
	IsArray         bool   `json:"is_array"`
	OrdinalPosition int    `json:"ordinal_position"`
}

type uniqueConstraintInfo struct {
	ConstraintName string   `json:"constraint_name"`
	Columns        []string `json:"columns"`
}

type foreignKeyInfo struct {
	ConstraintName        string   `json:"constraint_name"`
	Columns               []string `json:"columns"`
	ReferencedTableSchema string   `json:"referenced_table_schema"`
	ReferencedTable       string   `json:"referenced_table"`
	ReferencedColumns     []string `json:"referenced_columns"`
}

type junctionInfo struct {
	TableName string
	FKs       []foreignKeyInfo
}

// buildRelations creates Relation objects from foreign key information
func buildRelations(
	tableMap map[string]*importer.ObjectType,
	foreignKeyMap map[string][]foreignKeyInfo,
	junctionTableSet map[string]bool,
	pkMap map[string][]string,
	uniqueColumnMap map[string]map[string]bool,
	junctionTableInfo map[string]junctionInfo,
) error {
	// Build ONE_TO_ONE and ONE_TO_MANY relations (non-junction tables)
	// We need to create relations on the referenced table (the "one" side), not the table with the FK
	for tableKey, objType := range tableMap {
		// Skip junction tables - they're handled separately
		if junctionTableSet[tableKey] {
			continue
		}

		foreignKeys := foreignKeyMap[tableKey]

		for _, fk := range foreignKeys {
			refTableKey := fmt.Sprintf("%s.%s", fk.ReferencedTableSchema, fk.ReferencedTable)
			refTable, exists := tableMap[refTableKey]
			if !exists {
				continue
			}

			// Skip if referenced table is a junction table (handled in MANY_TO_MANY logic)
			if junctionTableSet[refTableKey] {
				continue
			}

			// Check if FK columns in the source table are unique (ONE_TO_ONE)
			// Otherwise it's ONE_TO_MANY (refTable -> objType)
			sourceUniqueCols := uniqueColumnMap[tableKey]
			allUnique := true
			for _, fkCol := range fk.Columns {
				if !sourceUniqueCols[fkCol] {
					allUnique = false
					break
				}
			}

			// Convert column names to GraphQL field names (camelCase)
			fields := make([]string, len(fk.Columns))
			for i, col := range fk.Columns {
				fields[i] = strcase.ToLowerCamel(col)
			}
			references := make([]string, len(fk.ReferencedColumns))
			for i, col := range fk.ReferencedColumns {
				references[i] = strcase.ToLowerCamel(col)
			}

			if allUnique {
				// ONE_TO_ONE: FK columns are unique, so each record in source relates to exactly one in target
				// Create relation on the source table pointing to the referenced table
				// Use singular form for the field name
				fieldName := inflection.Singular(strcase.ToLowerCamel(refTable.Name))
				relation := &importer.Relation{
					FieldName:  fieldName,
					Type:       schemapkg.OneToOne,
					Fields:     fields,
					References: references,
					TargetType: refTable.Name,
				}
				objType.Relations = append(objType.Relations, relation)
			} else {
				// ONE_TO_MANY: FK columns are not unique, so many records in source relate to one in target
				// Create relation on the source table for FK navigation (singular, pointing to referenced table)
				fieldName := inflection.Singular(strcase.ToLowerCamel(refTable.Name))
				fkRelation := &importer.Relation{
					FieldName:  fieldName,
					Type:       schemapkg.OneToOne, // The FK itself is a ONE_TO_ONE from the many side
					Fields:     fields,
					References: references,
					TargetType: refTable.Name,
				}
				objType.Relations = append(objType.Relations, fkRelation)

				// Also create ONE_TO_MANY relation on the referenced table (the "one" side) pointing to the source table (the "many" side)
				oneToManyFields := make([]string, len(fk.ReferencedColumns))
				for i, col := range fk.ReferencedColumns {
					oneToManyFields[i] = strcase.ToLowerCamel(col)
				}
				oneToManyReferences := make([]string, len(fk.Columns))
				for i, col := range fk.Columns {
					oneToManyReferences[i] = strcase.ToLowerCamel(col)
				}
				oneToManyRelation := &importer.Relation{
					FieldName:  strcase.ToLowerCamel(objType.Name), // Plural form for the array
					Type:       schemapkg.OneToMany,
					Fields:     oneToManyFields,     // The "one" side's columns (converted to camelCase)
					References: oneToManyReferences, // The "many" side's FK columns (converted to camelCase)
					TargetType: objType.Name,
				}
				refTable.Relations = append(refTable.Relations, oneToManyRelation)
			}
		}
	}

	// Build MANY_TO_MANY relations on parent tables
	for tableKey, objType := range tableMap {
		if junctionTableSet[tableKey] {
			continue // Skip junction tables themselves
		}

		// Check if any junction table references this table
		for _, jInfo := range junctionTableInfo {
			for _, jfk := range jInfo.FKs {
				if jfk.ReferencedTable == objType.TableName {
					// Find the other FK in the junction table
					var otherFK foreignKeyInfo
					for _, other := range jInfo.FKs {
						if other.ReferencedTable != objType.TableName {
							otherFK = other
							break
						}
					}

					// Get the other parent table
					otherTableKey := fmt.Sprintf("%s.%s", otherFK.ReferencedTableSchema, otherFK.ReferencedTable)
					otherTable, exists := tableMap[otherTableKey]
					if !exists {
						continue
					}

					// Convert junction table column names to GraphQL field names (camelCase)
					manyToManyFields := make([]string, len(jfk.Columns))
					for i, col := range jfk.Columns {
						manyToManyFields[i] = strcase.ToLowerCamel(col)
					}
					manyToManyReferences := make([]string, len(otherFK.Columns))
					for i, col := range otherFK.Columns {
						manyToManyReferences[i] = strcase.ToLowerCamel(col)
					}
					// Create MANY_TO_MANY relation on this table
					relation := &importer.Relation{
						FieldName:            strcase.ToLowerCamel(otherTable.Name),
						Type:                 schemapkg.ManyToMany,
						Fields:               []string{"id"}, // This table's PK
						References:           []string{"id"}, // Other table's PK
						TargetType:           otherTable.Name,
						ManyToManyTable:      jInfo.TableName,
						ManyToManyFields:     manyToManyFields,     // Junction table FK pointing to this table
						ManyToManyReferences: manyToManyReferences, // Junction table FK pointing to other table
					}
					objType.Relations = append(objType.Relations, relation)
				}
			}
		}
	}

	return nil
}

// mapPostgresTypeToGraphQL maps PostgreSQL types to GraphQL types
func mapPostgresTypeToGraphQL(dataType, udtName string, isArray bool) string {
	// Handle arrays - get the base type
	if isArray {
		baseType := strings.TrimPrefix(udtName, "_")
		return mapPostgresTypeToGraphQL(dataType, baseType, false)
	}

	// Handle JSON types
	if udtName == "json" || udtName == "jsonb" {
		return schemapkg.GraphQLTypeMap
	}

	// Map standard PostgreSQL types
	switch udtName {
	case "int2", "smallint":
		return schemapkg.GraphQLTypeInt
	case "int4", "integer":
		return schemapkg.GraphQLTypeInt
	case "int8", "bigint":
		return schemapkg.GraphQLTypeInt
	case "float4", "real":
		return schemapkg.GraphQLTypeFloat
	case "float8", "double precision":
		return schemapkg.GraphQLTypeFloat
	case "numeric", "decimal":
		return schemapkg.GraphQLTypeFloat
	case "bool", "boolean":
		return schemapkg.GraphQLTypeBoolean
	case "text", "varchar", "char", "character varying", "character":
		return schemapkg.GraphQLTypeString
	case "uuid":
		return schemapkg.GraphQLTypeID
	case "date", "time", "timestamp", "timetz", "timestamptz":
		return schemapkg.GraphQLTypeString
	default:
		// Default to String for unknown types
		return schemapkg.GraphQLTypeString
	}
}
