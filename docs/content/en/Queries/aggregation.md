---
title: "Aggregate Query Results"
linkTitle: "Aggregation"
date: 2021-02-016
description: >
    Aggregate query results
weight: 6
---

We can aggregate our query results using the `_[fieldName]Aggergate` field. The field aggregate is added automatically 
by fastGQL based on the `@generate` directive in our schema.

### Aggregate queries

Example: Aggregate all posts

```graphql
query {
    _postsAggregate {
        count
    }
}
```

### Aggregate queries filtering

Similar to object fields, aggregate fields have a `filter` argument allowing us to filter out aggregate result.  

Example: Count all posts that have a category with id `1`

```graphql
query {
    _postsAggregate(filter: {categories: {id:{eq: 1}}}) {
        count
    }
}
```
