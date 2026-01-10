-- ONE_TO_ONE relationship test schema
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL
);

CREATE TABLE profiles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER UNIQUE NOT NULL REFERENCES users(id),
    bio TEXT,
    avatar_url VARCHAR(255)
);

INSERT INTO users (name, email) VALUES
    ('Alice', 'alice@example.com'),
    ('Bob', 'bob@example.com');

INSERT INTO profiles (user_id, bio, avatar_url) VALUES
    (1, 'Software developer', 'https://example.com/alice.jpg'),
    (2, 'Designer', 'https://example.com/bob.jpg');

