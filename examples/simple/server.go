//go:generate go run github.com/roneli/fastgql generate -c gqlgen.yml -f
package main

const defaultPort = "8081"

const defaultPGConnection = "postgresql://localhost/movies?user=postgres&password=password"

func main() {
	//port := os.Getenv("PORT")
	//if port == "" {
	//	port = defaultPort
	//}
	//pgConnectionString := os.Getenv("PG_CONN_STR")
	//if pgConnectionString == "" {
	//	pgConnectionString = defaultPGConnection
	//}
	//
	//pool, err := pgxpool.New(context.Background(), pgConnectionString)
	//if err != nil {
	//	panic(err)
	//}
	////resolver := &graph.Resolver{}
	////executableSchema := generated.NewExecutableSchema(generated.Config{Resolvers: resolver})
	////// Set configuration
	////cfg := &builders.Config{Schema: executableSchema.Schema(), Logger: adapters.NewZerologAdapter(log.Logger)}
	//resolver.Cfg = cfg
	//resolver.Executor = execution.NewExecutor(map[string]execution.Driver{
	//	"mongo":    mongo.NewDriver("mongo", cfg, "mongodb://root:example@127.0.0.1:27017/"),
	//	"postgres": sql.NewDriver("postgres", cfg, pool),
	//})
	//srv := handler.NewDefaultServer(executableSchema)
	//
	//http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	//http.Handle("/query", srv)
	//
	//log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	//if err := http.ListenAndServe(":"+port, nil); err != nil {
	//	log.Fatal().Err(err)
	//}
}
