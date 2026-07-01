package ws

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLookPathWithEnv_RespectsEnvPATH(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses unix executable bits")
	}

	dir := t.TempDir()
	exe := filepath.Join(dir, "turbo")
	if err := os.WriteFile(exe, []byte("#!/usr/bin/env sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write temp executable: %v", err)
	}

	got, err := lookPathWithEnv("turbo", []string{"PATH=" + dir})
	if err != nil {
		t.Fatalf("expected to resolve turbo, got err: %v", err)
	}
	if got != exe {
		t.Fatalf("expected %q, got %q", exe, got)
	}
}

func TestLookPathWithEnv_ExplicitPathPassthrough(t *testing.T) {
	explicit := "/some/explicit/path"
	if runtime.GOOS == "windows" {
		explicit = `C:\some\explicit\path`
	}
	got, err := lookPathWithEnv(explicit, []string{"PATH=/nope"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got != explicit {
		t.Fatalf("expected %q, got %q", explicit, got)
	}
}

func TestLookPathWithEnvWindows_RespectsPATHEXT(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific PATH/PATHEXT behavior")
	}

	dir := t.TempDir()
	exe := filepath.Join(dir, "turbo.cmd")
	if err := os.WriteFile(exe, []byte("@echo off\r\nexit /b 0\r\n"), 0o644); err != nil {
		t.Fatalf("write temp executable: %v", err)
	}

	got, err := lookPathWithEnv("turbo", []string{"PATH=" + dir, "PATHEXT=.COM;.EXE;.BAT;.CMD"})
	if err != nil {
		t.Fatalf("expected to resolve turbo, got err: %v", err)
	}
	if got != exe {
		t.Fatalf("expected %q, got %q", exe, got)
	}
}
