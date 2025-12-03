-- 0004_external_results.sql
-- External search cache table

CREATE TABLE IF NOT EXISTS external_results (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  query      TEXT NOT NULL,
  language   TEXT NOT NULL,
  title      TEXT NOT NULL,
  url        TEXT NOT NULL,
  snippet    TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_external_query_lang
  ON external_results (query, language);
