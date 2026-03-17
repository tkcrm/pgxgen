CREATE TABLE users (
    id INTEGER PRIMARY KEY NOT NULL,
    username TEXT NOT NULL,
    email TEXT NOT NULL,
    age INTEGER NULL,
    is_active BOOLEAN NOT NULL DEFAULT 0,
    balance REAL NOT NULL DEFAULT 0.0,
    metadata TEXT NULL,
    avatar BLOB NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NULL
);

CREATE TABLE posts (
    id INTEGER PRIMARY KEY NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    author_id INTEGER NOT NULL REFERENCES users(id),
    published_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE "quoted_table" (
    "quoted_id" INTEGER PRIMARY KEY NOT NULL,
    "quoted_name" TEXT NOT NULL
);

-- Table to be dropped
CREATE TABLE temp_data (
    id INTEGER PRIMARY KEY NOT NULL,
    data TEXT
);
DROP TABLE temp_data;

-- IF NOT EXISTS
CREATE TABLE IF NOT EXISTS users (
    should_not_override TEXT
);
