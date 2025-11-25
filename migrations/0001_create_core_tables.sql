-- Baseline schema matching handlers and current usage.

CREATE TABLE IF NOT EXISTS users (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  username  TEXT    NOT NULL UNIQUE,
  email     TEXT    NOT NULL UNIQUE,
  password  TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS pages (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  title        TEXT    NOT NULL UNIQUE,
  url          TEXT    NOT NULL UNIQUE,
  language     TEXT    NOT NULL CHECK(language IN ('en','da')) DEFAULT 'en',
  last_updated TIMESTAMP,
  content      TEXT    NOT NULL
);
