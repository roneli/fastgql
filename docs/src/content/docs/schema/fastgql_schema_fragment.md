---
title: FastGQL Schema Fragment
description: fastGQL schema fragment
---
# FastGQL Schema Fragment

While editing your schema, you might find it useful to include this GraphQL schema
fragment. It sets up the definitions of the directives, etc. 
(like @generate) that you’ll use in your schema. 
If your editor is GraphQL aware, it may give you errors if you don’t have 
this available. 

```graphql

# Table directive is defined on OBJECTS, if no table directive is defined defaults are assumed
# i.e <type_name>, "postgres", ""
directive @table(name: String!, dialect: String! = "postgres", schema: String = "") on OBJECT | INTERFACE

# Relation directive defines relations cross tables and dialects
directive @relation(type: _relationType!, fields: [String!]!, references: [String!]!, manyToManyTable: String = "", manyToManyFields: [String] = [], manyToManyReferences: [String] = []) on FIELD_DEFINITION

# used to skip generate on a certain field, this should be used if we use recursive = True in @generate
directive @skipGenerate(resolver: Boolean = True) on FIELD_DEFINITION

# Generate filter input on an object
directive @generateFilterInput(name: String!, description: String) repeatable on OBJECT | INTERFACE

# Generate arguments for a given field or all object fields
directive @generate(filter: Boolean = True, pagination: Boolean = True, ordering: Boolean = True, aggregate: Boolean = True,
    recursive: Boolean = True, wrapper:Boolean = False) repeatable  on OBJECT

# Generate mutations object
directive @generateMutations(create: Boolean = True, delete: Boolean = True, update: Boolean = True) on OBJECT

enum _relationType {
    ONE_TO_ONE
    ONE_TO_MANY
    MANY_TO_MANY
}

enum _OrderingTypes {
    ASC
    DESC
    ASC_NULL_FIRST
    DESC_NULL_FIRST
}

type _AggregateResult {
    count: Int!
}

input StringComparator {
    eq: String
    neq: String
    contains: [String]
    notContains: [String]
    like: String
    ilike: String
    suffix: String
    prefix: String
    isNull: Boolean
}

input StringListComparator {
    eq: [String]
    neq: [String]
    contains: [String]
    containedBy: [String]
    overlap: [String]
    isNull: Boolean
}

input IntComparator {
    eq: Int
    neq: Int
    gt: Int
    gte: Int
    lt: Int
    lte: Int
    isNull: Boolean
}

input IntListComparator {
    eq: [Int]
    neq: [Int]
    contains: [Int]
    contained: [Int]
    overlap: [Int]
    isNull: Boolean
}

input FloatComparator {
    eq: Float
    neq: Float
    gt: Float
    gte: Float
    lt: Float
    lte: Float
    isNull: Boolean
}

input FloatListComparator {
    eq: [Float]
    neq: [Float]
    contains: [Float]
    contained: [Float]
    overlap: [Float]
    isNull: Boolean
}


input BooleanComparator {
    eq: Boolean
    neq: Boolean
    isNull: Boolean
}

input BooleanListComparator {
    eq: [Boolean]
    neq: [Boolean]
    contains: [Boolean]
    contained: [Boolean]
    overlap: [Boolean]
    isNull: Boolean
}

```