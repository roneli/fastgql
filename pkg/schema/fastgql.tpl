{{- if or (hasPrefix .Field.Name "create") (hasPrefix .Field.Name "delete") (hasPrefix .Field.Name "update") -}}
var data {{.Field.TypeReference.GO | deref}}
if err := r.Executor.Execute(ctx, &data); err != nil {
    return nil, err
}
return &data, nil
{{- else if eq .Field.TypeReference.Definition.Kind  "INTERFACE" -}}
{{- reserveImport "reflect" -}}
var data {{.Field.TypeReference.GO | ref}}
if err := r.Executor.ExecuteWithTypes(ctx, &data, map[string]reflect.Type{
{{- range  $key, $value := .Implementors }}
    {{$key|quote}}: reflect.TypeOf({{$value.Type | deref}}{}),
{{- end -}}
}, {{.ImplementorsTypeName|quote}}); err != nil {
    return nil, err
}
return data, nil
{{- else -}}
var data {{.Field.TypeReference.GO | ref}}
if err := r.Executor.Execute(ctx, &data); err != nil {
    return nil, err
}
return data, nil
{{- end -}}

