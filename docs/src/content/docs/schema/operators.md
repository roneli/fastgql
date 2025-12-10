---
title: Custom Operators   
description: fastGQL custom operators type
---

FastGQL defines a set of operators that can be used to extend the schema. These operators are defined in the `fastgql.graphql` file.
These operators are used to run filters and queries on the database.

## Standard Operators

By default, FastGQL supports these operators for scalar types (String, Int, Float, etc.):

- `eq` - equals
- `neq` - not equals
- `gt` - greater than
- `lt` - less than
- `gte` - greater than or equal
- `lte` - less than or equal
- `in` - in
- `notIn` - not in
- `like` - like (string matching with wildcards)
- `ilike` - case-insensitive like
- `isNull` - is null
- `suffix` - string ends with
- `prefix` - string starts with

## List/Array Operators

The following operators are available specifically for list/array types (StringListComparator, IntListComparator, etc.):

- `contains` - array contains value(s)
- `notContains` - array does not contain value(s)
- `overlap` - arrays have overlapping elements
- `containedBy` - array is contained by another array

**Note:** These list operators are NOT available for scalar comparators like StringComparator or IntComparator.

## JSON Filtering Operators

FastGQL provides specialized operators for filtering PostgreSQL JSONB columns through two input types:

### MapComparator

The `MapComparator` input type is used to filter fields of type `Map` (dynamic JSON data). It provides several operators for querying JSON content:

```graphql
input MapComparator {
    contains: Map
    where: [MapPathCondition!]
    whereAny: [MapPathCondition!]
    isNull: Boolean
}
```

**Operators:**

- **`contains`**: Performs a partial JSON match using PostgreSQL's `@>` containment operator. The JSON in the database must contain all the key-value pairs specified in the filter.

  ```graphql
  query {
    products(filter: { metadata: { contains: { discount: "true" } } }) {
      name
    }
  }
  ```

- **`where`**: Accepts an array of JSONPath conditions that are combined with AND logic. All conditions must be true for a record to match.

  ```graphql
  query {
    products(filter: {
      metadata: {
        where: [
          { path: "price", lt: 100 },
          { path: "discount", eq: "true" }
        ]
      }
    }) {
      name
    }
  }
  ```

- **`whereAny`**: Accepts an array of JSONPath conditions that are combined with OR logic. At least one condition must be true for a record to match.

  ```graphql
  query {
    products(filter: {
      metadata: {
        whereAny: [
          { path: "rating", gt: 4 },
          { path: "discount", eq: "true" }
        ]
      }
    }) {
      name
    }
  }
  ```

- **`isNull`**: Checks if the JSON field is NULL.

  ```graphql
  query {
    products(filter: { metadata: { isNull: false } }) {
      name
    }
  }
  ```

**Combining operators:**

You can combine multiple operators in a single filter. They are combined with AND logic:

```graphql
query {
  products(filter: {
    metadata: {
      contains: { discount: "true" },
      where: [{ path: "price", lt: 75 }]
    }
  }) {
    name
  }
}
```

### MapPathCondition

The `MapPathCondition` input type defines a single condition for JSONPath-based filtering:

```graphql
input MapPathCondition {
    path: String!
    eq: String
    neq: String
    gt: Float
    gte: Float
    lt: Float
    lte: Float
    like: String
    isNull: Boolean
}
```

**Fields:**

- **`path`** (required): The JSON path to the field you want to filter. Supports nested fields and array indices:
  - Simple field: `"price"`
  - Nested field: `"address.city"`
  - Array index: `"items[0]"`
  - Complex path: `"items[0].details.name"`

**Operators:**

- **`eq`**: Equals (string comparison)
- **`neq`**: Not equals
- **`gt`**: Greater than (numeric comparison)
- **`gte`**: Greater than or equal
- **`lt`**: Less than
- **`lte`**: Less than or equal
- **`like`**: Pattern matching using PostgreSQL regex
- **`isNull`**: Check if the field at the path is null

**Path validation:**

For security, paths are validated to prevent SQL injection. Valid paths must:
- Start with a letter or underscore
- Contain only alphanumeric characters, underscores, dots (for nesting), and bracket notation for arrays
- Array indices must be non-negative integers

**Examples:**

Valid paths:
- `price`
- `nested.field`
- `items[0]`
- `items[0].name`
- `address.details.city`

Invalid paths (will be rejected):
- `$.field` (JSONPath operators not allowed)
- `field; DROP TABLE` (SQL injection attempt)
- `items[-1]` (negative indices)
- `items[*]` (wildcards not supported in path specification)

## Adding Custom Operators

FastGQL allows you to add custom operators to the schema. This can be done by defining a new input type in the `fastgql.graphql` file, 
or by adding a new operator to an existing input type, by extending the input type.

```graphql

# extend our input type to include a custom operator
extend input StringComparator  {
    # The custom operator to use for the comparison.
    myCustomOperator: String
}
```

We will define our custom operator to always do "1 = 1"

```go
func MyCustomOperator(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return goqu.L("1 = 1")
}
```

In our `serve.go` where we define our schema, we will add a resolver for the custom operator.

```go
	cfg := &builders.Config{
        Schema: executableSchema.Schema(),
        Logger: adapters.NewZerologAdapter(log.Logger),
        CustomOperators: map[string]builders.Operator{
            "myCustomOperator": MyCustomOperator,
        },
}
```



An example of a custom operator can be found [here](https://github.com/roneli/fastgql/tree/master/examples/custom_operator).


## Adding FilterInputs for Custom Scalar Types

Similar to how StringComparator is used to filter strings, you can define a custom input type to filter custom scalar types.

```graphql

# Define a custom scalar type
scalar MyCustomScalar

# Define a custom input type to filter the custom scalar type
input MyCustomScalarComparator {
    eq: MyCustomScalar
    # The custom operator to use for the comparison.
    myCustomOperator: MyCustomScalar
}
```

