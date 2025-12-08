package testhelpers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/require"
)

// GraphQLRequest represents a GraphQL operation.
type GraphQLRequest struct {
	Query         string         `json:"query"`
	OperationName string         `json:"operationName,omitempty"`
	Variables     map[string]any `json:"variables,omitempty"`
}

// GraphQLResponse represents the response from a GraphQL operation.
type GraphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []GraphQLError  `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error.
type GraphQLError struct {
	Message string `json:"message"`
	Path    []any  `json:"path,omitempty"`
}

// TestClient wraps a test server for executing GraphQL operations.
type TestClient struct {
	server *httptest.Server
	t      *testing.T
}

// NewTestClient creates a test client from a gqlgen handler.
func NewTestClient(t *testing.T, h *handler.Server) *TestClient {
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	return &TestClient{server: server, t: t}
}

// Query executes a GraphQL query and returns the response.
func (c *TestClient) Query(query string, variables map[string]any) *GraphQLResponse {
	return c.execute(GraphQLRequest{
		Query:     query,
		Variables: variables,
	})
}

// MustQuery executes a query and fails the test if there are errors.
func (c *TestClient) MustQuery(query string, variables map[string]any, dest any) {
	resp := c.Query(query, variables)
	require.Empty(c.t, resp.Errors, "GraphQL errors: %v", resp.Errors)
	require.NoError(c.t, json.Unmarshal(resp.Data, dest))
}

func (c *TestClient) execute(req GraphQLRequest) *GraphQLResponse {
	body, err := json.Marshal(req)
	require.NoError(c.t, err)

	resp, err := http.Post(c.server.URL, "application/json", bytes.NewReader(body))
	require.NoError(c.t, err)
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	require.NoError(c.t, err)

	var gqlResp GraphQLResponse
	require.NoError(c.t, json.Unmarshal(data, &gqlResp))
	return &gqlResp
}

