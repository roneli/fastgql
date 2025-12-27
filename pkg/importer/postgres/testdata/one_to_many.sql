-- ONE_TO_MANY relationship test schema
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    title VARCHAR(255) NOT NULL,
    content TEXT
);

INSERT INTO users (name) VALUES
    ('Alice'),
    ('Bob');

INSERT INTO posts (user_id, title, content) VALUES
    (1, 'First Post', 'Content of first post'),
    (1, 'Second Post', 'Content of second post'),
    (2, 'Third Post', 'Content of third post');

