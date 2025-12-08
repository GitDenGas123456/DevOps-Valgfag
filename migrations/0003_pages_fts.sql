-- 0003_pages_fts.sql
-- PostgreSQL full-text search support for pages

-- Add tsvector column for FTS
ALTER TABLE pages
    ADD COLUMN IF NOT EXISTS content_tsv tsvector;

-- Populate the tsvector column initially
UPDATE pages
SET content_tsv = to_tsvector('simple', coalesce(title, '') || ' ' || coalesce(content, ''));

-- Create a GIN index for fast FTS
CREATE INDEX IF NOT EXISTS idx_pages_content_tsv
ON pages
USING GIN (content_tsv);

-- Automatically update tsvector on INSERT/UPDATE
CREATE OR REPLACE FUNCTION pages_tsv_trigger()
RETURNS trigger AS $$
BEGIN
  NEW.content_tsv =
    to_tsvector('simple', coalesce(NEW.title, '') || ' ' || coalesce(NEW.content, ''));
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE TRIGGER pages_tsvector_update
BEFORE INSERT OR UPDATE ON pages
FOR EACH ROW EXECUTE FUNCTION pages_tsv_trigger();
