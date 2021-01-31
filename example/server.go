package main

import (
	"context"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/roneli/fastgql/builders"
	"github.com/roneli/fastgql/example/graph"
	"github.com/roneli/fastgql/example/graph/generated"
	"log"
	"net/http"
	"os"
)

const defaultPort = "8080"

const defaultPGConnection = "postgresql://localhost/postgres?user=postgres"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	pgConnectionString := os.Getenv("PG_CONN_STR")
	if pgConnectionString == "" {
		pgConnectionString = defaultPGConnection
	}

	pool, err := pgxpool.Connect(context.Background(), pgConnectionString)

	if err != nil {
		panic(err)
	}
	resolver :=  &graph.Resolver{Sql: pool}
	executableSchema := generated.NewExecutableSchema(generated.Config{Resolvers:resolver})
	resolver.Cfg = &builders.Config{Schema: executableSchema.Schema()}

	srv := handler.NewDefaultServer(executableSchema)

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":" + port, nil))
}
