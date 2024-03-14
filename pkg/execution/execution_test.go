package execution_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/roneli/fastgql/pkg/log/adapters"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/roneli/fastgql/pkg/execution/builders"
	"github.com/roneli/fastgql/pkg/execution/test/graph"
	"github.com/roneli/fastgql/pkg/execution/test/graph/generated"
	"github.com/rs/zerolog/log"
)

const defaultPGConnection = "postgresql://localhost/postgres?user=postgres&password="

// Test Postgres Graph Sanity checks, this assumes that the postgresql exists
// NOTE: run init.sql on the postgres so data will be seeded
func TestPostgresGraph(t *testing.T) {
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
			name:       "FetchPostsWithAggregate",
			query:      &graphql.RawParams{Query: `query { posts { categories { name } _categoriesAggregate(filter: {name: {like: "%w%"}}) { count } }}`},
			want:       "{\"data\":{\"posts\":[{\"categories\":[{\"name\":\"News\"},{\"name\":\"Technology\"}],\"_categoriesAggregate\":[{\"count\":1}]},{\"categories\":[{\"name\":\"Technology\"},{\"name\":\"Science\"}],\"_categoriesAggregate\":[{\"count\":0}]},{\"categories\":[{\"name\":\"Science\"},{\"name\":\"Sports\"}],\"_categoriesAggregate\":[{\"count\":0}]},{\"categories\":[{\"name\":\"Sports\"},{\"name\":\"Entertainment\"}],\"_categoriesAggregate\":[{\"count\":0}]},{\"categories\":[{\"name\":\"Entertainment\"},{\"name\":\"News\"}],\"_categoriesAggregate\":[{\"count\":1}]}]}}",
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

	pool, err := pgxpool.New(context.Background(), defaultPGConnection)
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
