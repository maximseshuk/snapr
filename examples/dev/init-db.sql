-- Example database initialization script
-- This script creates sample tables and data for testing backups

-- Create a sample users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create a sample posts table
CREATE TABLE IF NOT EXISTS posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    title VARCHAR(200) NOT NULL,
    content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create a temporary table (will be excluded in backup config)
CREATE TABLE IF NOT EXISTS temp_table (
    id SERIAL PRIMARY KEY,
    temp_data TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample data
INSERT INTO users (username, email) VALUES
    ('john_doe', 'john@example.com'),
    ('jane_smith', 'jane@example.com'),
    ('bob_wilson', 'bob@example.com')
ON CONFLICT (username) DO NOTHING;

INSERT INTO posts (user_id, title, content) VALUES
    (1, 'First Post', 'This is my first post!'),
    (1, 'Second Post', 'Another great post.'),
    (2, 'Hello World', 'Welcome to my blog.'),
    (3, 'Test Post', 'Just testing the backup system.')
ON CONFLICT DO NOTHING;

-- Insert temporary data
INSERT INTO temp_table (temp_data) VALUES
    ('This data should be excluded from backups');
