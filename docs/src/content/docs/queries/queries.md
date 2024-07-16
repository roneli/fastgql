---
title: Queries
description: Executing queries on a generated FastGQL Server
---

## Introduction

FastGQL allows to easily execute queries against a remote data source, i.e. postgres and convert the GraphQL AST into a valid SQL query and return queried data.

FastGQL auto-generates query filters, aggregation, pagination and ordering from your schema definition using the [#generate](../schema/directives#generate "mention") and [#generatefilterinput](../schema/directives#generatefilterinput "mention") directives.

## Queries

### Fetch list of objects

```graphql
query {
  users {
    name
  }
}
```

Fetch all available users.

### Fetch nested objects

```graphql
query {
  users {
    name
    posts {
      name
    }
  }
}
```

**fetch all users and their posts.**

### Fetch nested object recursively

```graphql
query {
  users {
    name
    posts {
      name
      categories {
        name
      }
      user {
        name
      }
    }
  }
}
```

fetch all users and their posts, for each post we fetch it's categories and the user who posted it.
