//go:generate go run github.com/roneli/fastgql generate -c gqlgen.yml
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/roneli/fastgql/examples/init/graph"
	"github.com/roneli/fastgql/examples/init/graph/generated"
	"github.com/roneli/fastgql/pkg/execution"
	"github.com/roneli/fastgql/pkg/execution/builders"
	"github.com/roneli/fastgql/pkg/execution/builders/sql"
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

	resolver := &graph.Resolver{}
	executableSchema := generated.NewExecutableSchema(generated.Config{Resolvers: resolver})

	// Create config and executor
	cfg := &builders.Config{Schema: executableSchema.Schema()}
	multiExec := execution.NewMultiExecutor(executableSchema.Schema(), "postgres")
	multiExec.Register("postgres", sql.NewExecutor(pool, cfg))
	resolver.Executor = multiExec

	srv := handler.NewDefaultServer(executableSchema)
	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)
	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Println(err)
	}
}
