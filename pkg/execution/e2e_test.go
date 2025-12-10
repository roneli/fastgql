package execution_test

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/roneli/fastgql/pkg/execution"
	"github.com/roneli/fastgql/pkg/execution/__test__/graph"
	"github.com/roneli/fastgql/pkg/execution/__test__/graph/generated"
	"github.com/roneli/fastgql/pkg/execution/builders"
	"github.com/roneli/fastgql/pkg/execution/builders/sql"
	"github.com/roneli/fastgql/pkg/execution/testhelpers"
)

// e2eTestCase defines a single e2e test case
type e2eTestCase struct {
	Name     string
	Query    string
	Validate func(t *testing.T, data json.RawMessage)
}

// setupPostgresSuite creates a test client connected to a real PostgreSQL database.
func setupPostgresSuite(t *testing.T) (*testhelpers.TestClient, func()) {
	ctx := context.Background()
	pool, cleanup, err := testhelpers.GetTestPostgresPool(ctx)
	require.NoError(t, err)

	resolver := &graph.Resolver{}
	executableSchema := generated.NewExecutableSchema(generated.Config{Resolvers: resolver})

	cfg := &builders.Config{Schema: executableSchema.Schema()}
	multiExec := execution.NewMultiExecutor(executableSchema.Schema(), "postgres")
	multiExec.Register("postgres", sql.NewExecutor(pool, cfg))
	resolver.Executor = multiExec

	srv := handler.NewDefaultServer(executableSchema)
	client := testhelpers.NewTestClient(t, srv)

	return client, cleanup
}

func TestE2E(t *testing.T) {
	client, cleanup := setupPostgresSuite(t)
	defer cleanup()

	tests := []e2eTestCase{
		// Basic Query Tests
		{
			Name:  "basic/fetch_all_users",
			Query: `query { users { id name } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Users []struct {
						ID   int    `json:"id"`
						Name string `json:"name"`
					} `json:"users"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				assert.Len(t, result.Users, 5)
				assert.Equal(t, "Alice", result.Users[0].Name)
			},
		},
		{
			Name:  "basic/fetch_with_limit",
			Query: `query { users(limit: 2) { id } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Users []struct{ ID int } `json:"users"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				assert.Len(t, result.Users, 2)
			},
		},
		{
			Name:  "basic/fetch_with_filter",
			Query: `query { users(filter: { name: { eq: "Alice" } }) { name } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Users []struct{ Name string } `json:"users"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				require.Len(t, result.Users, 1)
				assert.Equal(t, "Alice", result.Users[0].Name)
			},
		},
		{
			Name:  "basic/fetch_posts",
			Query: `query { posts { id name } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Posts []struct {
						ID   int    `json:"id"`
						Name string `json:"name"`
					} `json:"posts"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				assert.Len(t, result.Posts, 5)
			},
		},

		// Relation Tests
		{
			Name:  "relations/one_to_one_post_user",
			Query: `query { posts(limit: 1) { name user { name } } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Posts []struct {
						Name string                `json:"name"`
						User struct{ Name string } `json:"user"`
					} `json:"posts"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				require.Len(t, result.Posts, 1)
				assert.NotEmpty(t, result.Posts[0].User.Name)
			},
		},
		{
			Name:  "relations/one_to_many_user_posts",
			Query: `query { users(filter: { name: { eq: "Alice" } }) { name posts { name } } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Users []struct {
						Name  string                  `json:"name"`
						Posts []struct{ Name string } `json:"posts"`
					} `json:"users"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				require.Len(t, result.Users, 1)
				assert.Equal(t, "Alice", result.Users[0].Name)
				assert.GreaterOrEqual(t, len(result.Users[0].Posts), 1)
			},
		},
		{
			Name:  "relations/many_to_many_posts_categories",
			Query: `query { posts(limit: 1) { name categories { name } } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Posts []struct {
						Name       string                  `json:"name"`
						Categories []struct{ Name string } `json:"categories"`
					} `json:"posts"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				require.Len(t, result.Posts, 1)
				assert.GreaterOrEqual(t, len(result.Posts[0].Categories), 1)
			},
		},

		// Interface Type Tests
		{
			Name:  "interface/type_discrimination",
			Query: `query { animals { id name type ... on Cat { color } ... on Dog { breed } } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Animals []map[string]any `json:"animals"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				assert.GreaterOrEqual(t, len(result.Animals), 2)

				hasCat, hasDog := false, false
				for _, animal := range result.Animals {
					animalType, ok := animal["type"].(string)
					if !ok {
						continue
					}
					if animalType == "cat" {
						hasCat = true
						assert.Contains(t, animal, "color")
					}
					if animalType == "dog" {
						hasDog = true
						assert.Contains(t, animal, "breed")
					}
				}
				assert.True(t, hasCat, "should have at least one cat")
				assert.True(t, hasDog, "should have at least one dog")
			},
		},

		// Aggregation Tests
		{
			Name:  "aggregation/count_users",
			Query: `query { _usersAggregate { count } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Aggregate []struct{ Count int } `json:"_usersAggregate"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				require.Len(t, result.Aggregate, 1)
				assert.Equal(t, 5, result.Aggregate[0].Count)
			},
		},
		{
			Name:  "aggregation/count_posts",
			Query: `query { _postsAggregate { count } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Aggregate []struct{ Count int } `json:"_postsAggregate"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				require.Len(t, result.Aggregate, 1)
				assert.Equal(t, 5, result.Aggregate[0].Count)
			},
		},

		// Ordering Tests
		{
			Name:  "ordering/ascending",
			Query: `query { users(orderBy: [{name: ASC}]) { name } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Users []struct{ Name string } `json:"users"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				assert.Equal(t, "Alice", result.Users[0].Name)
			},
		},
		{
			Name:  "ordering/descending",
			Query: `query { users(orderBy: [{name: DESC}]) { name } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Users []struct{ Name string } `json:"users"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				assert.Equal(t, "Eve", result.Users[0].Name)
			},
		},

		// Mutation Tests
		{
			Name:  "mutation/create_post",
			Query: `mutation { createPosts(inputs: [{ id: 100, name: "E2E Test Post", user_id: 1 }]) { rows_affected posts { id name } } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					CreatePosts struct {
						RowsAffected int `json:"rows_affected"`
						Posts        []struct {
							ID   int    `json:"id"`
							Name string `json:"name"`
						} `json:"posts"`
					} `json:"createPosts"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				assert.Equal(t, 1, result.CreatePosts.RowsAffected)
				require.Len(t, result.CreatePosts.Posts, 1)
				assert.Equal(t, "E2E Test Post", result.CreatePosts.Posts[0].Name)
			},
		},
		{
			Name:  "mutation/delete_post",
			Query: `mutation { deletePosts(filter: { name: { eq: "E2E Test Post" } }) { rows_affected } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					DeletePosts struct {
						RowsAffected int `json:"rows_affected"`
					} `json:"deletePosts"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				assert.Equal(t, 1, result.DeletePosts.RowsAffected)
			},
		},

		// JSON Filtering Tests - Typed JSON (@json directive)
		{
			Name:  "json/typed_filter_simple_field",
			Query: `query { products(filter: { attributes: { color: { eq: "red" } } }) { name } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Products []struct{ Name string } `json:"products"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				assert.Len(t, result.Products, 2) // Widget and Gizmo
				names := []string{result.Products[0].Name, result.Products[1].Name}
				assert.Contains(t, names, "Widget")
				assert.Contains(t, names, "Gizmo")
			},
		},
		{
			Name:  "json/typed_filter_nested_object",
			Query: `query { products(filter: { attributes: { details: { manufacturer: { eq: "Acme" } } } }) { name } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Products []struct{ Name string } `json:"products"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				assert.Len(t, result.Products, 2) // Widget and Gizmo
			},
		},
		{
			Name:  "json/typed_filter_multiple_fields",
			Query: `query { products(filter: { attributes: { color: { eq: "blue" }, size: { gt: 15 } } }) { name } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Products []struct{ Name string } `json:"products"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				assert.Len(t, result.Products, 2) // Gadget (size 20) and Device (size 25)
			},
		},
		{
			Name:  "json/typed_filter_with_AND",
			Query: `query { products(filter: { attributes: { AND: [{ color: { eq: "red" } }, { size: { gt: 12 } }] } }) { name } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Products []struct{ Name string } `json:"products"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				assert.Len(t, result.Products, 1) // Only Gizmo (red, size 15)
				assert.Equal(t, "Gizmo", result.Products[0].Name)
			},
		},
		{
			Name:  "json/typed_filter_with_OR",
			Query: `query { products(filter: { attributes: { OR: [{ color: { eq: "green" } }, { size: { lt: 10 } }] } }) { name } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Products []struct{ Name string } `json:"products"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				assert.Len(t, result.Products, 1) // Only Tool (green, size 5)
				assert.Equal(t, "Tool", result.Products[0].Name)
			},
		},

		// JSON Filtering Tests - Map scalar (dynamic JSON)
		{
			Name:  "json/map_contains_simple",
			Query: `query { products(filter: { metadata: { contains: { discount: "true" } } }) { name } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Products []struct{ Name string } `json:"products"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				assert.Len(t, result.Products, 2) // Widget and Gizmo have discount
			},
		},
		{
			Name:  "json/map_where_single_condition",
			Query: `query { products(filter: { metadata: { where: [{ path: "price", gt: 100 }] } }) { name } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Products []struct{ Name string } `json:"products"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				assert.Len(t, result.Products, 2) // Gadget (149.99) and Device (199.99)
			},
		},
		{
			Name:  "json/map_where_multiple_conditions",
			Query: `query { products(filter: { metadata: { where: [{ path: "price", lt: 100 }, { path: "discount", eq: "true" }] } }) { name } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Products []struct{ Name string } `json:"products"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				assert.Len(t, result.Products, 2) // Widget (99.99 with discount) and Gizmo (49.99 with discount)
			},
		},
		{
			Name:  "json/map_whereAny_or_conditions",
			Query: `query { products(filter: { metadata: { whereAny: [{ path: "rating", gt: 4 }, { path: "discount", eq: "true" }] } }) { name } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Products []struct{ Name string } `json:"products"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				assert.Len(t, result.Products, 3) // Widget, Gizmo (discount), Device (rating 4.5)
			},
		},
		{
			Name:  "json/map_combined_contains_and_where",
			Query: `query { products(filter: { metadata: { contains: {discount: "true"}, where: [{ path: "price", lt: 75 }] } }) { name } }`,
			Validate: func(t *testing.T, data json.RawMessage) {
				var result struct {
					Products []struct{ Name string } `json:"products"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				assert.Len(t, result.Products, 1) // Only Gizmo (discount + price 49.99)
				assert.Equal(t, "Gizmo", result.Products[0].Name)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			resp := client.Query(tc.Query, nil)
			require.Empty(t, resp.Errors, "GraphQL errors: %v", resp.Errors)
			tc.Validate(t, resp.Data)
		})
	}
}

// TestE2E_Pagination tests pagination with multiple queries (needs sequential execution).
func TestE2E_Pagination(t *testing.T) {
	client, cleanup := setupPostgresSuite(t)
	defer cleanup()

	// Fetch page 1
	resp1 := client.Query(`query { users(limit: 2, offset: 0, orderBy: [{id: ASC}]) { id } }`, nil)
	require.Empty(t, resp1.Errors)

	var page1 struct {
		Users []struct{ ID int } `json:"users"`
	}
	require.NoError(t, json.Unmarshal(resp1.Data, &page1))

	// Fetch page 2
	resp2 := client.Query(`query { users(limit: 2, offset: 2, orderBy: [{id: ASC}]) { id } }`, nil)
	require.Empty(t, resp2.Errors)

	var page2 struct {
		Users []struct{ ID int } `json:"users"`
	}
	require.NoError(t, json.Unmarshal(resp2.Data, &page2))

	assert.Len(t, page1.Users, 2)
	assert.Len(t, page2.Users, 2)
	assert.NotEqual(t, page1.Users[0].ID, page2.Users[0].ID)
}

// TestE2E_MutationSequence tests update mutation which requires create first.
func TestE2E_MutationSequence(t *testing.T) {
	client, cleanup := setupPostgresSuite(t)
	defer cleanup()

	// Create a post
	createResp := client.Query(`mutation { createPosts(inputs: [{ id: 101, name: "To Update", user_id: 1 }]) { posts { id } } }`, nil)
	require.Empty(t, createResp.Errors)

	var createResult struct {
		CreatePosts struct {
			Posts []struct{ ID int } `json:"posts"`
		} `json:"createPosts"`
	}
	require.NoError(t, json.Unmarshal(createResp.Data, &createResult))
	require.Len(t, createResult.CreatePosts.Posts, 1)
	postID := createResult.CreatePosts.Posts[0].ID

	// Update the post
	updateQuery := `mutation { updatePosts(input: { name: "Updated Post" }, filter: { id: { eq: ` + strconv.Itoa(postID) + ` } }) { rows_affected } }`
	updateResp := client.Query(updateQuery, nil)
	require.Empty(t, updateResp.Errors)

	var updateResult struct {
		UpdatePosts struct {
			RowsAffected int `json:"rows_affected"`
		} `json:"updatePosts"`
	}
	require.NoError(t, json.Unmarshal(updateResp.Data, &updateResult))
	assert.Equal(t, 1, updateResult.UpdatePosts.RowsAffected)

	// Verify update by querying
	verifyResp := client.Query(`query { posts(filter: { id: { eq: `+strconv.Itoa(postID)+` } }) { name } }`, nil)
	require.Empty(t, verifyResp.Errors)

	var verifyResult struct {
		Posts []struct{ Name string } `json:"posts"`
	}
	require.NoError(t, json.Unmarshal(verifyResp.Data, &verifyResult))
	require.Len(t, verifyResult.Posts, 1)
	assert.Equal(t, "Updated Post", verifyResult.Posts[0].Name)
}
