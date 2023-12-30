---
title: Aggregation
description: Execute aggregation queries
---

FastGQL auto extends the schema using the `@generate` directive to add aggregation queries to the extended types. Aggregation results can be accessed using the `_[fieldName]Aggregate` field in the GraphQL query.&#x20;

## Count Aggregate

```graphql
query {
    _postsAggregate {
        count
    }
}
```

## Aggregate Filter

Similar to [filter ](filtering.mdx)queries we can filter our aggregate queries to returned different results. The following example will count all posts that have a category with the `id == 1` .

```graphql
query {
    _postsAggregate(filter: {categories: {id:{eq: 1}}}) {
        count
    }
}
```
