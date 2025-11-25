-- Track which migrations are applied
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY
);
