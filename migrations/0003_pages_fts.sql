-- 0003_pages_fts.sql
-- PostgreSQL full-text search support for pages

-- 1) Add tsvector column for FTS
ALTER TABLE pages
    ADD COLUMN IF NOT EXISTS content_tsv tsvector;

-- 2) Populate the tsvector column initially
--    (intentional full-table UPDATE to backfill existing rows)
UPDATE pages
SET content_tsv = to_tsvector(
    'simple',
    coalesce(title, '') || ' ' || coalesce(content, '')
);

-- 3) Create a GIN index for fast FTS
CREATE INDEX IF NOT EXISTS idx_pages_content_tsv
ON pages
USING GIN (content_tsv);

-- 4) Automatically update tsvector on INSERT/UPDATE
CREATE OR REPLACE FUNCTION pages_tsv_trigger()
RETURNS trigger AS $$
BEGIN
  NEW.content_tsv :=
    to_tsvector(
      'simple',
      coalesce(NEW.title, '') || ' ' || coalesce(NEW.content, '')
    );
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

-- 5) Trigger to keep content_tsv in sync
CREATE TRIGGER IF NOT EXISTS pages_tsvector_update
BEFORE INSERT OR UPDATE ON pages
FOR EACH ROW EXECUTE FUNCTION pages_tsv_trigger();
