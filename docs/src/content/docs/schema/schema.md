---
title: FastGQL Schema Fragment
description: fastGQL schema fragment
---

While editing your schema, you might find it useful to include this GraphQL schema
fragment. It sets up the definitions of the directives, etc. 
(like @generate) that you’ll use in your schema. 
If your editor is GraphQL aware, it may give you errors if you don’t have 
this available. 

### Fragment

```graphql

# ================== schema generation fastgql directives  ==================

# ================== schema generation fastgql directives  ==================

# Generate Resolver directive tells fastgql to generate an automatic resolver for a given field
# @generateResolver can only be defined on Query and Mutation fields.
# adding pagination, ordering, aggregate, filter to false will disable the generation of the corresponding arguments
# for filter to work @generateFilterInput must be defined on the object, if its missing you will get an error
# recursive will generate pagination, filtering, ordering and aggregate for all the relations of the object,
# this will modify the object itself and add arguments to the object fields.
directive @generate(filter: Boolean = True, pagination: Boolean = True, ordering: Boolean = True, aggregate: Boolean = True, recursive: Boolean = True, filterTypeName: String) on FIELD_DEFINITION

# Generate mutations for an object
directive @generateMutations(create: Boolean = True, delete: Boolean = True, update: Boolean = True) on OBJECT

# Generate filter input on an object
directive @generateFilterInput(description: String) repeatable on OBJECT | INTERFACE

directive @isInterfaceFilter on INPUT_FIELD_DEFINITION

# ================== Directives supported by fastgql for Querying ==================

# Table directive is defined on OBJECTS, if no table directive is defined defaults are assumed
# i.e <type_name>, "postgres", ""
directive @table(name: String!, dialect: String! = "postgres", schema: String = "") on OBJECT | INTERFACE

# Relation directive defines relations cross tables and dialects
directive @relation(type: _relationType!, fields: [String!]!, references: [String!]!, manyToManyTable: String = "", manyToManyFields: [String] = [], manyToManyReferences: [String] = []) on FIELD_DEFINITION

# This will make the field skipped in select, this is useful for fields that are not columns in the database, and you want to resolve it manually
directive @fastgqlField(skipSelect: Boolean = True) on FIELD_DEFINITION

# Typename is the field name that will be used to resolve the type of the interface,
# default model is the default model that will be used to resolve the interface if none is found.
directive @typename(name: String!) on INTERFACE

# =================== Default Scalar types supported by fastgql ===================
scalar Map
# ================== Default Filter input types supported by fastgql ==================

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
    ASC_NULL_LAST
    DESC_NULL_LAST
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