//go:generate go run github.com/roneli/fastgql generate -c gqlgen.yml
package main

import (
	"context"

	"github.com/99designs/gqlgen/graphql/playground"

	"github.com/roneli/fastgql/log/adapters"
	"github.com/rs/zerolog/log"

	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/roneli/fastgql/builders"
	"github.com/roneli/fastgql/example/graph"
	"github.com/roneli/fastgql/example/graph/generated"
)

const defaultPort = "8081"

const defaultPGConnection = "postgresql://localhost/main?user=postgres&password=password"

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
	resolver := &graph.Resolver{Executor: pool}
	executableSchema := generated.NewExecutableSchema(generated.Config{Resolvers: resolver})
	// Set configuration

	// Create a new instance of the logger. You can have any number of instances.
	resolver.Cfg = &builders.Config{Schema: executableSchema.Schema(), Logger: adapters.NewZerologAdapter(log.Logger)}
	srv := handler.NewDefaultServer(executableSchema)

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal().Err(err)
	}
}
