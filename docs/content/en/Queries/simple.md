---
title: "Simple Queries"
linkTitle: "Basic"
date: 2021-02-02
description: >
    Simple query examples
weight: 1
---

You can fetch a single or multiple objects of the same type using a simple object query.

### Fetch list of objects
**Example**: We want to fetch all users of the system

```graphql
query {
  users {
    name
  }
}
```

### Fetch nested objects
**Example**: We want to fetch all users and their posts

```graphql
query {
  users {
    name
    posts {
      name
    }
  }
}
```

### Fetch recursive nested

**Example**: We want to fetch all users and their posts, for each post want its categories and user who posted it.

```graphql
query {
  users {
    name
    posts {
      name
      categories {
        name
      }
      user {
        name
      }
    }
  }
}
```