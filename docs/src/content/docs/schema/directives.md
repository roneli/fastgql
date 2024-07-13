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
directive @generateFilterInput(name: String!, description: String) 
    on OBJECT | INTERFACE
```

**Example**:

In this example, we can are creating a filter input for the category type, when arguments will be generated for categories in `Query` the CategoryFilterInput will be used for the `filter` argument.

```graphql
type Category @generateFilterInput(name: "CategoryFilterInput"){
  id: Int!
  name: String
}

type Query @generate {
    categories: [Category]
}
```

This also works for objects that contain fields of objects, see [Object filters](../../../queries/filter/#object-filters)

### @generate

The `@generate` tells the augmenter on which `OBJECT` to generate arguments i.e. filter, pagination and ordering, aggregation. By default, all arguments are created, and are recursive, this means that arguments are generated from the top level `OBJECT` until exhausted meaning no new OBJECT fields are found in a certain level.

```graphql
# Generate arguments for a given field or all object fields
directive @generate(filter: Boolean = True, 
  pagination: Boolean = True, ordering: Boolean = True, 
  aggregate: Boolean = True, recursive: Boolean = True, filterTypeName: String) on FIELD_DEFINITION 
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

We have two builder directives:

* [#table](directives#table "mention")
* [#relation](directives#relation "mention")
* [#typename](directives#typename "mention")

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