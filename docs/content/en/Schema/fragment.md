---
title: "Fragment"
linkTitle: "Fragments"
date: 2021-02-02
description: >
    While editing your schema, this GraphQL schema fragment can be useful. It sets up the definitions of the directives that you’ll use in your schema.
---

While editing your schema, you might find it useful to include this GraphQL schema fragment.  It sets up the definitions of the directives, etc. (like `@sqlRelation`) that you'll use in your schema.  If your editor is GraphQL aware, it may give you errors if you don't have this available and context sensitive help if you do.

```graphql
# Used if Object/Interface type name is different then the actual table name
directive @tableName(name: String!) on OBJECT | INTERFACE

# Generate filter input on an object
directive @generateFilterInput(name: String!, description: String) on OBJECT | INTERFACE

# Generate arguments for a given field or all object fields
directive @generate(filter: Boolean = True, pagination: Boolean = True, ordering: Boolean = True, aggregate: Boolean = True, recursive: Boolean = True) on OBJECT


enum _relationType {
    ONE_TO_ONE
    ONE_TO_MANY
    MANY_TO_MANY
}

directive @sqlRelation(relationType: _relationType!, baseTable: String!, refTable: String!, fields: [String!]!,
    references: [String!]!, manyToManyTable: String = "", manyToManyFields: [String] = [], manyToManyReferences: [String] = []) on FIELD_DEFINITION

directive @skipGenerate(resolver:Boolean = True) on FIELD_DEFINITION

enum _OrderingTypes {
    ASC
    DESC
    ASC_NULL_FIRST
    DESC_NULL_FIRST
}

input StringComparator {
    eq: String
    neq: String
    contains: [String]
    not_contains: [String]
    like: String
    ilike: String
    suffix: String
    prefix: String
}

input StringListComparator {
    eq: [String]
    neq: [String]
    contains: [String]
    containedBy: [String]
    overlap: [String]
}

input IntComparator {
    eq: Int
    neq: Int
    gt: Int
    gte: Int
    lt: Int
    lte: Int
}

input IntListComparator {
    eq: [Int]
    neq: [Int]
    contains: [Int]
    contained: [Int]
    overlap: [Int]
}

input FloatComparator {
    eq: Float
    neq: Float
    gt: Float
    gte: Float
    lt: Float
    lte: Float
}

input FloatListComparator {
    eq: [Float]
    neq: [Float]
    contains: [Float]
    contained: [Float]
    overlap: [Float]
}

input BooleanComparator {
    eq: Boolean
    neq: Boolean
}

input BooleanListComparator {
    eq: [Boolean]
    neq: [Boolean]
    contains: [Boolean]
    contained: [Boolean]
    overlap: [Boolean]
}
```