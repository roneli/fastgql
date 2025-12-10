-- create schema for example/interface/schema.graphql

CREATE TABLE animals (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  breed TEXT NOT NULL,
  color TEXT NOT NULL
);

CREATE TABLE "user" (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE post (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE category (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE posts_to_categories (
    post_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL,
    PRIMARY KEY (post_id, category_id)
);

-- initialize base data
INSERT INTO "user" (name) VALUES ('Alice');
INSERT INTO "user" (name) VALUES ('Bob');
INSERT INTO "user" (name) VALUES ('Charlie');
INSERT INTO "user" (name) VALUES ('David');
INSERT INTO "user" (name) VALUES ('Eve');

INSERT INTO category (name) VALUES ('News');
INSERT INTO category (name) VALUES ('Technology');
INSERT INTO category (name) VALUES ('Science');
INSERT INTO category (name) VALUES ('Sports');
INSERT INTO category (name) VALUES ('Entertainment');

INSERT INTO post (name, user_id) VALUES ('Hello World', 1);
INSERT INTO post (name, user_id) VALUES ('GraphQL is awesome', 2);
INSERT INTO post (name, user_id) VALUES ('Postgres is cool', 3);
INSERT INTO post (name, user_id) VALUES ('Deno is interesting', 4);
INSERT INTO post (name, user_id) VALUES ('Node.js is fast', 5);

-- some posts are in multiple categories
INSERT INTO posts_to_categories (post_id, category_id) VALUES (1, 1);
INSERT INTO posts_to_categories (post_id, category_id) VALUES (2, 2);
INSERT INTO posts_to_categories (post_id, category_id) VALUES (3, 3);
INSERT INTO posts_to_categories (post_id, category_id) VALUES (4, 4);
INSERT INTO posts_to_categories (post_id, category_id) VALUES (5, 5);
INSERT INTO posts_to_categories (post_id, category_id) VALUES (1, 2);
INSERT INTO posts_to_categories (post_id, category_id) VALUES (2, 3);
INSERT INTO posts_to_categories (post_id, category_id) VALUES (3, 4);
INSERT INTO posts_to_categories (post_id, category_id) VALUES (4, 5);
INSERT INTO posts_to_categories (post_id, category_id) VALUES (5, 1);


-- insert some animals
INSERT INTO animals (name, type, breed, color) VALUES ('Fido', 'dog', 'labrador', 'black');
INSERT INTO animals (name, type, breed, color) VALUES ('Whiskers', 'cat', 'siamese', 'white');
INSERT INTO animals (name, type, breed, color) VALUES ('Spot', 'dog', 'dalmatian', 'white');
INSERT INTO animals (name, type, breed, color) VALUES ('Fluffy', 'cat', 'persian', 'grey');
INSERT INTO animals (name, type, breed, color) VALUES ('Rover', 'dog', 'bulldog', 'brown');
INSERT INTO animals (name, type, breed, color) VALUES ('Mittens', 'cat', 'maine coon', 'black');

-- Products table with JSONB columns for testing JSON filtering
CREATE TABLE product (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    attributes JSONB,    -- Typed JSON for structured attributes
    metadata JSONB,      -- Dynamic Map for arbitrary data
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Insert products with various JSON structures
INSERT INTO product (name, attributes, metadata) VALUES 
    ('Widget', 
     '{"color": "red", "size": 10, "details": {"manufacturer": "Acme", "model": "Pro-100"}}',
     '{"tags": ["sale", "featured"], "price": 99.99, "discount": "true"}'),
    ('Gadget', 
     '{"color": "blue", "size": 20, "details": {"manufacturer": "TechCo", "model": "Ultra-200"}}',
     '{"tags": ["new"], "price": 149.99}'),
    ('Gizmo', 
     '{"color": "red", "size": 15, "details": {"manufacturer": "Acme", "model": "Basic-50"}}',
     '{"tags": ["sale"], "price": 49.99, "discount": "true"}'),
    ('Tool', 
     '{"color": "green", "size": 5, "details": {"manufacturer": "ToolCorp", "model": "Mini-10"}}',
     '{"tags": ["featured"], "price": 29.99}'),
    ('Device', 
     '{"color": "blue", "size": 25, "details": {"manufacturer": "TechCo", "model": "Pro-300"}}',
     '{"tags": ["new", "featured"], "price": 199.99, "rating": 4.5}');

