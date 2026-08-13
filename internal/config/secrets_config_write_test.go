package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ws.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestAddOrUpdateEnvPathMapping(t *testing.T) {
	base := `---
default:
  provider: aws
  env:
    FOO:
      value: bar
`

	t.Run("writes scalar for single path", func(t *testing.T) {
		path := writeTempConfig(t, base)
		err := AddOrUpdateEnvPathMapping(path, "default", "*", "secret-path", []string{"/shared"})
		require.NoError(t, err)

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		text := string(data)
		require.Contains(t, text, `"*"`)
		require.Contains(t, text, "secret-path: /shared")
	})

	t.Run("writes sequence for multiple paths", func(t *testing.T) {
		path := writeTempConfig(t, base)
		err := AddOrUpdateEnvPathMapping(path, "default", "DB", "param-path", []string{"/a", "/b"})
		require.NoError(t, err)

		cfg, err := LoadSecretsConfig(path)
		require.NoError(t, err)
		require.Equal(t, []string{"/a", "/b"}, cfg.Environments["default"].Env["DB"].ParamPath)
	})

	t.Run("rejects empty paths", func(t *testing.T) {
		path := writeTempConfig(t, base)
		err := AddOrUpdateEnvPathMapping(path, "default", "*", "secret-path", []string{"", "  "})
		require.Error(t, err)
	})

	t.Run("rejects invalid kind", func(t *testing.T) {
		path := writeTempConfig(t, base)
		err := AddOrUpdateEnvPathMapping(path, "default", "*", "secret-key", []string{"/a"})
		require.Error(t, err)
	})
}

func TestPathValueNode(t *testing.T) {
	scalar := PathValueNode([]string{"/a"})
	require.Equal(t, yaml.ScalarNode, scalar.Kind)
	require.Equal(t, "/a", scalar.Value)

	seq := PathValueNode([]string{"/a", "/b"})
	require.Equal(t, yaml.SequenceNode, seq.Kind)
	require.Len(t, seq.Content, 2)
	require.Equal(t, "/a", seq.Content[0].Value)
	require.Equal(t, "/b", seq.Content[1].Value)
}

func TestFormatPathRef(t *testing.T) {
	require.Equal(t, "/a, /b", FormatPathRef([]string{"/a", "/b"}))
	require.Equal(t, "", FormatPathRef(nil))
}

func TestParsePathLines(t *testing.T) {
	require.Equal(t, []string{"/a", "/b"}, ParsePathLines("/a\n/b\n"))
	require.Equal(t, []string{"/a", "/b"}, ParsePathLines(" /a \r\n\r\n /b "))
	require.Empty(t, ParsePathLines("\n  \n"))
}

func TestQuotedStarKeyRoundTrip(t *testing.T) {
	path := writeTempConfig(t, `---
default:
  provider: aws
  env:
    FOO:
      value: bar
`)
	require.NoError(t, AddOrUpdateEnvPathMapping(path, "default", "*", "param-path", []string{"/a", "/b"}))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.True(t, strings.Contains(string(data), `"*"`), "expected quoted * key, got:\n%s", data)

	cfg, err := LoadSecretsConfig(path)
	require.NoError(t, err)
	require.Equal(t, []string{"/a", "/b"}, cfg.Environments["default"].Env["*"].ParamPath)
}
