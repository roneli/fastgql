---
title: Custom Operators   
description: fastGQL custom operators type
---

FastGQL defines a set of operators that can be used to extend the schema. These operators are defined in the `fastgql.graphql` file.
These operators are used to run filters and queries on the database. By default, FastGQL supports common operators:

- `eq` - equals
- `ne` - not equals
- `gt` - greater than
- `lt` - less than
- `gte` - greater than or equal
- `lte` - less than or equal
- `in` - in
- `nin` - not in
- `like` - like
- `ilike` - ilike
- `isNull` - is null
- `contains` - contains
- `notContains` - not contains
- `overlap` - overlap
- `containedBy` - contained by
- `suffix` - suffix
- `prefix` - prefix

## Adding Custom Operators

FastGQL allows you to add custom operators to the schema. This can be done by defining a new input type in the `fastgql.graphql` file, 
or by adding a new operator to an existing input type, by extending the input type.

```graphql

# extend our input type to include a custom operator
extend input StringComparator  {
    # The custom operator to use for the comparison.
    myCustomOperator: String
}
```

We will define our custom operator to always do "1 = 1"

```go
func MyCustomOperator(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return goqu.L("1 = 1")
}
```

In our `serve.go` where we define our schema, we will add a resolver for the custom operator.

```go
	cfg := &builders.Config{
        Schema: executableSchema.Schema(),
        Logger: adapters.NewZerologAdapter(log.Logger),
        CustomOperators: map[string]builders.Operator{
            "myCustomOperator": MyCustomOperator,
        },
}
```



An example of a custom operator can be found [here](https://github.com/roneli/fastgql/tree/master/examples/custom_operator).


## Adding FilterInputs for Custom Scalar Types

Similar to how StringComparator is used to filter strings, you can define a custom input type to filter custom scalar types.

```graphql

# Define a custom scalar type
scalar MyCustomScalar

# Define a custom input type to filter the custom scalar type
input MyCustomScalarComparator {
    eq: MyCustomScalar
    # The custom operator to use for the comparison.
    myCustomOperator: MyCustomScalar
}
```

