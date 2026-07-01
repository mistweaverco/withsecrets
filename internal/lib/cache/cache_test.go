package cache

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestInitSchemaUsesConfigEnvColumn(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "db.sqlite")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	cache := &Cache{db: db}
	if err := cache.initSchema(); err != nil {
		t.Fatalf("initSchema: %v", err)
	}

	if !tableHasColumn(t, db, "secrets", "config_env") {
		t.Fatalf("expected secrets.config_env column")
	}
	if tableHasColumn(t, db, "secrets", "kuba_env") {
		t.Fatalf("did not expect secrets.kuba_env column")
	}
}

func TestDropLegacySchemaRecreatesWithoutMigratingRows(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "db.sqlite")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	legacySchema := `
	CREATE TABLE secrets (
		path TEXT NOT NULL,
		kuba_env TEXT NOT NULL,
		env TEXT NOT NULL,
		value TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		PRIMARY KEY (path, kuba_env, env)
	);`
	if _, err := db.Exec(legacySchema); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO secrets (path, kuba_env, env, value, expires_at) VALUES (?, ?, ?, ?, ?)`,
		"/tmp/ws.yaml", "default", "FOO", "secret", time.Now().Add(time.Hour),
	); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	cache := &Cache{db: db}
	if err := cache.initSchema(); err != nil {
		t.Fatalf("initSchema: %v", err)
	}

	if !tableHasColumn(t, db, "secrets", "config_env") {
		t.Fatalf("expected recreated secrets.config_env column")
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM secrets`).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected legacy cache rows to be discarded, got %d", count)
	}
}

func tableHasColumn(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		t.Fatalf("pragma table_info: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid       int
			name      string
			colType   string
			notNull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			t.Fatalf("scan table_info: %v", err)
		}
		if name == column {
			return true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table_info: %v", err)
	}
	return false
}

func TestGetCacheDirPrefersWithsecretsPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))

	got, err := getCacheDir()
	if err != nil {
		t.Fatalf("getCacheDir: %v", err)
	}
	want := filepath.Join(home, ".cache", "withsecrets")
	if got != want {
		t.Fatalf("getCacheDir() = %q, want %q", got, want)
	}
	if err := os.MkdirAll(got, 0755); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
}
