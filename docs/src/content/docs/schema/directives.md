---
title: Directives
description: fastGQL supported directives
---

In this section we will go over all the custom _fastGQL_ directives, what they do and how to use them. These directives are added automatically when we augment and generate the server code. You can also use the following command `go run github.com/roneli/fastgql init` or copy the [fastgql\_schema\_fragment.md](fastgql\_schema\_fragment.md "mention").

## Augmentation directives

These directives extend the original schema functionality adding filters, pagination, aggregation, ordering etc' and are commonly used to tell _fastgql_ what parts of the schema we want to augment.

We currently have four augmentation directives:

* [#generatefilterinput](directives#generatefilterinput "mention")
* [#generate](directives#generate "mention")
* [#generateMutations](directives#generatemutations "mention")



### @generateFilterInput

The `@generateFilterInput` tells the augmenter on which `OBJECT | INTERFACE` to generate filter inputs, this is required so _fastGQL_ knows what objects to build filters for giving you more control over which filters are created.

```graphql
# Generate filter input on an object
directive @generateFilterInput(description: String) 
    on OBJECT | INTERFACE
```

**Example**:

In this example, we can are creating a filter input for the category type, when arguments will be generated for categories in `Query` the CategoryFilterInput will be used for the `filter` argument.

```graphql
type Category @generateFilterInput {
  id: Int!
  name: String
}

type Query {
    categories: [Category] @generate
}
```

This also works for objects that contain fields of objects, see [Object filters](../../../queries/filter/#object-filters)

### @generate

The `@generate` tells the augmenter on which `FIELD_DEFINTION` to generate arguments i.e. filter, pagination and ordering, aggregation. By default, all arguments are created, and are recursive, this means that arguments are generated from the top level `OBJECT` until exhausted meaning no new OBJECT fields are found in a certain level.

```graphql
# Generate arguments for a given field or all object fields
directive @generate(filter: Boolean = True, pagination: Boolean = True, ordering: Boolean = True, aggregate: Boolean = True, recursive: Boolean = True, filterTypeName: String) on FIELD_DEFINITION
```

**Example**: The following example generates resolvers for posts and users, but doesn't add aggregation to users.

```graphql
type Query {
  posts: [Post] @generate
  users: [User] @generate(aggregate: false)
  categories: [Category]
}
```

### @generateMutations

The `@generateMutations` tells the augmenter on which `OBJECT` to generate mutations on. 
There are 3 possible mutations, create, update and delete, by default all of them are set to true. 

```graphql
# Generate filter input on an object
directive @generateMutations(create: Boolean = True, delete: Boolean = True, update: Boolean = True) on OBJECT
```

## Builder directives

Builder directives are used by builders to build queries based on the given GraphQL query requested.

We have the following builder directives:

* [#table](directives#table "mention")
* [#relation](directives#relation "mention")
* [#typename](directives#typename "mention")
* [#json](directives#json "mention")
* [#fastgqlfield](directives#fastgqlfield "mention")

### @table

The `@table` directive is used to define the name of the table of the graphql `OBJECT` in the database.

By default, the table name is the name of type, we use this directive when the table name is different from the object typename.

If our table resides in a schema that isn't the one set as the default we need to add the schema argument.

```graphql
# Used if Object/Interface type name is different then the actual table name or if the table resides in a schema other than default path.
directive @table(name: String!, dialect: String, schema: String) on OBJECT | INTERFACE
```

**Example:**

```graphql

type User @table(name: "users", dialect: "postgres", schema: "app"){
    id: Int
    name: String
}
```

### @relation

The `@relation` directive tells the builder about relations in your database.

```graphql
directive @relation(type: _relationType!, fields: [String!]!, references: [String!]!, manyToManyTable: String = "", 
     manyToManyFields: [String] = [], manyToManyReferences: [String] = []) on FIELD_DEFINITION
```

#### Table Relationships

There are three major types of database relationships:

* <mark style="color:purple;">`ONE_TO_ONE`</mark>
* <mark style="color:purple;">`ONE_TO_MANY`</mark>
* <mark style="color:purple;">`MANY_TO_MANY`</mark>

### @typename 

The `@typename` directive is used for interface support (experimental), the typename tell fastgql builder what field in the table we should use
to use when scanning into the original type of the interface

```graphql
interface Animal @table(name: "animals") @typename(name: "type") @generateFilterInput {
	id: Int!
	name: String!
	type: String!
}
```

### @fastgqlField

The `@fastgqlField` directive is used to mark fields that should be skipped during SELECT query generation. This is useful for fields that are not actual columns in the database and need to be resolved manually in your resolver code.

```graphql
# Mark a field to be skipped in SELECT queries
directive @fastgqlField(skipSelect: Boolean = True) on FIELD_DEFINITION
```

**Example:**

In this example, the `fullName` field is computed from other fields and doesn't exist as a database column. We use `@fastgqlField` to tell fastGQL to skip it when building SELECT queries:

```graphql
type User @table(name: "users") @generateFilterInput {
    id: Int!
    firstName: String!
    lastName: String!
    fullName: String! @fastgqlField
}
```

You would then manually resolve the `fullName` field in your resolver:

```go
func (r *userResolver) FullName(ctx context.Context, obj *model.User) (string, error) {
    return obj.FirstName + " " + obj.LastName, nil
}
```

### @json

The `@json` directive marks a field as stored in a PostgreSQL JSONB column. This allows you to work with structured JSON data in your database while providing both type-safe filtering capabilities and efficient nested field selection in GraphQL.

```graphql
# Marks a field as stored in a JSONB column
directive @json(column: String!) on FIELD_DEFINITION
```

**Arguments:**
- `column` (required): The name of the JSONB column in the database table. By default, FastGQL converts GraphQL field names to snake_case for database columns, so you typically specify the snake_case column name here.

There are two approaches for working with JSON data in FastGQL:

#### 1. Typed JSON (Recommended)

For structured JSON data with a known schema, use the `@json` directive on a field with a GraphQL object type. FastGQL will automatically generate a FilterInput that allows type-safe filtering with the same operators available for regular fields.

**Example:**

```graphql
# Define the structure of your JSON data
type ProductAttributes {
    color: String
    size: Int
    tags: [String]
    details: ProductDetails
}

type ProductDetails {
    manufacturer: String
    model: String
}

type Product @generateFilterInput @table(name: "products") {
    id: Int!
    name: String!
    # Typed JSON field - supports filtering and nested field selection
    attributes: ProductAttributes @json(column: "attributes")
}

type Query {
    products: [Product] @generate
}
```

**Filtering:** This automatically generates a `ProductAttributesFilterInput` that you can use to filter:

```graphql
query {
    # Filter products where attributes.color == "red"
    products(filter: { attributes: { color: { eq: "red" } } }) {
        name
        attributes
    }
}
```

**Nested Field Selection:** You can select specific nested fields from the JSON data, and FastGQL will efficiently extract only the requested fields using PostgreSQL's native `->` operator:

```graphql
query {
    # Select only color and size from attributes
    products {
        name
        attributes {
            color
            size
        }
    }
}
```

```graphql
query {
    # Select nested object fields
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

FastGQL uses `jsonb_build_object` to construct the response matching your GraphQL query structure, extracting only the fields you request for optimal performance.

**Benefits:**
- Type-safe filtering with full GraphQL type validation
- Efficient nested field selection (only extracts requested fields)
- Supports all standard operators (eq, neq, gt, lt, etc.)
- Supports logical operators (AND, OR, NOT)
- Supports nested objects to any depth
- Auto-completion in GraphQL IDEs
- Uses native PostgreSQL operators for performance

#### 2. Map Scalar (Dynamic JSON)

For dynamic JSON data where the structure is not known at schema definition time, use the `Map` scalar type. This provides runtime filtering using JSONPath expressions.

**Example:**

```graphql
type Product @generateFilterInput @table(name: "products") {
    id: Int!
    name: String!
    # Dynamic JSON field - uses MapComparator for filtering
    metadata: Map
}
```

With `Map` scalar, the entire JSON value is returned as-is. You cannot select specific nested fields like with typed JSON.

See [MapComparator](../operators#mapcomparator) for filtering options with dynamic JSON.

**When to use which approach:**

- **Use Typed JSON (@json)** when:
  - Your JSON structure is known and consistent
  - You want type safety and validation
  - You need IDE auto-completion
  - You want to select specific nested fields efficiently
  - Your JSON data represents a well-defined domain object

- **Use Map scalar** when:
  - Your JSON structure varies between records
  - You need maximum runtime flexibility
  - You're storing arbitrary metadata or configuration
  - Your JSON structure is defined by users or external systems
  - You always need the entire JSON value

**Performance Note:** Typed JSON with `@json` directive uses PostgreSQL's native `->` operator for field extraction and `jsonb_build_object` for constructing the response. This is highly efficient and allows the database to extract only the fields specified in your GraphQL query.