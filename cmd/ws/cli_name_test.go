package ws

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBinaryAssetName(t *testing.T) {
	t.Run("ws", func(t *testing.T) {
		old := os.Args[0]
		os.Args[0] = "ws"
		t.Cleanup(func() { os.Args[0] = old })
		if got := BinaryAssetName(); got != "ws" {
			t.Fatalf("BinaryAssetName() = %q, want ws", got)
		}
	})

	t.Run("kuba", func(t *testing.T) {
		old := os.Args[0]
		os.Args[0] = "kuba"
		t.Cleanup(func() { os.Args[0] = old })
		if got := BinaryAssetName(); got != "kuba" {
			t.Fatalf("BinaryAssetName() = %q, want kuba", got)
		}
	})

	t.Run("kuba with path", func(t *testing.T) {
		old := os.Args[0]
		os.Args[0] = filepath.Join("/usr/local/bin", "kuba")
		t.Cleanup(func() { os.Args[0] = old })
		if got := BinaryAssetName(); got != "kuba" {
			t.Fatalf("BinaryAssetName() = %q, want kuba", got)
		}
	})
}
