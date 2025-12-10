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

FastGQL supports efficient nested field selection for typed JSON fields stored in PostgreSQL JSONB columns. When you define a typed GraphQL object for a field with the `@json` directive, you can select specific nested fields just like you would with regular object types, and FastGQL will extract only the requested fields from the JSON data.

### Setup

First, define the structure of your JSON data as GraphQL types and mark the field with the `@json` directive:

```graphql
type ProductDetails {
    manufacturer: String
    model: String
    warranty: WarrantyInfo
}

type WarrantyInfo {
    years: Int
    provider: String
}

type ProductAttributes {
    color: String
    size: Int
    details: ProductDetails
}

type Product @table(name: "products", schema: "app") {
    id: Int!
    name: String!
    # Typed JSON field - supports nested field selection
    attributes: ProductAttributes @json(column: "attributes")
}

type Query {
    products: [Product] @generate
}
```

### Simple Scalar Selection

Select only specific scalar fields from the JSON data:

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

FastGQL extracts only `color` and `size` from the JSON column, ignoring other fields that may exist in the data.

### Nested Object Selection

Select fields from nested objects within the JSON:

```graphql
query {
  products {
    name
    attributes {
      color
      details {
        manufacturer
        model
      }
    }
  }
}
```

This query selects the `color` field and specific fields from the nested `details` object.

### Deep Nesting

FastGQL supports field selection at any nesting depth:

```graphql
query {
  products {
    name
    attributes {
      details {
        warranty {
          years
          provider
        }
      }
    }
  }
}
```

This extracts data three levels deep: `attributes` -> `details` -> `warranty`.

### Mixed Scalar and Nested Fields

Combine scalar and nested object selections in the same query:

```graphql
query {
  products {
    name
    attributes {
      color
      size
      details {
        manufacturer
      }
    }
  }
}
```

### How It Works

Under the hood, FastGQL uses PostgreSQL's native operators for efficient JSON field extraction:

- **PostgreSQL `->` operator**: Used for extracting nested fields from JSONB columns
- **`jsonb_build_object`**: Constructs the response JSON matching your GraphQL query structure
- **Efficient projection**: Only the fields specified in your GraphQL query are extracted from the database

This means that selecting specific fields is not just a GraphQL feature but is pushed down to the database level, making queries more efficient especially when dealing with large JSON objects.

### Performance Benefits

- Only requested fields are extracted from the JSON column
- Uses native PostgreSQL JSONB operators (highly optimized)
- Reduces data transfer between database and application
- Works efficiently even with deeply nested structures

### Complete Example

For a complete working example with database setup, test data, and various query patterns, see the `examples/json/` directory in the FastGQL repository:

- `examples/json/init.sql` - Database schema and test data
- `examples/json/graph/schema.graphql` - GraphQL schema definition
- `examples/json/README.md` - Setup instructions and test queries

### Limitations

- JSON field selection only works with typed JSON fields (fields with `@json` directive and a GraphQL object type)
- For dynamic JSON with the `Map` scalar type, the entire JSON value is always returned
- Field selection is distinct from filtering - see [JSON Filtering](filtering.mdx#json-filtering) for how to filter rows based on JSON content
