CREATE TABLE IF NOT EXISTS external_results (
  id         BIGSERIAL PRIMARY KEY,
  query      TEXT NOT NULL,
  language   VARCHAR(16) NOT NULL,
  title      TEXT NOT NULL,
  url        TEXT NOT NULL,
  snippet    TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT external_results_unique_result UNIQUE (query, language, url)
);

CREATE INDEX IF NOT EXISTS idx_external_query_lang
  ON external_results (query, language);
