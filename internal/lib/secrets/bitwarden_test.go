package secrets

import (
	"context"
	"os"
	"testing"
)

func TestNewBitwardenManager_MissingEnv(t *testing.T) {
	ctx := context.Background()

	// Ensure relevant env vars are unset so we hit the validation path
	os.Unsetenv("BITWARDEN_ORGANIZATION_ID")
	os.Unsetenv("BITWARDEN_ACCESS_TOKEN")
	os.Unsetenv("ACCESS_TOKEN")

	manager, err := NewBitwardenManager(ctx, "")
	if err == nil {
		// If this unexpectedly succeeds, make sure we can close it cleanly
		if manager != nil {
			_ = manager.Close()
		}
		t.Fatalf("expected error when required Bitwarden env vars are missing")
	}
}

func TestNewBitwardenManager_WithEnv(t *testing.T) {
	ctx := context.Background()

	// Set fake env vars; the SDK may still fail to initialize (e.g. missing native libs),
	// which is acceptable in this test - we just care that we don't panic.
	os.Setenv("BITWARDEN_ORGANIZATION_ID", "test-org")
	os.Setenv("BITWARDEN_ACCESS_TOKEN", "test-token")
	defer os.Unsetenv("BITWARDEN_ORGANIZATION_ID")
	defer os.Unsetenv("BITWARDEN_ACCESS_TOKEN")

	manager, err := NewBitwardenManager(ctx, "")
	if err != nil {
		// In CI or local dev without Bitwarden SDK/native libs, an error is expected.
		t.Logf("Bitwarden manager could not be created in this environment (expected in tests): %v", err)
		return
	}

	if manager == nil {
		t.Fatal("expected Bitwarden manager to be created")
	}

	if cerr := manager.Close(); cerr != nil {
		t.Fatalf("expected Bitwarden manager to close without error, got: %v", cerr)
	}
}
