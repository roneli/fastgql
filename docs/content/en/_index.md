
---
title: "Documentation"
linkTitle: "Documentation"
weight: 20
type: list
menu:
  main:
    weight: 20
---

## What is it?

*fastGQL* is a **schema-first** extension library built on top [gqlgen](https://gqlgen.com), adding automatic schema augmentation & resolving similar to projects
such as [Hasura](https://github.com/hasura/graphql-engine/).


## What is supported?

*fastGql* supports slicing and dicing of the requested data with the following operations:

- [Filtering]({{< ref "/queries/filter" >}})
- [Pagination]({{< ref "/queries/pagination" >}})
- [Sorting]({{< ref "/queries/sorting" >}} )

future releases will add mutation(i.e insert, delete, update), aggregations, relational ordering and more!


## Why do I want it?

**Make powerful queries** Automatically adds filtering, pagination, and ordering to your schema.

* **What is it good for?**:
  - **fastgql is based on a Schema first approach** — unlike hasura this gives you more granulated control on your resolvers and code.
  - **fastgql prioritizes extendability** — You can modify resolvers, and your own custom operators and even your own database query builder.
  - **fastgql enables codegen** — We generate even more of the boring CRUD bits, so you can focus on building your app even faster!

* **What is it *not yet* good for?**: more CRUD support, support for multiple database (mongo, neo4j, cassandra etc'), custom middleware (auth, multi-tenancy, logging etc').

## Where should I go next?

Check out the *fastGQL* "getting started" to easily create a server and gqlgen's documentation:

* [Getting Started](/getting-started/): Get started with *fastGQL*
* [gqlgen](https://gqlgen.com): Check out gqlgen Go library for building GraphQL servers without any fuss.
