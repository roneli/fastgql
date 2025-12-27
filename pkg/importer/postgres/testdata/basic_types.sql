-- Basic types test schema
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    age INTEGER,
    active BOOLEAN DEFAULT true,
    balance DECIMAL(10, 2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO users (name, email, age, active, balance) VALUES
    ('Alice', 'alice@example.com', 30, true, 1000.50),
    ('Bob', 'bob@example.com', 25, false, 500.00),
    ('Charlie', 'charlie@example.com', 35, true, 2000.00);

