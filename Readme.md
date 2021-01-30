![Alt text](./docs/assets/icons/logo.svg)

## What is fastGQL?

[*fastGQL*](https://github.com/roneli/fastgql) is a Go library that extends [gqlgen](https://github.com/99designs/gqlgen) to create a blazing-fast GraphQL server that gives you instant, 
realtime GraphQL APIs over Postgres.

- **fastgql is based on a Schema first approach** — You get to Define your API using the GraphQL [Schema Definition Language](http://graphql.org/learn/schema/).
- **fastgql prioritizes extendability** — You can modify resolvers, and your own custom operators and even your own database query builder.
- **fastgql enables codegen** — We generate even more of the boring CRUD bits, so you can focus on building your app even faster!


## Getting Started
- To install fastgql run the command `go get github.com/roneli/fastgql` in your project directory.<br/> 
- You could initialize a new project using the recommended folder structure by running this command `go run github.com/roneli/fastgql init`.

You could find a more comprehensive guide on [gqlgen](https://github.com/99designs/gqlgen) to help you get started [here](https://gqlgen.com/getting-started/).
We also have a couple of  [examples](https://github.com/roneli/fastgql/tree/master/example) that show how fastgql generates the full API seamlessly.

## Reporting Issues

If you think you've found a bug, or something isn't behaving the way you think it should, please raise an [issue](https://github.com/roneli/fastgql/issues) on GitHub.

## Contributing
Feel free to open Pull-Request for small fixes and changes. For bigger changes and new providers please open an [issue](https://github.com/roneli/fastgql/issues) first to prevent double work and discuss relevant stuff.

## Roadmap

- aggregate queries
- configurable database connections
- tests
- Support multiple database (mongodb, cockroachDB, neo4j)
- full CRUD creation
