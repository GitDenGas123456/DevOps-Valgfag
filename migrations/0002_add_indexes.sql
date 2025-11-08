-- B-tree index to speed up filtering by language
CREATE INDEX IF NOT EXISTS idx_pages_language ON pages(language);
