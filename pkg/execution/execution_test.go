package execution_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/roneli/fastgql/pkg/log/adapters"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/roneli/fastgql/pkg/execution/builders"
	"github.com/roneli/fastgql/pkg/execution/test/graph"
	"github.com/roneli/fastgql/pkg/execution/test/graph/generated"
	"github.com/rs/zerolog/log"
)

// Test Postgres Graph Sanity checks, this assumes that the postgresql exists
// NOTE: run init.sql on the postgres so data will be seeded
func TestPostgresGraph(t *testing.T) {
	ctx := context.Background()
	connStr := os.Getenv("TEST_DATABASE_URL")

	if connStr == "" {
		// No connection string provided, spin up a test container
		_, b, _, _ := runtime.Caller(0)
		basepath := filepath.Dir(b)
		rootDir := filepath.Join(basepath, "..", "..")
		initScriptPath := filepath.Join(rootDir, ".github", "workflows", "data", "init.sql")

		// Check for Ryuk disabled env var or try to auto-detect if we should disable it
		// This is often needed in environments with SELinux or restricted Docker socket access
		if os.Getenv("TESTCONTAINERS_RYUK_DISABLED") == "" {
			// Optional: you could attempt to detect the environment here
			// For now, we rely on the user/environment to set the flag if needed.
			// However, explicitly disabling it in code for local dev iteration where it's known to fail
			// is also an option, but environment variable is cleaner.
		}

		pgContainer, err := postgres.Run(ctx,
			"postgres:latest",
			postgres.WithInitScripts(initScriptPath),
			postgres.WithDatabase("postgres"),
			postgres.WithUsername("postgres"),
			postgres.WithPassword("password"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(5*time.Second)),
		)
		require.NoError(t, err)
		defer func() {
			if err := pgContainer.Terminate(ctx); err != nil {
				t.Logf("failed to terminate container: %s", err)
			}
		}()

		connStr, err = pgContainer.ConnectionString(ctx, "sslmode=disable")
		require.NoError(t, err)
	}

	tt := []struct {
		name         string
		query        *graphql.RawParams
		want         string
		statusCode   int
		cleanupQuery *graphql.RawParams
		cleanupCode  int
		cleanupWant  string
	}{
		{
			name:       "BaseQuery",
			query:      &graphql.RawParams{Query: `query { users { name } }`},
			want:       "{\"data\":{\"users\":[{\"name\":\"Alice\"},{\"name\":\"Bob\"},{\"name\":\"Charlie\"},{\"name\":\"David\"},{\"name\":\"Eve\"}]}}",
			statusCode: 200,
		},

		{
			name:       "FetchPosts",
			query:      &graphql.RawParams{Query: `query { posts { name } }`},
			want:       "{\"data\":{\"posts\":[{\"name\":\"Hello World\"},{\"name\":\"GraphQL is awesome\"},{\"name\":\"Postgres is cool\"},{\"name\":\"Deno is interesting\"},{\"name\":\"Node.js is fast\"}]}}",
			statusCode: 200,
		},
		{
			name:       "PostsAggregate",
			query:      &graphql.RawParams{Query: `{ _postsAggregate { count sum { id } min { name } } }`},
			want:       `{"data":{"_postsAggregate":[{"count":5,"sum":{"id":15},"min":{"name":"Deno is interesting"}}]}}`,
			statusCode: 200,
		},

		{
			name:       "FetchPostsWithRelationAggregate",
			query:      &graphql.RawParams{Query: `query { posts(orderBy: {name: DESC}) { categories { name } _categoriesAggregate(filter: {name: {like: "%w%"}}) { count } }}`},
			want:       `{"data":{"posts":[{"categories":[{"name":"Science"},{"name":"Sports"}],"_categoriesAggregate":[{"count":0}]},{"categories":[{"name":"Entertainment"},{"name":"News"}],"_categoriesAggregate":[{"count":1}]},{"categories":[{"name":"News"},{"name":"Technology"}],"_categoriesAggregate":[{"count":1}]},{"categories":[{"name":"Technology"},{"name":"Science"}],"_categoriesAggregate":[{"count":0}]},{"categories":[{"name":"Sports"},{"name":"Entertainment"}],"_categoriesAggregate":[{"count":0}]}]}}`,
			statusCode: 200,
		},
		{
			name:       "FetchPostsWithAggregateSumAvg",
			query:      &graphql.RawParams{Query: `query { posts(orderBy: {name: DESC}) { categories { name } _categoriesAggregate { count sum { id } avg { id } } }}`},
			want:       `{"data":{"posts":[{"categories":[{"name":"Science"},{"name":"Sports"}],"_categoriesAggregate":[{"count":2,"sum":{"id":7},"avg":{"id":3.5}}]},{"categories":[{"name":"Entertainment"},{"name":"News"}],"_categoriesAggregate":[{"count":2,"sum":{"id":6},"avg":{"id":3}}]},{"categories":[{"name":"News"},{"name":"Technology"}],"_categoriesAggregate":[{"count":2,"sum":{"id":3},"avg":{"id":1.5}}]},{"categories":[{"name":"Technology"},{"name":"Science"}],"_categoriesAggregate":[{"count":2,"sum":{"id":5},"avg":{"id":2.5}}]},{"categories":[{"name":"Sports"},{"name":"Entertainment"}],"_categoriesAggregate":[{"count":2,"sum":{"id":9},"avg":{"id":4.5}}]}]}}`,
			statusCode: 200,
		},
		{
			name:         "InsertPost",
			query:        &graphql.RawParams{Query: `mutation { createPosts(inputs: [{id: 66, name: "ron", user_id: 1}]) { rows_affected posts { id name user { id name } }}}`},
			want:         "{\"data\":{\"createPosts\":{\"rows_affected\":1,\"posts\":[{\"id\":66,\"name\":\"ron\",\"user\":{\"id\":1,\"name\":\"Alice\"}}]}}}",
			statusCode:   200,
			cleanupQuery: &graphql.RawParams{Query: `mutation { deletePosts(filter: {id: {eq: 66}}) { rows_affected }}`},
			cleanupCode:  200,
			cleanupWant:  "{\"data\":{\"deletePosts\":{\"rows_affected\":1}}}",
		},
		{
			name:         "UpdatePost",
			query:        &graphql.RawParams{Query: `mutation { updatePosts(filter: {id: {eq: 1}}, input: {name: "ron"}) { rows_affected posts { id name user { id name } }}}`},
			want:         "{\"data\":{\"updatePosts\":{\"rows_affected\":1,\"posts\":[{\"id\":1,\"name\":\"ron\",\"user\":{\"id\":1,\"name\":\"Alice\"}}]}}}",
			statusCode:   200,
			cleanupQuery: &graphql.RawParams{Query: `mutation { updatePosts(filter: {id: {eq: 1}}, input: {name: "Hello World"}) { rows_affected posts { id name user { id name } }}}`},
			cleanupCode:  200,
			cleanupWant:  "{\"data\":{\"updatePosts\":{\"rows_affected\":1,\"posts\":[{\"id\":1,\"name\":\"Hello World\",\"user\":{\"id\":1,\"name\":\"Alice\"}}]}}}",
		},
	}

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		fmt.Printf("failed to create pool: %s", err)
		return
	}
	defer pool.Close()
	resolver := &graph.Resolver{}
	executableSchema := generated.NewExecutableSchema(generated.Config{Resolvers: resolver})
	// Set configuration
	cfg := &builders.Config{Schema: executableSchema.Schema(), Logger: adapters.NewZerologAdapter(log.Logger)}
	resolver.Cfg = cfg
	resolver.Executor = pool
	graphServer := handler.NewDefaultServer(executableSchema)
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			testQuery(t, graphServer, tc.query, tc.want, tc.statusCode)
			if tc.cleanupQuery != nil {
				testQuery(t, graphServer, tc.cleanupQuery, tc.cleanupWant, tc.cleanupCode)
			}
		})
	}
}

func testQuery(t *testing.T, server *handler.Server, params *graphql.RawParams, want string, wantCode int) {
	data, err := json.Marshal(params)
	require.Nil(t, err)
	request := httptest.NewRequest("POST", "/", bytes.NewBuffer(data))
	request.Header.Add("Content-Type", "application/json")
	responseRecorder := httptest.NewRecorder()
	server.ServeHTTP(responseRecorder, request)
	assert.Equal(t, want, responseRecorder.Body.String())
	assert.Equal(t, wantCode, responseRecorder.Code)
}
