CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE SCHEMA IF NOT EXISTS custom_schema;

-- Enum type
CREATE TYPE status AS ENUM ('active', 'inactive', 'pending');

-- Users table with various column types
CREATE TABLE users (
    id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(255) NOT NULL,
    email TEXT NOT NULL,
    age INTEGER NULL,
    balance NUMERIC(10,2) NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    metadata JSONB NULL,
    avatar BYTEA NULL,
    tags TEXT[] NULL,
    matrix INTEGER[][] NULL,
    status status NOT NULL DEFAULT 'pending',
    login_count BIGINT NOT NULL DEFAULT 0,
    score REAL NULL,
    rating DOUBLE PRECISION NULL,
    small_val SMALLINT NOT NULL DEFAULT 0,
    auto_id SERIAL NOT NULL,
    big_auto_id BIGSERIAL NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL
);

COMMENT ON TABLE users IS 'Main users table';
COMMENT ON COLUMN users.email IS 'User email address';
COMMENT ON COLUMN users.metadata IS 'Arbitrary JSON metadata';

-- Posts table
CREATE TABLE posts (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    body TEXT NOT NULL,
    author_id UUID NOT NULL REFERENCES users(id),
    published_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Schema-qualified table
CREATE TABLE custom_schema.settings (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) NOT NULL,
    value TEXT NULL
);

-- Table to be altered
CREATE TABLE profiles (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL
);

ALTER TABLE profiles ADD COLUMN bio TEXT NULL;
ALTER TABLE profiles ADD COLUMN website VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE profiles DROP COLUMN website;
ALTER TABLE profiles ALTER COLUMN bio SET NOT NULL;
ALTER TABLE profiles ALTER COLUMN bio TYPE VARCHAR(1000);

-- Table to be dropped
CREATE TABLE temp_data (
    id SERIAL PRIMARY KEY,
    data TEXT
);
DROP TABLE temp_data;

-- Enum to be dropped and re-created
CREATE TYPE priority AS ENUM ('low', 'medium', 'high');
DROP TYPE priority;

-- IF NOT EXISTS test
CREATE TABLE IF NOT EXISTS users (
    should_not_override TEXT
);
