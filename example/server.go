package main

import (
	"context"
	"fastgql/builders"
	"fastgql/example/graph"
	"fastgql/example/graph/generated"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4"
	requestid "github.com/thanhhh/gin-requestid"
)

type MockRepo struct {}

func (m MockRepo) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {

	conn, err := pgx.Connect(ctx, "postgresql://localhost/postgres?user=postgres&password=changeme&search_path=main")
	if err != nil {
		return nil, err
	}
	return conn.Query(ctx, query, args...)
}

// Defining the Graphql handler
func graphqlHandler() gin.HandlerFunc {
	// NewExecutableSchema and Config are in the generated.go file
	// Resolver is in the resolver.go file
	resolver :=  &graph.Resolver{Sql: MockRepo{}}
	executableSchema := generated.NewExecutableSchema(generated.Config{Resolvers:resolver})
	// Set Schema
	resolver.Cfg = &builders.Config{
		Schema:   executableSchema.Schema(),
		Logger:   nil,
		LogLevel: 0,
	}
	h := handler.NewDefaultServer(executableSchema)

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// Defining the Playground handler
func playgroundHandler() gin.HandlerFunc {
	h := playground.Handler("GraphQL", "/query")

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func main() {
	// Setting up Gin
	r := gin.Default()
	r.Use(requestid.RequestID())
	r.POST("/query", graphqlHandler())
	r.GET("/", playgroundHandler())
	r.Run()
}
