-- SQL fixtures for JSON integration tests
-- This file contains comprehensive test data for testing JSON filtering

-- Create test schema
CREATE SCHEMA IF NOT EXISTS app;

-- Products table for JSON filtering tests
CREATE TABLE IF NOT EXISTS app.products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    attributes JSONB,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert comprehensive test data
INSERT INTO app.products (name, attributes, metadata) VALUES
-- Product with all attributes
('Red Widget Pro',
 '{"color": "red", "size": 10, "price": 99.99, "tags": ["sale", "featured"], "details": {"manufacturer": "Acme", "model": "Pro", "warranty": {"years": 2, "provider": "Acme"}}, "specs": {"weight": 1.5, "dimensions": {"width": 10.0, "height": 5.0, "depth": 3.0}}}'::jsonb,
 '{"category": "electronics", "sku": "RWP-001", "stock": 50}'::jsonb),

-- Product with different attributes
('Blue Gadget Basic',
 '{"color": "blue", "size": 20, "price": 149.99, "tags": ["new"], "details": {"manufacturer": "TechCorp", "model": "Basic", "warranty": {"years": 1, "provider": "TechCorp"}}, "specs": {"weight": 2.5, "dimensions": {"width": 15.0, "height": 8.0, "depth": 4.0}}}'::jsonb,
 '{"category": "gadgets", "sku": "BGB-002", "stock": 30}'::jsonb),

-- Product with Acme manufacturer
('Green Tool Deluxe',
 '{"color": "green", "size": 5, "price": 29.99, "tags": [], "details": {"manufacturer": "Acme", "model": "Deluxe", "warranty": {"years": 3, "provider": "Extended"}}, "specs": {"weight": 0.5, "dimensions": {"width": 5.0, "height": 3.0, "depth": 2.0}}}'::jsonb,
 '{"category": "tools", "sku": "GTD-003", "stock": 100}'::jsonb),

-- Product with null attributes
('Basic Item',
 NULL,
 '{"category": "misc", "sku": "BI-004", "stock": 10}'::jsonb),

-- Product with empty JSON object
('Empty Attributes Product',
 '{}'::jsonb,
 '{}'::jsonb),

-- Product for testing special characters
('Special Chars Product',
 '{"name": "O''Brien''s Item", "description": "Line1\nLine2", "path": "C:\\Users\\test", "unicode": "Hello 世界", "emoji": "Test 🚀 emoji"}'::jsonb,
 '{"notes": "Contains \"quotes\" and ''apostrophes''"}'::jsonb),

-- Product with edge case values
('Edge Case Product',
 '{"zero": 0, "negative": -10, "large": 999999, "decimal": 123.456789, "boolean_true": true, "boolean_false": false, "null_value": null, "empty_string": "", "empty_array": [], "empty_object": {}}'::jsonb,
 NULL),

-- Products for testing arrays
('Array Test 1',
 '{"items": [{"name": "widget", "qty": 5, "price": 10.0}, {"name": "gadget", "qty": 2, "price": 20.0}]}'::jsonb,
 '{"order_id": "ORD-001"}'::jsonb),

('Array Test 2',
 '{"items": [{"name": "tool", "qty": 1, "price": 15.0}, {"name": "widget", "qty": 3, "price": 10.0}]}'::jsonb,
 '{"order_id": "ORD-002"}'::jsonb),

('Array Test 3',
 '{"items": [{"name": "gadget", "qty": 10, "price": 20.0}]}'::jsonb,
 '{"order_id": "ORD-003"}'::jsonb),

-- Products for testing deep nesting (5 levels)
('Deep Nesting Test',
 '{"level1": {"level2": {"level3": {"level4": {"level5": "deep_value", "number": 42}}}}}'::jsonb,
 NULL),

-- Product with multiple array indices test data
('Matrix Test',
 '{"matrix": [[{"value": "00"}, {"value": "01"}], [{"value": "10"}, {"value": "11"}]], "grid": {"rows": [{"cols": [{"val": 1}, {"val": 2}]}, {"cols": [{"val": 3}, {"val": 4}]}]}}'::jsonb,
 NULL),

-- Products for testing ranges
('Small Size', '{"size": 1}'::jsonb, NULL),
('Medium Size', '{"size": 15}'::jsonb, NULL),
('Large Size', '{"size": 100}'::jsonb, NULL);

-- Additional table for MapComparator tests
CREATE TABLE IF NOT EXISTS app.configurations (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    settings JSONB
);

INSERT INTO app.configurations (name, settings) VALUES
('Production Config', '{"timeout": 30, "enabled": true, "mode": "production", "max_connections": 100}'::jsonb),
('Staging Config', '{"timeout": 60, "enabled": false, "mode": "staging", "max_connections": 50}'::jsonb),
('Dev Config', '{"timeout": 10, "enabled": true, "mode": "development", "debug": true}'::jsonb),
('Null Config', NULL),
('Empty Config', '{}'::jsonb);

-- Table for testing complex queries
CREATE TABLE IF NOT EXISTS app.orders (
    id SERIAL PRIMARY KEY,
    customer_name TEXT NOT NULL,
    order_data JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO app.orders (customer_name, order_data) VALUES
('Alice Johnson', '{"total": 150.50, "status": "shipped", "items": [{"id": 1, "name": "Widget", "qty": 3, "price": 25.00}, {"id": 2, "name": "Gadget", "qty": 2, "price": 37.75}], "shipping": {"address": {"street": "123 Main St", "city": "Springfield", "state": "IL", "zip": "62701"}, "method": "express"}}'::jsonb),
('Bob Smith', '{"total": 89.99, "status": "processing", "items": [{"id": 3, "name": "Tool", "qty": 1, "price": 89.99}], "shipping": {"address": {"street": "456 Oak Ave", "city": "Portland", "state": "OR", "zip": "97201"}, "method": "standard"}}'::jsonb),
('Charlie Brown', '{"total": 250.00, "status": "delivered", "items": [{"id": 1, "name": "Widget", "qty": 10, "price": 25.00}], "shipping": {"address": {"street": "789 Elm Dr", "city": "Austin", "state": "TX", "zip": "78701"}, "method": "express"}}'::jsonb),
('Diana Prince', NULL);

-- Create indexes for performance testing
CREATE INDEX IF NOT EXISTS idx_products_attributes ON app.products USING GIN (attributes);
CREATE INDEX IF NOT EXISTS idx_products_metadata ON app.products USING GIN (metadata);
CREATE INDEX IF NOT EXISTS idx_configurations_settings ON app.configurations USING GIN (settings);
CREATE INDEX IF NOT EXISTS idx_orders_data ON app.orders USING GIN (order_data);

-- Add some comments for documentation
COMMENT ON TABLE app.products IS 'Test table for JSON filtering with typed attributes';
COMMENT ON TABLE app.configurations IS 'Test table for MapComparator filtering';
COMMENT ON TABLE app.orders IS 'Test table for complex nested JSON queries';
