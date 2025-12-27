package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/roneli/fastgql/pkg/importer"
	"github.com/roneli/fastgql/pkg/importer/postgres"
	"github.com/spf13/cobra"
)

var (
	importConnStr         string
	importSchemaName      string
	importTables          []string
	importOutputFile      string
	importGenerateQueries bool
	importGenerateFilters bool
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "import GraphQL schema from a PostgreSQL database",
	Long:  `Import and generate a GraphQL schema from an existing PostgreSQL database by introspecting the database structure`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for DATABASE_URL environment variable if conn string not provided
		if importConnStr == "" {
			if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
				importConnStr = dbURL
			} else {
				return fmt.Errorf("connection string is required (use --conn or set DATABASE_URL)")
			}
		}

		ctx := context.Background()

		// Create PostgreSQL source
		source := postgres.NewSource(nil)
		if err := source.Connect(ctx, importConnStr); err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer source.Close()

		// Build options
		options := importer.IntrospectOptions{
			SchemaName:      importSchemaName,
			Tables:          importTables,
			GenerateQueries: importGenerateQueries,
			GenerateFilters: importGenerateFilters,
		}

		// Introspect database
		schema, err := source.Introspect(ctx, options)
		if err != nil {
			return fmt.Errorf("failed to introspect database: %w", err)
		}

		// Generate GraphQL schema
		astSource, err := importer.GenerateSchema(schema)
		if err != nil {
			return fmt.Errorf("failed to generate schema: %w", err)
		}

		// Determine output file
		outputFile := importOutputFile
		if outputFile == "" {
			outputFile = "graph/schema.graphql"
		}

		// Create directory if needed
		if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Write schema to file
		if err := os.WriteFile(outputFile, []byte(astSource.Input), 0644); err != nil {
			return fmt.Errorf("failed to write schema file: %w", err)
		}

		fmt.Printf("Successfully imported schema from PostgreSQL database\n")
		fmt.Printf("Schema written to: %s\n", outputFile)
		fmt.Printf("Found %d object types\n", len(schema.ObjectTypes))
		if len(schema.QueryFields) > 0 {
			fmt.Printf("Generated %d query fields\n", len(schema.QueryFields))
		}

		return nil
	},
}

func init() {
	importCmd.Flags().StringVarP(&importConnStr, "conn", "d", "", "PostgreSQL connection string (or set DATABASE_URL environment variable)")
	importCmd.Flags().StringVarP(&importSchemaName, "schema", "s", "public", "Database schema name to introspect")
	importCmd.Flags().StringSliceVarP(&importTables, "tables", "t", []string{}, "Comma-separated list of table names to import (empty = all tables)")
	importCmd.Flags().StringVarP(&importOutputFile, "output", "o", "", "Output file path for generated schema (default: graph/schema.graphql)")
	importCmd.Flags().BoolVarP(&importGenerateQueries, "queries", "q", false, "Generate query fields for all tables")
	importCmd.Flags().BoolVarP(&importGenerateFilters, "filters", "f", false, "Generate filter inputs for all types")
}
