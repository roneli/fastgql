-- MANY_TO_MANY relationship test schema
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT
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

INSERT INTO posts (title, content) VALUES
    ('Post 1', 'Content 1'),
    ('Post 2', 'Content 2'),
    ('Post 3', 'Content 3');

INSERT INTO categories (name) VALUES
    ('Technology'),
    ('Science'),
    ('Arts');

INSERT INTO posts_categories (post_id, category_id) VALUES
    (1, 1),
    (1, 2),
    (2, 2),
    (2, 3),
    (3, 1),
    (3, 3);

