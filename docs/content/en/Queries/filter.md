---
title: "Filtering"
linkTitle: "Filtering"
date: 2021-02-07
description: >
    Filter query results
weight: 2
---

We can use `filter` in our queries to filter results based on some field’s values.  
we can use multiple filters in the same where clause using AND, OR, NOT.

## Simple Operators

**Example**: fetch posts with `id` equal to 1

```graphql
query {
  posts(limit: 3, filter: { id: { eq: 3 } }) {
    id
  }
}
```

## Using multiple filters in the same query

You can group multiple parameters in the same where clause using the 
logical AND, OR and NOT operations. These logical operations can be infinitely nested to apply complex filters.

### Example: AND

By default, a logical AND operation is performed on all the conditions mentioned in the filter clause. If we want to use the same field key, then we use the AND operator in GraphQL.

Let's say we want to filter `posts` endpoint. This is how we would do it:

{{< tabs tabTotal="2" tabID="1" tabName1="Multi" tabName2="Default">}}
{{< tab tabNum="1" >}}

```graphql
query {
    posts(
        # expected filter: name = "article 1" AND name like "%article%" AND id = 1
        filter: {
            AND: [
                { name: { eq: "article 1" } }
                { name: { like: "%article%" } }
                { id: { eq: 1 } }
            ]
        }
    ) {
        id
        name
    }
}
```

{{< /tab >}}
{{< tab tabNum="2" >}}

```graphql
query {
    # expected filter: name = "article 1" AND id = 1
    posts(filter: { name: { eq: "article 1" }, id: { eq: 1 } }) {
        id
        name
    }
}
```
{{< /tab >}}
{{< /tabs >}}

### Example: OR

To perform a logical OR operation, we have to use the OR operator in GraphQL.

Let’s say we want to fetch information of all post's name that start with "a" or ends with "b". This is how you would do it:
```graphql
query {
    posts(
        filter: { OR: [{ name: { prefix: "a" } }, { name: { suffix: "b" } }] }
    ) {
        id
        name
    }
}
```

### Example NOT

To perform a logical NOT operation, we have to use the NOT operator in GraphQL.

Let’s say we want to fetch information of all post's name that **don't** start with "a" or ends with "b". This is how you would do it:
```graphql
query {
    posts(
        filter: {
            NOT: { OR: [{ name: { prefix: "a" } }, { name: { suffix: "b" } }] }
        }
    ) {
        id
        name
    }
}
```

### Example Nested logical

We can also perform complex nested logical operation such as OR inside AND or [NOT on OR operator]({{< ref "#example-not" >}}) 

```graphql
query {
    posts(
        filter: {
            AND: [
                { OR: [{ name: { prefix: "a" } }, { name: { suffix: "b" } }] }
                { name: { like: "%c%" } }
            ]
        }
    ) {
        id
        name
    }
}

```
## Object filters

Most object usually have fields that aren't scalars, and we would like to filter by them.
In our example each post has one or more categories to filter by them by categories the following filter
will be used:

```graphql
query ObjectFilterExample {
    posts(filter: {categories: {name: {eq: "Security"}}}) {
        id
        name
    }
}
```

##

