{{- if or (hasPrefix .Field.Name "create") (hasPrefix .Field.Name "delete") }}
var data {{.Field.TypeReference.GO | deref}}
if err := r.Executor.Scan(ctx, {{.Dialect | quote}}, &data); err != nil {
    return nil, err
}
return &data, nil
{{- else }}
var data {{.Field.TypeReference.GO | ref}}
if err := r.Executor.Scan(ctx, {{.Dialect | quote}}, &data); err != nil {
    return nil, err
}
return data, nil
{{- end }}
