#!/usr/bin/env bash
set -euo pipefail

DB_PATH="${1:-data/seed/whoknows.db}"
MIGRATIONS_DIR="${2:-migrations}"

if [[ ! -d "$MIGRATIONS_DIR" ]]; then
  echo "Migrations dir not found: $MIGRATIONS_DIR" >&2
  exit 2
fi
mkdir -p "$(dirname "$DB_PATH")"
# Create the DB file if it doesn't exist and fix truncate on subsequent runs
 [[ -f "$DB_PATH" ]] || touch "$DB_PATH"

# Ensure tracking table exists
sqlite3 "$DB_PATH" <<'SQL'
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY
);
SQL

# Apply migrations in lexicographic order if not yet applied
while IFS= read -r -d '' file; do
  base="$(basename "$file")"
  if ! sqlite3 "$DB_PATH" "SELECT 1 FROM schema_migrations WHERE version='$base' LIMIT 1;" | grep -q 1; then
    echo ">> Applying $base"
    sqlite3 "$DB_PATH" < "$file"
    sqlite3 "$DB_PATH" "INSERT INTO schema_migrations(version) VALUES('$base');"
  else
    echo "-- Skipping already applied: $base"
  fi
done < <(find "$MIGRATIONS_DIR" -maxdepth 1 -type f -name '*.sql' -print0 | sort -z)

sqlite3 "$DB_PATH" 'SELECT version FROM schema_migrations ORDER BY version;'