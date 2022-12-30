//go:build integration_test

package execution_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/roneli/fastgql/pkg/log/adapters"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/roneli/fastgql/pkg/execution"
	"github.com/roneli/fastgql/pkg/execution/builders"
	"github.com/roneli/fastgql/pkg/execution/builders/sql"
	"github.com/roneli/fastgql/pkg/execution/test/graph"
	"github.com/roneli/fastgql/pkg/execution/test/graph/generated"
)

const defaultPGConnection = "postgresql://localhost/postgres?user=postgres&password=password"

// Test Postgres Graph Sanity checks, this assumes that the posgresql exists
// NOTE: run init.sql on the postgres so data will be seeded
func TestPostgresGraph(t *testing.T) {
	tt := []struct {
		name       string
		query      *graphql.RawParams
		want       string
		statusCode int
	}{
		{
			name:       "BaseQuery",
			query:      &graphql.RawParams{Query: `query { users { name } }`},
			want:       "{\"data\":{\"users\":[{\"name\":\"userA\"},{\"name\":\"userB\"}]}}",
			statusCode: 200,
		},

		{
			name:       "FetchPosts",
			query:      &graphql.RawParams{Query: `query { posts { name } }`},
			want:       "{\"data\":{\"posts\":[{\"name\":\"postA\"},{\"name\":\"postB\"},{\"name\":\"postC\"}]}}",
			statusCode: 200,
		},
	}

	pool, err := pgxpool.Connect(context.Background(), defaultPGConnection)
	if err != nil {
		panic(err)
	}
	resolver := &graph.Resolver{}
	executableSchema := generated.NewExecutableSchema(generated.Config{Resolvers: resolver})
	// Set configuration
	cfg := &builders.Config{Schema: executableSchema.Schema(), Logger: adapters.NewZerologAdapter(log.Logger)}
	resolver.Cfg = cfg
	resolver.Executor = execution.NewExecutor(map[string]execution.Driver{
		"postgres": sql.NewDriver("postgres", cfg, pool),
	})
	graphServer := handler.NewDefaultServer(executableSchema)
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.query)
			require.Nil(t, err)
			request := httptest.NewRequest("POST", "/", bytes.NewBuffer(data))
			request.Header.Add("Content-Type", "application/json")
			responseRecorder := httptest.NewRecorder()
			graphServer.ServeHTTP(responseRecorder, request)
			assert.Equal(t, tc.want, responseRecorder.Body.String())
		})
	}

}
