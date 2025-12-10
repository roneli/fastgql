-- Database initialization script for JSON field selection example
-- Run this script to set up the database schema and test data

-- Create schema
CREATE SCHEMA IF NOT EXISTS app;

-- Create products table with JSONB columns
CREATE TABLE IF NOT EXISTS app.products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    attributes JSONB,
    metadata JSONB
);

-- Create indexes for JSONB columns (optional but recommended for performance)
CREATE INDEX IF NOT EXISTS idx_products_attributes ON app.products USING GIN (attributes);
CREATE INDEX IF NOT EXISTS idx_products_metadata ON app.products USING GIN (metadata);

-- Insert test data covering all complex JSON field selection scenarios

-- Simple case: basic attributes
INSERT INTO app.products (id, name, attributes, metadata) VALUES
(1, 'Product 1', 
 '{"color": "red", "size": 10}'::jsonb,
 '{"type": "premium", "price": 100}'::jsonb);

-- Nested object: with details
INSERT INTO app.products (id, name, attributes, metadata) VALUES
(2, 'Product 2',
 '{"color": "blue", "size": 20, "details": {"manufacturer": "Acme Corp", "model": "X-2000"}}'::jsonb,
 '{"type": "standard", "price": 50}'::jsonb);

-- Deep nesting: with warranty info
INSERT INTO app.products (id, name, attributes, metadata) VALUES
(3, 'Product 3',
 '{"color": "green", "size": 15, "details": {"manufacturer": "Tech Inc", "model": "Y-3000", "warranty": {"years": 3, "provider": "TechCare"}}}'::jsonb,
 '{"type": "premium", "price": 200}'::jsonb);

-- Three-level nesting: with specs and dimensions
INSERT INTO app.products (id, name, attributes, metadata) VALUES
(4, 'Product 4',
 '{"color": "black", "size": 25, "specs": {"weight": 2.5, "dimensions": {"width": 10.0, "height": 20.0, "depth": 5.0}}}'::jsonb,
 '{"type": "premium", "price": 300}'::jsonb);

-- Complex: all fields including nested objects
INSERT INTO app.products (id, name, attributes, metadata) VALUES
(5, 'Product 5',
 '{"color": "white", "size": 30, "tags": ["new", "featured"], "details": {"manufacturer": "MegaCorp", "model": "Z-4000", "warranty": {"years": 5, "provider": "MegaCare"}}, "specs": {"weight": 3.0, "dimensions": {"width": 15.0, "height": 25.0, "depth": 8.0}}}'::jsonb,
 '{"type": "premium", "price": 500, "featured": true}'::jsonb);

-- Example queries you can test in GraphQL playground:
--
-- 1. Simple scalar selection:
--    query { products { name, attributes { color, size } } }
--
-- 2. Nested object selection:
--    query { products { name, attributes { color, details { manufacturer, model } } } }
--
-- 3. Deep nesting (3 levels):
--    query { products { name, attributes { details { warranty { years, provider } } } } }
--
-- 4. Three-level nesting with dimensions:
--    query { products { name, attributes { specs { dimensions { width, height, depth } } } } }
--
-- 5. Mixed scalar and nested:
--    query { products { name, attributes { color, size, details { manufacturer }, specs { weight } } } }
--
-- 6. Complex nested object with all fields:
--    query { products { name, attributes { details { manufacturer, model, warranty { years, provider } } } } }

