---
title: "Sort Query Results"
linkTitle: "Sorting"
date: 2021-02-07
description: >
    Sorting query results
weight: 5
---

You can sort your query results using the `orderby` . The sort argument can be used to sort nested queries too.

The value of sort argument should be an array containing the name of fields to sort the results by.

### Sorting simple queries

Example: Sort all posts by their `name`

```graphql
query {
    posts(orderBy: {name: ASC}) {
        name
    }
}
```

### Sorting multiple fields

Example: Sort all posts by their `name` and then by `id` 

```graphql
query {
    posts(orderBy: [{name: ASC}, {id: DESC}]) {
        name
    }
}
```

### Sorting nested queries

Example: Sort all posts by `name` in DESC ordering, for each post sort its categories `name` by ASC order

```graphql
query {
    posts(orderBy: {name: DESC}) {
        name
        categories(orderBy: {name: ASC_NULL_FIRST}) {
            name
        }
    }
}
```