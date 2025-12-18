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
// Each migration is executed in a transaction and recorded in schema_migrations.
func RunMigrations(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Run everything on a single connection to ensure advisory lock is held consistently.
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to get DB connection for migrations: %w", err)
	}
	defer func() { _ = conn.Close() }()

	locked := false
	defer func() {
		if locked {
			_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", migrationLockID)
		}
	}()

	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", migrationLockID); err != nil {
		return fmt.Errorf("failed to acquire migration advisory lock: %w", err)
	}
	locked = true

	if err := ensureSchemaMigrationsTable(ctx, conn); err != nil {
		return err
	}

	files, err := loadMigrationFiles("migrations")
	if err != nil {
		return err
	}

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

// migrationVersionFromFile extracts the migration version from the filename.
func migrationVersionFromFile(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// migrationApplied checks whether a given migration version is already recorded.
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

	statements := splitSQLStatements(string(content))
	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %s failed: %w", version, err)
		}
	}

	if _, err := tx.ExecContext(
		ctx,
		"INSERT INTO schema_migrations (version) VALUES ($1)",
		version,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to insert migration record %s: %w", version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction for migration %s: %w", version, err)
	}

	return nil
}

type splitState struct {
	inSingle  bool
	inDollar  bool
	dollarTag string
}

type dollarParse struct {
	ok     bool
	tagEnd int
	tag    string
}

// splitSQLStatements breaks a migration file into individual statements.
// It keeps semicolons inside single quotes and dollar-quoted blocks intact.
func splitSQLStatements(content string) []string {
	var (
		statements []string
		buf        strings.Builder
		st         splitState
	)

	for i := 0; i < len(content); i++ {
		ch := content[i]

		if tryHandleDollarStartOrEnd(content, &i, ch, &st, &buf) {
			continue
		}

		if tryHandleSingleQuote(content, &i, ch, &st, &buf) {
			continue
		}

		if tryHandleStatementBoundary(ch, &st, &buf, &statements) {
			continue
		}

		buf.WriteByte(ch)
	}

	flushStatement(&buf, &statements)
	return statements
}

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

func writeRange(buf *strings.Builder, content string, from, to int) {
	for j := from; j <= to; j++ {
		buf.WriteByte(content[j])
	}
}

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

func flushStatement(buf *strings.Builder, statements *[]string) {
	if tail := strings.TrimSpace(buf.String()); tail != "" {
		*statements = append(*statements, tail)
	}
}

func isDollarTagChar(b byte) bool {
	return b == '_' ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9')
}
