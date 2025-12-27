-- JSON types test schema
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    attributes JSONB,
    metadata JSON,
    tags TEXT[]
);

INSERT INTO products (name, attributes, metadata, tags) VALUES
    ('Product 1', '{"color": "red", "size": "large"}'::jsonb, '{"source": "warehouse"}'::json, ARRAY['electronics', 'gadgets']),
    ('Product 2', '{"color": "blue", "size": "medium"}'::jsonb, '{"source": "store"}'::json, ARRAY['clothing', 'accessories']);

