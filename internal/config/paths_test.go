package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindConfigFilePriority(t *testing.T) {
	root := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	write := func(name string) {
		if err := os.WriteFile(filepath.Join(root, name), []byte("default:\n  provider: local\n  env:\n    X:\n      value: y\n"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	if _, err := FindConfigFile(); err == nil {
		t.Fatalf("expected error when no config exists")
	}

	write("kuba.yaml")
	got, err := FindConfigFile()
	if err != nil || filepath.Base(got) != "kuba.yaml" {
		t.Fatalf("expected kuba.yaml, got %q err=%v", got, err)
	}

	write("withsecrets.yaml")
	got, err = FindConfigFile()
	if err != nil || filepath.Base(got) != "withsecrets.yaml" {
		t.Fatalf("expected withsecrets.yaml, got %q err=%v", got, err)
	}

	write("ws.yaml")
	got, err = FindConfigFile()
	if err != nil || filepath.Base(got) != "ws.yaml" {
		t.Fatalf("expected ws.yaml, got %q err=%v", got, err)
	}
}

func TestGlobalConfigFallbackToLegacyPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if runtimeHome := os.Getenv("USERPROFILE"); runtimeHome != "" {
		t.Setenv("USERPROFILE", home)
	}

	legacyDir := LegacyGlobalConfigDir(home)
	if err := os.MkdirAll(legacyDir, 0755); err != nil {
		t.Fatalf("mkdir legacy: %v", err)
	}
	legacyPath := filepath.Join(legacyDir, "config.yaml")
	if err := os.WriteFile(legacyPath, []byte("cache: false\n"), 0644); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	cfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("LoadGlobalConfig: %v", err)
	}
	if cfg.Cache.Enabled {
		t.Fatalf("expected cache disabled from legacy config")
	}
}

func TestCacheDirUsesLegacyWhenNewMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if runtimeHome := os.Getenv("USERPROFILE"); runtimeHome != "" {
		t.Setenv("USERPROFILE", home)
	}

	legacyDir := LegacyCacheDir(home)
	if err := os.MkdirAll(legacyDir, 0755); err != nil {
		t.Fatalf("mkdir legacy cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "db.sqlite"), []byte("legacy"), 0644); err != nil {
		t.Fatalf("write legacy db: %v", err)
	}

	got, err := GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir: %v", err)
	}
	if got != legacyDir {
		t.Fatalf("expected legacy cache dir %q, got %q", legacyDir, got)
	}
}
