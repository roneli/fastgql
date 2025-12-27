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

## JSON Field Selection

FastGQL supports efficient nested field selection for typed JSON fields stored in PostgreSQL JSONB columns. Select specific nested fields from the JSON data, and FastGQL extracts only the requested fields using PostgreSQL's native operators.

### Setup

Define the structure of your JSON data with the `@json` directive:

```graphql
type ProductAttributes {
    color: String
    size: Int
    details: ProductDetails
}

type ProductDetails {
    manufacturer: String
    warranty: WarrantyInfo
}

type WarrantyInfo {
    years: Int
    provider: String
}

type Product @table(name: "products") {
    id: Int!
    name: String!
    attributes: ProductAttributes @json(column: "attributes")
}
```

### Examples

**Select scalar fields:**
```graphql
query {
  products {
    name
    attributes {
      color
      size
    }
  }
}
```

**Select nested objects:**
```graphql
query {
  products {
    name
    attributes {
      color
      details {
        manufacturer
      }
    }
  }
}
```

**Deep nesting:**
```graphql
query {
  products {
    name
    attributes {
      details {
        warranty {
          years
        }
      }
    }
  }
}
```

### How It Works

FastGQL uses PostgreSQL's `->` operator for field extraction and `jsonb_build_object` to construct the response. Only the fields specified in your GraphQL query are extracted from the database, making queries efficient even with large JSON objects.

### Limitations

- Only works with typed JSON fields (fields with `@json` directive and a GraphQL object type)
- For `Map` scalar type, the entire JSON value is always returned
- Field selection is distinct from filtering - see [JSON Filtering](filtering#json-filtering)

For a complete example, see [examples/json](https://github.com/roneli/fastgql/tree/master/examples/json).
