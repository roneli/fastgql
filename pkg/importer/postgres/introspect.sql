WITH columns_info AS (
    SELECT 
        table_schema,
        table_name,
        json_agg(
            json_build_object(
                'name', column_name,
                'type', data_type,
                'udt_name', udt_name,
                'is_nullable', CASE WHEN is_nullable = 'YES' THEN true ELSE false END,
                'is_json', CASE WHEN udt_name IN ('json', 'jsonb') THEN true ELSE false END,
                'is_array', CASE WHEN udt_name LIKE '\_%%' OR data_type = 'ARRAY' THEN true ELSE false END,
                'ordinal_position', ordinal_position
            ) ORDER BY ordinal_position
        ) AS columns,
        COUNT(*) AS column_count,
        array_agg(column_name ORDER BY ordinal_position) AS column_names
    FROM information_schema.columns
    WHERE table_schema = $1
    GROUP BY table_schema, table_name
),
primary_keys AS (
    SELECT 
        tc.table_schema,
        tc.table_name,
        array_agg(ku.column_name ORDER BY ku.ordinal_position) AS pk_columns
    FROM information_schema.table_constraints tc
    JOIN information_schema.key_column_usage ku
        ON tc.constraint_name = ku.constraint_name
        AND tc.table_schema = ku.table_schema
    WHERE tc.constraint_type = 'PRIMARY KEY'
        AND tc.table_schema = $1
    GROUP BY tc.table_schema, tc.table_name
),
unique_constraints AS (
    SELECT
        tc.table_schema,
        tc.table_name,
        tc.constraint_name,
        json_agg(
            json_build_object(
                'constraint_name', tc.constraint_name,
                'columns', (
                    SELECT array_agg(kcu.column_name ORDER BY kcu.ordinal_position)
                    FROM information_schema.key_column_usage kcu
                    WHERE kcu.constraint_name = tc.constraint_name
                        AND kcu.table_schema = tc.table_schema
                )
            )
        ) AS unique_constraints
    FROM information_schema.table_constraints tc
    WHERE tc.constraint_type = 'UNIQUE'
        AND tc.table_schema = $1
    GROUP BY tc.table_schema, tc.table_name, tc.constraint_name
),
foreign_keys AS (
    SELECT
        tc.table_schema,
        tc.table_name,
        json_agg(
            json_build_object(
                'constraint_name', tc.constraint_name,
                'columns', (
                    SELECT array_agg(kcu.column_name ORDER BY kcu.ordinal_position)
                    FROM information_schema.key_column_usage kcu
                    WHERE kcu.constraint_name = tc.constraint_name
                        AND kcu.table_schema = tc.table_schema
                ),
                'referenced_table_schema', (
                    SELECT ccu.table_schema
                    FROM information_schema.constraint_column_usage ccu
                    WHERE ccu.constraint_name = tc.constraint_name
                        AND ccu.table_schema = tc.table_schema
                    LIMIT 1
                ),
                'referenced_table', (
                    SELECT ccu.table_name
                    FROM information_schema.constraint_column_usage ccu
                    WHERE ccu.constraint_name = tc.constraint_name
                        AND ccu.table_schema = tc.table_schema
                    LIMIT 1
                ),
                'referenced_columns', (
                    SELECT array_agg(ccu.column_name ORDER BY kcu.ordinal_position)
                    FROM information_schema.constraint_column_usage ccu
                    JOIN information_schema.key_column_usage kcu
                        ON kcu.constraint_name = ccu.constraint_name
                        AND kcu.table_schema = ccu.table_schema
                    WHERE ccu.constraint_name = tc.constraint_name
                        AND ccu.table_schema = tc.table_schema
                )
            )
        ) AS foreign_keys
    FROM information_schema.table_constraints AS tc
    WHERE tc.constraint_type = 'FOREIGN KEY'
        AND tc.table_schema = $1
    GROUP BY tc.table_schema, tc.table_name
),
junction_tables AS (
    SELECT DISTINCT
        fk.table_schema,
        fk.table_name
    FROM foreign_keys fk
    CROSS JOIN LATERAL json_array_elements(fk.foreign_keys) AS fk_elem
    GROUP BY fk.table_schema, fk.table_name
    HAVING COUNT(DISTINCT fk_elem->>'referenced_table') = 2
        AND COUNT(*) = 2
)
SELECT
    t.table_schema,
    t.table_name,
    COALESCE(ci.columns, '[]'::json) AS columns,
    COALESCE(pk.pk_columns, ARRAY[]::text[]) AS pk_columns,
    COALESCE(uc.unique_constraints, '[]'::json) AS unique_constraints,
    COALESCE(fk.foreign_keys, '[]'::json) AS foreign_keys,
    CASE WHEN jt.table_name IS NOT NULL THEN true ELSE false END AS is_junction
FROM information_schema.tables t
LEFT JOIN columns_info ci ON t.table_name = ci.table_name AND t.table_schema = ci.table_schema
LEFT JOIN primary_keys pk ON t.table_name = pk.table_name AND t.table_schema = pk.table_schema
LEFT JOIN unique_constraints uc ON t.table_name = uc.table_name AND t.table_schema = uc.table_schema
LEFT JOIN foreign_keys fk ON t.table_name = fk.table_name AND t.table_schema = fk.table_schema
LEFT JOIN junction_tables jt ON t.table_name = jt.table_name AND t.table_schema = jt.table_schema
WHERE t.table_schema = $1
    AND t.table_type = 'BASE TABLE'
    %s
ORDER BY t.table_name;

