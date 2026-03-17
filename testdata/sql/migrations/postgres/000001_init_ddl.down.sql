-- INDEXES
DROP INDEX IF EXISTS author_email_idx;
DROP INDEX IF EXISTS author_phone_idx;
DROP INDEX IF EXISTS author_created_at_idx;
DROP INDEX IF EXISTS books_author_id_idx;
DROP INDEX IF EXISTS books_created_at_idx;

-- TABLES
DROP TABLE IF EXISTS books;
DROP TABLE IF EXISTS authors;

-- TYPES
DROP TYPE IF EXISTS books_type;
