-- 9998_parser_edgecases.sql
-- Parser edge cases smoke test
-- Only to validate splitSQLStatements() handles tricky Postgres syntax.

-- Ensure test table exists (so this file can run standalone).
CREATE TABLE IF NOT EXISTS migration_smoke_test (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    note TEXT
);

-- 1) Escaped single quote inside string (SQL uses '')
INSERT INTO migration_smoke_test (note)
VALUES ('it''s still one statement');

-- 2) Dollar-quoted block (semicolons inside must NOT split)
DO $$
BEGIN
  PERFORM 1;
  PERFORM 2;
END
$$;

-- 3) Tagged dollar-quoted block
DO $tag$
BEGIN
  PERFORM 3;
END
$tag$;

-- 4) Prove file finished after DO blocks
INSERT INTO migration_smoke_test (note)
VALUES ('dollar-quote blocks executed');