//go:generate go run github.com/roneli/fastgql generate -c gqlgen.yml -f
package main

import (
	"context"
	"net/http"
	"os"

	"github.com/roneli/fastgql/pkg/log/adapters"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/roneli/fastgql/examples/interface/graph"
	"github.com/roneli/fastgql/examples/interface/graph/generated"
	"github.com/roneli/fastgql/pkg/execution/builders"
	"github.com/rs/zerolog/log"
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

	pool, err := pgxpool.New(context.Background(), pgConnectionString)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	resolver := &graph.Resolver{Executor: pool}
	executableSchema := generated.NewExecutableSchema(generated.Config{Resolvers: resolver})
	// Add logger to config for building trace logging
	cfg := &builders.Config{Schema: executableSchema.Schema(), Logger: adapters.NewZerologAdapter(log.Logger)}
	resolver.Cfg = cfg
	resolver.Executor = pool

	srv := handler.NewDefaultServer(executableSchema)
	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)
	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal().Err(err)
	}
}
