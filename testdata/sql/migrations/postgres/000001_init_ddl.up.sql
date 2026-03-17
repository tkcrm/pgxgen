CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Authors
CREATE TABLE "authors" (
    id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(20) NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    notifications JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS author_email_idx ON authors("email") WHERE email != '';
CREATE UNIQUE INDEX IF NOT EXISTS author_phone_idx ON authors("phone") WHERE phone != '';
CREATE INDEX IF NOT EXISTS author_created_at_idx ON authors("created_at");

-- Books
DROP TYPE IF EXISTS book_type;
CREATE TYPE book_type AS ENUM (
    'adventure',
    'novel',
    'detective'
);
CREATE TABLE "books" (
    id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    "name" VARCHAR(255) NOT NULL,
    "description" TEXT NOT NULL,
    genre book_type NOT NULL,
    release_date TIMESTAMP NOT NULL,
    author_id UUID NOT NULL REFERENCES authors (id),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL
);
CREATE INDEX IF NOT EXISTS books_author_id_idx ON books("author_id");
CREATE INDEX IF NOT EXISTS books_created_at_idx ON books("created_at");
