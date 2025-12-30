package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const migrationLockID int64 = 8675309

// RunMigrations applies all pending .sql migrations found in the migrations/ folder.
//
// Key properties:
//  1) Safe under concurrency: uses a PostgreSQL advisory lock so only one process runs migrations at a time.
//  2) Deterministic: applies migrations in sorted filename order (e.g., 0001_..., 0002_...).
//  3) Idempotent: records applied versions in schema_migrations so reruns skip already-applied files.
//  4) Atomic: each migration file runs inside a DB transaction (all-or-nothing).
func RunMigrations(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use a single dedicated connection so the advisory lock is held consistently for the whole run.
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to get DB connection for migrations: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// Track lock state so we always release it on exit (even on error/panic paths).
	locked := false
	defer func() {
		if locked {
			// Unlock uses Background so we attempt best-effort unlock even if ctx is cancelled/timed out.
			_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", migrationLockID)
		}
	}()

	// Advisory lock prevents concurrent migration runners (e.g., multiple app replicas starting together).
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", migrationLockID); err != nil {
		return fmt.Errorf("failed to acquire migration advisory lock: %w", err)
	}
	locked = true

	// Ensure the bookkeeping table exists before checking/recording migration versions.
	if err := ensureSchemaMigrationsTable(ctx, conn); err != nil {
		return err
	}

	// Load all migration files from disk (sorted for deterministic order).
	files, err := loadMigrationFiles("migrations")
	if err != nil {
		return err
	}

	// Apply each migration exactly once (skip if its version is already recorded).
	for _, file := range files {
		version := migrationVersionFromFile(file)

		applied, err := migrationApplied(ctx, conn, version)
		if err != nil {
			return err
		}
		if applied {
			fmt.Println("Skipping migration:", version)
			continue
		}

		if err := applyMigrationFile(ctx, conn, version, file); err != nil {
			return err
		}
	}

	fmt.Println("All migrations applied successfully.")
	return nil
}

// ensureSchemaMigrationsTable makes sure the schema_migrations table exists.
// schema_migrations is our "ledger" of which migration versions have been applied.
func ensureSchemaMigrationsTable(ctx context.Context, conn *sql.Conn) error {
	_, err := conn.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to ensure schema_migrations: %w", err)
	}
	return nil
}

// loadMigrationFiles returns a sorted list of .sql migration files in the given directory.
// Sorting ensures deterministic execution order across environments.
func loadMigrationFiles(dir string) ([]string, error) {
	pattern := filepath.Join(dir, "*.sql")

	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations folder: %w", err)
	}

	// Ensure deterministic order (e.g. 0001_..., 0002_...)
	sort.Strings(files)
	return files, nil
}

// migrationVersionFromFile extracts the migration "version" from the filename.
// Example: migrations/0003_pages_fts.sql -> 0003_pages_fts
// We store this version string in schema_migrations.
func migrationVersionFromFile(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// migrationApplied checks whether a given migration version is already recorded.
// If recorded, we skip it to keep migrations safe to rerun.
func migrationApplied(ctx context.Context, conn *sql.Conn, version string) (bool, error) {
	var count int
	err := conn.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM schema_migrations WHERE version = $1",
		version,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to query schema_migrations: %w", err)
	}
	return count > 0, nil
}

// applyMigrationFile runs a single migration file inside a transaction and records it.
//
// Flow:
//  1) Read file content.
//  2) Split into executable SQL statements (careful: don't split inside strings or $$ blocks).
//  3) Execute statements in a transaction (rollback on first error).
//  4) Record the version in schema_migrations and commit.
func applyMigrationFile(ctx context.Context, conn *sql.Conn, version, file string) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read migration %s: %w", file, err)
	}

	fmt.Println("Applying migration:", version)

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for migration %s: %w", version, err)
	}

	// Split file into statements safely.
	// We can't just strings.Split(..., ";") because migration files may contain:
	// - string literals: 'text; with semicolon'
	// - dollar-quoted blocks: $$ BEGIN ...; ... END $$ (functions/triggers)
	// Those semicolons must NOT terminate the statement.
	statements := splitSQLStatements(string(content))
	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %s failed: %w", version, err)
		}
	}

	// Record migration version only if all statements succeeded.
	if _, err := tx.ExecContext(
		ctx,
		"INSERT INTO schema_migrations (version) VALUES ($1)",
		version,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to insert migration record %s: %w", version, err)
	}

	// Commit makes the migration visible atomically (statements + schema_migrations row).
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction for migration %s: %w", version, err)
	}

	return nil
}

// splitState tracks whether the SQL parser is currently inside:
// - a single-quoted string: '...'
// - a dollar-quoted block: $$...$$ or $tag$...$tag$
// When inside either, semicolons should NOT end a statement.
type splitState struct {
	inSingle  bool
	inDollar  bool
	dollarTag string
}

// dollarParse represents a parsed dollar-quote delimiter.
// Example delimiters: $$  or  $mytag$
type dollarParse struct {
	ok     bool
	tagEnd int
	tag    string
}

// splitSQLStatements breaks a migration file into individual statements.
//
// Core rule: split on ';' only when we are NOT inside:
//  - single quotes ('...'), or
//  - dollar-quoted blocks ($$...$$ / $tag$...$tag$).
//
// This is a small state machine that walks the file byte-by-byte, buffering output,
// and "emits" a statement whenever it finds a safe semicolon boundary.
func splitSQLStatements(content string) []string {
	var (
		statements []string
		buf        strings.Builder
		st         splitState
	)

	for i := 0; i < len(content); i++ {
		ch := content[i]

		// Handle entering/leaving dollar-quoted blocks ($$ or $tag$).
		if tryHandleDollarStartOrEnd(content, &i, ch, &st, &buf) {
			continue
		}

		// Handle entering/leaving single-quoted strings (including escaped quotes '').
		if tryHandleSingleQuote(content, &i, ch, &st, &buf) {
			continue
		}

		// Split on semicolon only if we are not inside a quoted region.
		if tryHandleStatementBoundary(ch, &st, &buf, &statements) {
			continue
		}

		buf.WriteByte(ch)
	}

	// Flush any trailing statement at EOF (even if file doesn't end with ';').
	flushStatement(&buf, &statements)
	return statements
}

// tryHandleDollarStartOrEnd detects $$ or $tag$ delimiters and toggles dollar-quote state.
// - When not inDollar, a delimiter starts a dollar block.
// - When inDollar, only the matching delimiter ends the block.
func tryHandleDollarStartOrEnd(content string, i *int, ch byte, st *splitState, buf *strings.Builder) bool {
	if st.inSingle || ch != '$' {
		return false
	}

	dp := parseDollarTag(content, *i)
	if !dp.ok {
		return false
	}

	// Always write the delimiter ($$ or $tag$) to the buffer.
	writeRange(buf, content, *i, dp.tagEnd)

	if st.inDollar {
		// Only matching tag closes.
		if dp.tag == st.dollarTag {
			st.inDollar = false
			st.dollarTag = ""
		}
	} else {
		// Start dollar block.
		st.inDollar = true
		st.dollarTag = dp.tag
	}

	*i = dp.tagEnd
	return true
}

// parseDollarTag parses a dollar-quote delimiter starting at position start.
// Valid examples: $$ or $tag$ where tag is [A-Za-z0-9_]*.
func parseDollarTag(content string, start int) dollarParse {
	tagEnd := start + 1
	for tagEnd < len(content) && isDollarTagChar(content[tagEnd]) {
		tagEnd++
	}
	if tagEnd < len(content) && content[tagEnd] == '$' {
		return dollarParse{
			ok:     true,
			tagEnd: tagEnd,
			tag:    content[start+1 : tagEnd], // may be empty for $$...$$
		}
	}
	return dollarParse{ok: false}
}

// writeRange appends content[from:to] inclusive into the buffer.
func writeRange(buf *strings.Builder, content string, from, to int) {
	for j := from; j <= to; j++ {
		buf.WriteByte(content[j])
	}
}

// tryHandleSingleQuote updates state when encountering a single quote.
// Supports escaped quotes inside strings using SQL convention: ''.
func tryHandleSingleQuote(content string, i *int, ch byte, st *splitState, buf *strings.Builder) bool {
	if st.inDollar || ch != '\'' {
		return false
	}

	buf.WriteByte(ch)

	if st.inSingle {
		// Escaped single quote inside string: ''
		if *i+1 < len(content) && content[*i+1] == '\'' {
			buf.WriteByte(content[*i+1])
			*i++
			return true
		}
		st.inSingle = false
		return true
	}

	st.inSingle = true
	return true
}

// tryHandleStatementBoundary emits the buffered statement when we hit a safe ';' boundary.
// We only treat ';' as a boundary if we're not inside single quotes or dollar blocks.
func tryHandleStatementBoundary(ch byte, st *splitState, buf *strings.Builder, statements *[]string) bool {
	if ch != ';' || st.inSingle || st.inDollar {
		return false
	}

	stmt := strings.TrimSpace(buf.String())
	if stmt != "" {
		*statements = append(*statements, stmt)
	}
	buf.Reset()
	return true
}

// flushStatement appends any trailing buffered SQL as a final statement.
func flushStatement(buf *strings.Builder, statements *[]string) {
	if tail := strings.TrimSpace(buf.String()); tail != "" {
		*statements = append(*statements, tail)
	}
}

// isDollarTagChar returns true if b is allowed in a $tag$ delimiter.
func isDollarTagChar(b byte) bool {
	return b == '_' ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9')
}