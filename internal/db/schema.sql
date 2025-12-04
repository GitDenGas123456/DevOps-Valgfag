-- ===============================
-- Drop and recreate users table
-- ===============================
DROP TABLE IF EXISTS users;

CREATE TABLE IF NOT EXISTS users (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  username  TEXT NOT NULL UNIQUE,
  email     TEXT NOT NULL UNIQUE,
  password  TEXT NOT NULL
);

-- ===============================
-- Drop and recreate pages table
-- ===============================
DROP TABLE IF EXISTS pages;

CREATE TABLE IF NOT EXISTS pages (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  title        TEXT UNIQUE,
  url          TEXT UNIQUE,
  language     TEXT NOT NULL CHECK(language IN ('en', 'da')) DEFAULT 'en',
  last_updated TIMESTAMP,
  content      TEXT NOT NULL
);

-- Sample content
INSERT INTO pages (title, url, language, last_updated, content)
VALUES
  ('Welcome', '/welcome', 'en', CURRENT_TIMESTAMP,
   'Welcome to WhoKnows, the best search engine!'),
  ('About Us', '/about', 'en', CURRENT_TIMESTAMP,
   'We intend to build the world’s best search engine.');

-- ===============================
-- Drop and recreate external_results table (Wikipedia / external cache)
-- ===============================
DROP TABLE IF EXISTS external_results;

CREATE TABLE IF NOT EXISTS external_results (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  query      TEXT NOT NULL,
  language   TEXT NOT NULL,
  title      TEXT NOT NULL,
  url        TEXT NOT NULL,
  snippet    TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(query, language, url)                  
);

CREATE INDEX IF NOT EXISTS idx_external_query_lang
  ON external_results (query, language);
