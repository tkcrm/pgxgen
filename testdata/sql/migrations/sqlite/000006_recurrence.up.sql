ALTER TABLE todos ADD COLUMN recurrence_rule TEXT;
ALTER TABLE todos ADD COLUMN recurrence_parent_id TEXT REFERENCES todos(id);
