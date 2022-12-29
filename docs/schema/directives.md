# Directives

In this section we will go over all supported directives *fastGQL* has to offer, what they do and how to use them.

These directives are found added automatically if you use `go run github.com/roneli/fastgql init`
or you can copy the fastgql.graphql [fragment]({{< ref "/schema/fragment" >}} )

## Augmentation directives

These directives are commonly used to tell *fastgql* what parts of the schema we want to augment.

We have three augmentation directives:
- [@generateFilterInput]({{< ref "#generatefilterinput" >}})
- [@generate]({{< ref "#generate" >}})
- [@skipGenerate]({{< ref "#generatefilterinput" >}})

### @generateFilterInput

The `@generateFilterInput` tells the augmenter on which `OBJECT | INTERFACE` to generate filter inputs, this is required
so *fastGQL* knows what objects to build filters for giving you more control over which filters are created.

```graphql
# Generate filter input on an object
directive @generateFilterInput(name: String!, description: String) on OBJECT | INTERFACE
```

**Example**:

In this example, we can are creating a filter input for the category type, when arguments will be generated
for categories in `Query` the CategoryFilterInput will be used for the `filter` argument.

```graphql
type Category @generateFilterInput(name: "CategoryFilterInput"){
  id: Int!
  name: String
}

type Query @generate {
    categories: [Category]
}
```

This also works for objects that contain fields of objects, see [Object filters](/queries/filter/#object-filters)

### @generate

The `@generate` tells the augmenter on which `OBJECT ` to generate arguments i.e filter, pagination and ordering, aggregation.
By default, all arguments created, and are recursive, this means that args are generated from the Top level all until exhausted (no object fields aren't augmented)

```graphql
# Generate arguments for a given field or all object fields
directive @generate(filter: Boolean = True, pagination: Boolean = True, ordering: Boolean = True, aggregate: Boolean = True, recursive: Boolean = True) on OBJECT
```

**Example**: The following example adds arguments to all fields in Query and does the same for each field type.

```graphql
type Query @generate {
  posts: [Post]
  users: [User]
  categories: [Category] @skipGenerate
}
```


### @skipGenerate

The `@skipGenerate` tells the codegen whether we want to skip creating the resolver code or not.

```graphql
# Tells fastgql to skip resolver generation on field
directive @skipGenerate(resolver:Boolean = True) on FIELD_DEFINITION
```


**Example:**

```graphql

type Query @generate {
    posts: [Post]
    users: [User]
    categories: [Category] @skipGenerate
}
```

## Builder directives

Builder directives are used by builders to build queries based on the given graphQL query requested.

We have two builder directives:
- [@tableName]({{< ref "#generateArguments" >}})
- [@sqlRelation]({{< ref "#sqlRelation" >}})

### @table

The `@table` directive is used to define the name of the table of the graphql `OBJECT` in the database.

By default, the table name is the name of type, we use this directive when the table name is different from the object typename.

If our table resides in a schema that isn't the one set as the default we need to add the schema argument.
```graphql
# Used if Object/Interface type name is different then the actual table name or if the table resides in a schema other than default path.
directive @table(name: String!, schema: String) on OBJECT | INTERFACE
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
- `ONE_TO_ONE`
- `ONE_TO_MANY`
- `MANY_TO_MANY`