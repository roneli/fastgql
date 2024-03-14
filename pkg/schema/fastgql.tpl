{{- reserveImport "github.com/georgysavva/scany/v2/pgxscan" -}}
{{- reserveImport "github.com/jackc/pgx/v5" }}
{{- reserveImport "github.com/roneli/fastgql/pkg/execution/builders/sql" -}}

{{- if or (hasPrefix .Field.Name "create") (hasPrefix .Field.Name "delete") (hasPrefix .Field.Name "update") -}}
var data {{.Field.TypeReference.GO | deref}}
q, args, err := sql.BuildQuery(ctx, sql.NewBuilder(r.Cfg))
if err != nil {
    return nil, err
}
if err := sql.ExecuteQuery(ctx, r.Executor, func(rows pgx.Rows) error {
    return pgxscan.ScanOne(&data, rows)
}, q, args...); err != nil {
    return nil, err
}
return &data, nil
{{- else if eq .Field.TypeReference.Definition.Kind  "INTERFACE" -}}
scanner := execution.NewTypeNameScanner[{{.FieldType | ref}}](map[string]reflect.Type{
{{- range  $key, $value := .Implementors }}
    {{$key|quote}}: reflect.TypeOf({{$value.Type | deref}}{}),
{{- end -}}
}, {{.ImplementorsTypeName|quote}})
q, args, err := sql.BuildQuery(ctx, sql.NewBuilder(r.Cfg))
if err != nil {
    return nil, err
}
return sql.Collect[{{.FieldType | ref}}](ctx, r.Executor, func(row pgx.CollectableRow) ({{.FieldType | ref}}, error) {
    return scanner.ScanRow(row)
}, q, args...)
{{- else -}}
var data {{.Field.TypeReference.GO | ref}}
q, args, err := sql.BuildQuery(ctx, sql.NewBuilder(r.Cfg))
if err != nil {
    return nil, err
}
if err := sql.ExecuteQuery(ctx, r.Executor, func(rows pgx.Rows) error {
    return pgxscan.ScanAll(&data, rows)
}, q, args...); err != nil {
    return nil, err
}
return data, nil
{{- end -}}

