-- Mixed schema test with all features
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    metadata JSONB,
    tags TEXT[]
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    title VARCHAR(255) NOT NULL,
    content TEXT,
    attributes JSON
);

CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE
);

CREATE TABLE posts_categories (
    post_id INTEGER NOT NULL REFERENCES posts(id),
    category_id INTEGER NOT NULL REFERENCES categories(id),
    PRIMARY KEY (post_id, category_id)
);

CREATE TABLE profiles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER UNIQUE NOT NULL REFERENCES users(id),
    bio TEXT
);

INSERT INTO users (name, email, metadata, tags) VALUES
    ('Alice', 'alice@example.com', '{"role": "admin"}'::jsonb, ARRAY['developer', 'designer']),
    ('Bob', 'bob@example.com', '{"role": "user"}'::jsonb, ARRAY['writer']);

INSERT INTO posts (user_id, title, content, attributes) VALUES
    (1, 'Post 1', 'Content 1', '{"published": true}'::json),
    (1, 'Post 2', 'Content 2', '{"published": false}'::json),
    (2, 'Post 3', 'Content 3', '{"published": true}'::json);

INSERT INTO categories (name) VALUES
    ('Tech'),
    ('Science'),
    ('Arts');

INSERT INTO posts_categories (post_id, category_id) VALUES
    (1, 1),
    (1, 2),
    (2, 2),
    (3, 3);

INSERT INTO profiles (user_id, bio) VALUES
    (1, 'Software developer'),
    (2, 'Content writer');

