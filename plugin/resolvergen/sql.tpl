{{ reserveImport "github.com/georgysavva/scany/pgxscan" }}
{{ reserveImport "github.com/roneli/fastgql/builders" }}
{{ reserveImport 	"github.com/roneli/fastgql/builders/sql" }}

builder := sql.NewBuilder(r.Cfg)
{{ if hasSuffix .Field.Name "Aggregate" }}
    q, args, err := builder.Aggregate(builders.CollectFields(ctx))
{{ else if hasPrefix .Field.Name "create" }}
    q, args, err := builder.Create(builders.CollectFields(ctx))
{{else}}
    q, args, err := builder.Query(builders.CollectFields(ctx))
{{end}}
if err != nil {
    return nil, err
}
rows, err := r.Executor.Query(ctx, q, args...)
if err != nil {
    return nil, err
}

{{- if (hasPrefix .Field.Name "create") }}
var data {{.Field.TypeReference.GO|deref}}
if err := pgxscan.ScanOne(&data, rows); err != nil {
    return nil, err
}
return &data, nil
{{- else }}

var data {{.Field.TypeReference.GO | ref}}
if err := {{ if hasSuffix .Field.Name "Aggregate" }} pgxscan.ScanOne(&data, rows) {{else}} pgxscan.ScanAll(&data, rows) {{end}}; err != nil {
    return nil, err
}
return data, nil

{{- end }}
