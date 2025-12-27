-- Test data for JSON field selection tests
-- This file contains example INSERT statements for testing JSON field extraction
-- 
-- Usage: These can be used in integration tests or as reference for expected JSON structure
--
-- Example product data with nested JSON attributes:

INSERT INTO app.products (id, name, attributes, metadata) VALUES
-- Simple case: basic attributes
(1, 'Product 1', 
 '{"color": "red", "size": 10}'::jsonb,
 '{"type": "premium", "price": 100}'::jsonb),

-- Nested object: with details
(2, 'Product 2',
 '{"color": "blue", "size": 20, "details": {"manufacturer": "Acme Corp", "model": "X-2000"}}'::jsonb,
 '{"type": "standard", "price": 50}'::jsonb),

-- Deep nesting: with warranty info
(3, 'Product 3',
 '{"color": "green", "size": 15, "details": {"manufacturer": "Tech Inc", "model": "Y-3000", "warranty": {"years": 3, "provider": "TechCare"}}}'::jsonb,
 '{"type": "premium", "price": 200}'::jsonb),

-- Three-level nesting: with specs and dimensions
(4, 'Product 4',
 '{"color": "black", "size": 25, "specs": {"weight": 2.5, "dimensions": {"width": 10.0, "height": 20.0, "depth": 5.0}}}'::jsonb,
 '{"type": "premium", "price": 300}'::jsonb),

-- Complex: all fields including nested objects
(5, 'Product 5',
 '{"color": "white", "size": 30, "tags": ["new", "featured"], "details": {"manufacturer": "MegaCorp", "model": "Z-4000", "warranty": {"years": 5, "provider": "MegaCare"}}, "specs": {"weight": 3.0, "dimensions": {"width": 15.0, "height": 25.0, "depth": 8.0}}}'::jsonb,
 '{"type": "premium", "price": 500, "featured": true}'::jsonb);

-- Expected query results for reference:
--
-- Query: { products { name, attributes { color, size } } }
-- Product 1: { name: "Product 1", attributes: { color: "red", size: 10 } }
-- Product 2: { name: "Product 2", attributes: { color: "blue", size: 20 } }
--
-- Query: { products { name, attributes { color, details { manufacturer, model } } } }
-- Product 2: { name: "Product 2", attributes: { color: "blue", details: { manufacturer: "Acme Corp", model: "X-2000" } } }
--
-- Query: { products { name, attributes { details { warranty { years, provider } } } } }
-- Product 3: { name: "Product 3", attributes: { details: { warranty: { years: 3, provider: "TechCare" } } } }
--
-- Query: { products { name, attributes { specs { dimensions { width, height, depth } } } } }
-- Product 4: { name: "Product 4", attributes: { specs: { dimensions: { width: 10.0, height: 20.0, depth: 5.0 } } } }

