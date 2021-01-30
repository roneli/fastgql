---
title: "Paginate Query Results"
linkTitle: "Pagination"
date: 2021-02-02
description: >
    Paginate query results
weight: 3
---

The operators `limit` and `offset` are used for pagination.

limit specifies the number of rows to retain from the result set and `offset` determines the number of objects to skip (i.e. the offset of the result)

The following are examples of different pagination scenarios:

### Limit Results

Example: Fetch the first 5 posts from list of posts

```graphql
query {
    posts(limit: 5) {
        name
        id
    }
}
```

### Offset and Limit Results

Example: Fetch the first 5 posts from list of posts, skip first 3

```graphql
query {
    posts(limit: 5, offset: 3) {
        name
        id
    }
}
```

### Offset and Limit in nested queries

Example: Fetch the first 5 posts from list of posts, skip first 3, and fetch only the first category of each post

```graphql
query {
    posts(limit: 5, offset: 3) {
        name
        id
        categories(limit: 1) {
            name
        }
    }
}
```