---
title: Pagination
description: Paginate your data with ease
---

The operators `limit` and `offset` are used for pagination of objects in our queries. `limit` specifies the number of rows to return from the result set, and `offset` determines the number of objects skip.&#x20;

The following are examples of different pagination queries:

### Limit Results

```graphql
query {
    posts(limit: 5) {
        name
        id
    }
}
```

Fetch the first 5 posts from a list of posts.

### Offset and Limit Results

```graphql
query {
    posts(limit: 5, offset: 3) {
        name
        id
    }
}
```

Fetch the first 5 posts from a list of posts, skip first 3.

### Offset and Limit Results in nested queries

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

Fetch the first 5 posts from list of posts, skip first 3, and fetch only the first category of each post.
