CREATE TABLE users (
    id UUID PRIMARY KEY NOT NULL,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL, -- Store the hashed password
    created_at TIMESTAMP DEFAULT current_timestamp
);
