package cache

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func newTestCache(t *testing.T) *Cache {
	t.Helper()
	dir := t.TempDir()
	db, err := sql.Open("sqlite3", filepath.Join(dir, "db.sqlite"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	c := &Cache{db: db}
	if err := c.initSchema(); err != nil {
		t.Fatalf("initSchema: %v", err)
	}
	return c
}

func TestPathMappingRoundTrip(t *testing.T) {
	c := newTestCache(t)
	mapping := CachedPathMapping{
		Entries: []CachedPathEntry{
			{EnvironmentVariable: "DB_PASSWORD", SourceRef: "database-password", Value: "s3cret"},
			{EnvironmentVariable: "DB_USERNAME", SourceRef: "database-username", Value: "alice"},
		},
	}

	if err := c.SetPathMapping("/tmp/ws.yaml", "default", "id-db", mapping, time.Hour); err != nil {
		t.Fatalf("SetPathMapping: %v", err)
	}

	got, found, err := c.GetPathMapping("/tmp/ws.yaml", "default", "id-db")
	if err != nil {
		t.Fatalf("GetPathMapping: %v", err)
	}
	if !found {
		t.Fatal("expected path mapping cache hit")
	}
	if got.Version != PathMappingVersion {
		t.Fatalf("version = %d, want %d", got.Version, PathMappingVersion)
	}
	if len(got.Entries) != 2 {
		t.Fatalf("entries = %d, want 2: %#v", len(got.Entries), got.Entries)
	}
	byEnv := map[string]CachedPathEntry{}
	for _, e := range got.Entries {
		byEnv[e.EnvironmentVariable] = e
	}
	if byEnv["DB_USERNAME"].Value != "alice" || byEnv["DB_USERNAME"].SourceRef != "database-username" {
		t.Fatalf("unexpected DB_USERNAME: %#v", byEnv["DB_USERNAME"])
	}
	if byEnv["DB_PASSWORD"].Value != "s3cret" || byEnv["DB_PASSWORD"].SourceRef != "database-password" {
		t.Fatalf("unexpected DB_PASSWORD: %#v", byEnv["DB_PASSWORD"])
	}
}

func TestPathMappingEmptyEntriesRoundTrip(t *testing.T) {
	c := newTestCache(t)
	if err := c.SetPathMapping("/tmp/ws.yaml", "default", "empty", CachedPathMapping{}, time.Hour); err != nil {
		t.Fatalf("SetPathMapping: %v", err)
	}
	got, found, err := c.GetPathMapping("/tmp/ws.yaml", "default", "empty")
	if err != nil {
		t.Fatalf("GetPathMapping: %v", err)
	}
	if !found {
		t.Fatal("expected empty path mapping to be a cache hit")
	}
	if len(got.Entries) != 0 {
		t.Fatalf("entries = %#v, want empty", got.Entries)
	}
}

func TestPathMappingExpiredIsMiss(t *testing.T) {
	c := newTestCache(t)
	mapping := CachedPathMapping{
		Entries: []CachedPathEntry{{EnvironmentVariable: "USERNAME", SourceRef: "USERNAME", Value: "alice"}},
	}
	if err := c.SetPathMapping("/tmp/ws.yaml", "default", "id", mapping, -time.Hour); err != nil {
		t.Fatalf("SetPathMapping: %v", err)
	}
	_, found, err := c.GetPathMapping("/tmp/ws.yaml", "default", "id")
	if err != nil {
		t.Fatalf("GetPathMapping: %v", err)
	}
	if found {
		t.Fatal("expected expired path mapping to be a miss")
	}
}

func TestPathMappingMalformedIsMiss(t *testing.T) {
	c := newTestCache(t)
	if err := c.Set("/tmp/ws.yaml", "default", PathMappingCacheKey("id"), "not-json", time.Hour); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, found, err := c.GetPathMapping("/tmp/ws.yaml", "default", "id")
	if err != nil {
		t.Fatalf("GetPathMapping: %v", err)
	}
	if found {
		t.Fatalf("expected malformed record to be a miss, got %#v", got)
	}
}

func TestPathMappingVersionMismatchIsMiss(t *testing.T) {
	c := newTestCache(t)
	if err := c.Set("/tmp/ws.yaml", "default", PathMappingCacheKey("id"), `{"version":99,"entries":[]}`, time.Hour); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, found, err := c.GetPathMapping("/tmp/ws.yaml", "default", "id")
	if err != nil {
		t.Fatalf("GetPathMapping: %v", err)
	}
	if found {
		t.Fatalf("expected version mismatch to be a miss, got %#v", got)
	}
}

func TestPathMappingMissingIsMiss(t *testing.T) {
	c := newTestCache(t)
	_, found, err := c.GetPathMapping("/tmp/ws.yaml", "default", "missing")
	if err != nil {
		t.Fatalf("GetPathMapping: %v", err)
	}
	if found {
		t.Fatal("expected missing path mapping to be a miss")
	}
}

func TestIsPathMappingKey(t *testing.T) {
	if !IsPathMappingKey(PathMappingCacheKey("abc")) {
		t.Fatal("expected PathMappingCacheKey to be recognized")
	}
	if IsPathMappingKey("DB_USERNAME") || IsPathMappingKey("*") || IsPathMappingKey("DB") {
		t.Fatal("did not expect regular env names to be path mapping keys")
	}
}

func TestPathMappingReplaceDropsStaleMembers(t *testing.T) {
	c := newTestCache(t)
	first := CachedPathMapping{
		Entries: []CachedPathEntry{
			{EnvironmentVariable: "DB_USERNAME", SourceRef: "database-username", Value: "alice"},
			{EnvironmentVariable: "DB_PASSWORD", SourceRef: "database-password", Value: "s3cret"},
		},
	}
	if err := c.SetPathMapping("/tmp/ws.yaml", "default", "id", first, time.Hour); err != nil {
		t.Fatalf("SetPathMapping first: %v", err)
	}
	second := CachedPathMapping{
		Entries: []CachedPathEntry{
			{EnvironmentVariable: "DB_USERNAME", SourceRef: "database-username", Value: "alice"},
			{EnvironmentVariable: "DB_HOST", SourceRef: "database-host", Value: "localhost"},
		},
	}
	if err := c.SetPathMapping("/tmp/ws.yaml", "default", "id", second, time.Hour); err != nil {
		t.Fatalf("SetPathMapping second: %v", err)
	}
	got, found, err := c.GetPathMapping("/tmp/ws.yaml", "default", "id")
	if err != nil || !found {
		t.Fatalf("GetPathMapping: found=%v err=%v", found, err)
	}
	byEnv := map[string]CachedPathEntry{}
	for _, e := range got.Entries {
		byEnv[e.EnvironmentVariable] = e
	}
	if _, ok := byEnv["DB_PASSWORD"]; ok {
		t.Fatalf("stale DB_PASSWORD survived refresh: %#v", got.Entries)
	}
	if byEnv["DB_HOST"].Value != "localhost" {
		t.Fatalf("expected DB_HOST after refresh, got %#v", got.Entries)
	}
}

func TestExpandListEntries(t *testing.T) {
	now := time.Now()
	expires := now.Add(time.Hour)
	direct := CacheEntry{
		Path: "/tmp/ws.yaml", ConfigEnv: "default", Env: "APP_TOKEN",
		Value: "t0ken", CreatedAt: now, ExpiresAt: expires,
	}
	mapping := CachedPathMapping{
		Version: PathMappingVersion,
		Entries: []CachedPathEntry{
			{EnvironmentVariable: "PASSWORD", SourceRef: "/local/withsecrets/config/PASSWORD", Value: "s3cret"},
			{EnvironmentVariable: "USERNAME", SourceRef: "/local/withsecrets/terraform/USERNAME", Value: "alice"},
		},
	}
	data, err := marshalPathMapping(mapping)
	if err != nil {
		t.Fatalf("marshalPathMapping: %v", err)
	}
	pathRow := CacheEntry{
		Path: "/tmp/ws.yaml", ConfigEnv: "default", Env: PathMappingCacheKey("id"),
		Value: string(data), CreatedAt: now, ExpiresAt: expires,
	}
	malformed := CacheEntry{
		Path: "/tmp/ws.yaml", ConfigEnv: "default", Env: PathMappingCacheKey("bad"),
		Value: "not-json", CreatedAt: now, ExpiresAt: expires,
	}

	got := ExpandListEntries([]CacheEntry{pathRow, direct, malformed})
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3: %#v", len(got), got)
	}
	if got[0].Env != "APP_TOKEN" || got[1].Env != "PASSWORD" || got[2].Env != "USERNAME" {
		t.Fatalf("unexpected order: %#v", []string{got[0].Env, got[1].Env, got[2].Env})
	}
	if got[1].Value != "s3cret" || got[2].Value != "alice" {
		t.Fatalf("unexpected expanded values: %#v", got)
	}
	for _, e := range got {
		if IsPathMappingKey(e.Env) {
			t.Fatalf("list still contains path mapping key: %#v", e)
		}
	}
}
