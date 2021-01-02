opctx := graphql.GetOperationContext(ctx)
fctx := graphql.GetFieldContext(ctx)

builder := sql.NewBuilder(fctx.Field.Name, fctx.Field.Field)
err := resolvers.CollectFields(&builder, fctx.Field.Field, opctx.Variables)
if err != nil {
    return nil, err
}

q, _, _ := builder.Query()
fmt.Println(q)
return nil, nil
