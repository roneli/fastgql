{{ reserveImport "github.com/georgysavva/scany/pgxscan" }}
{{ reserveImport "github.com/roneli/fastgql/builders" }}
{{ reserveImport 	"github.com/roneli/fastgql/builders/sql" }}

builder := sql.NewBuilder(r.Cfg)
{{ if hasSuffix .Field.Name "Aggregate" }} q, args, err := builder.Aggregate(ctx) {{else}} q, args, err := builder.Query(ctx) {{end}}
if err != nil {
    return nil, err
}
rows, err := r.Executor.Query(ctx, q, args...)
if err != nil {
    return nil, err
}
var data {{.Field.TypeReference.GO | ref}}
if err := {{ if hasSuffix .Field.Name "Aggregate" }} pgxscan.ScanOne(&data, rows) {{else}} pgxscan.ScanAll(&data, rows) {{end}}; err != nil {
    return nil, err
}
return data, nil
