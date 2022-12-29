---
description: Sort query results
---

# Ordering

We can sort query results using the `orderBy` argument that is added when extending the schema with fastGQL. The value of the sort argument should be an array containing the name of fields to sort the result by.

{% hint style="info" %}
Ordering is ALPHA so it API might change in future versions
{% endhint %}

## Simple Sort

```graphql
query {
    posts(orderBy: {name: ASC}) {
        name
    }
}
```

## Multiple fields

```graphql
query {
    posts(orderBy: [{name: ASC}, {id: DESC}]) {
        name
    }
}
```

## Sorting Nested Queries

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
