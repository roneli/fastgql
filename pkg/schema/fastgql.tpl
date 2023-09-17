{{- reserveImport "github.com/georgysavva/scany/v2/pgxscan" -}}
{{- reserveImport "github.com/jackc/pgx/v5" }}
{{- reserveImport "github.com/roneli/fastgql/pkg/execution/builders/sql" -}}
{{- if or (hasPrefix .Field.Name "create") (hasPrefix .Field.Name "delete") (hasPrefix .Field.Name "update") -}}
var data {{.Field.TypeReference.GO | deref}}
if err := r.Executor.Scan(ctx, {{.Dialect | quote}}, &data); err != nil {
    return nil, err
}
return &data, nil
{{- else }}
var data {{.Field.TypeReference.GO | ref}}
q, args, err := sql.BuildQuery(ctx, sql.NewBuilder(r.Cfg))
if err != nil {
    return nil, err
}
if err := sql.ExecuteQuery(ctx, nil, func(rows pgx.Rows) error {
    return pgxscan.ScanAll(&data, rows)
}, q, args...); err != nil {
    return nil, err
}
return data, nil
{{- end -}}

