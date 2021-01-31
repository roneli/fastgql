{{ reserveImport "github.com/georgysavva/scany/pgxscan" }}
{{ reserveImport "github.com/roneli/fastgql/builders" }}
{{ reserveImport 	"github.com/roneli/fastgql/builders/sql" }}

opCtx := graphql.GetOperationContext(ctx)
fCtx := graphql.GetFieldContext(ctx)

builder, _ := sql.NewBuilder(r.Cfg, fCtx.Field.Field)
err := builders.BuildQuery(&builder, fCtx.Field.Field, opCtx.Variables)
if err != nil {
    return nil, err
}

q, args, err := builder.Query()
if err != nil {
    return nil, err
}
rows, err := r.Sql.Query(ctx, q, args...)
if err != nil {
    return nil, err
}

var data {{.Field.TypeReference.GO | ref}}
if err := pgxscan.ScanAll(&data, rows); err != nil {
    return nil, err
}
return data, nil
