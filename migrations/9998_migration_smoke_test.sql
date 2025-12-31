-- Migration smoke test
-- This migration exists only to verify that the migration system works end-to-end.

CREATE TABLE IF NOT EXISTS migration_smoke_test (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    note TEXT
);

INSERT INTO migration_smoke_test (note)
VALUES ('migration runner works');

-- Ensure semicolon inside string does not break parser
INSERT INTO migration_smoke_test (note)
VALUES ('this ; semicolon should not split the statement')