var data {{.Field.TypeReference.GO | ref}}
if err := r.Executor.Scan(ctx, {{.Dialect | quote}}, &data); err != nil {
    return nil, err
}
return data, nil