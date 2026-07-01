package ws

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunShowCommandListsEnvironments(t *testing.T) {
	t.Cleanup(func() {
		showEnvironment = "default"
		showConfigFile = ""
		showSensitive = false
		showOutput = "dotenv"
	})

	tmpFile, err := os.CreateTemp("", "kuba-show-*.yaml")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(tmpFile.Name()) })

	configContent := `
default:
  provider: local
  env:
    FOO:
      value: foo
staging:
  provider: local
  env:
    BAR:
      value: bar
`
	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	showEnvironment = showListEnvironmentsValue
	showConfigFile = tmpFile.Name()

	// Capture stdout
	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	runErr := runShowCommand(nil, true)

	require.NoError(t, runErr)

	require.NoError(t, w.Close())
	os.Stdout = originalStdout

	outputBytes, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())

	output := strings.Split(strings.TrimSpace(string(outputBytes)), "\n")
	assert.Equal(t, []string{"default", "staging"}, output)
}

func TestRunShowCommandUsesProvidedEnvironment(t *testing.T) {
	t.Cleanup(func() {
		showEnvironment = "default"
		showConfigFile = ""
		showSensitive = false
		showOutput = "dotenv"
	})

	tmpFile, err := os.CreateTemp("", "kuba-show-*.yaml")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(tmpFile.Name()) })

	configContent := `
default:
  provider: local
  env:
    FOO:
      value: foo
staging:
  provider: local
  env:
    BAR:
      value: bar
`
	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	showEnvironment = "staging"
	showConfigFile = tmpFile.Name()

	// Capture stdout
	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	runErr := runShowCommand(nil, false)

	require.NoError(t, runErr)

	require.NoError(t, w.Close())
	os.Stdout = originalStdout

	outputBytes, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())

	output := strings.TrimSpace(string(outputBytes))
	assert.Equal(t, "BAR=bar", output)
}

func TestRunShowCommandConsumesArgWhenEnvFlagNoOptSet(t *testing.T) {
	t.Cleanup(func() {
		showEnvironment = "default"
		showConfigFile = ""
		showSensitive = false
		showOutput = "dotenv"
	})

	tmpFile, err := os.CreateTemp("", "kuba-show-*.yaml")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(tmpFile.Name()) })

	configContent := `
default:
  provider: local
  env:
    FOO:
      value: foo
staging:
  provider: local
  env:
    BAR:
      value: bar
`
	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	showEnvironment = showListEnvironmentsValue
	showConfigFile = tmpFile.Name()

	// Capture stdout
	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	runErr := runShowCommand([]string{"staging"}, true)

	require.NoError(t, runErr)

	require.NoError(t, w.Close())
	os.Stdout = originalStdout

	outputBytes, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())

	output := strings.TrimSpace(string(outputBytes))
	assert.Equal(t, "BAR=bar", output)
}

func TestRunShowCommandOutputsJSON(t *testing.T) {
	t.Cleanup(func() {
		showEnvironment = "default"
		showConfigFile = ""
		showSensitive = false
		showOutput = "dotenv"
	})

	tmpFile, err := os.CreateTemp("", "kuba-show-*.yaml")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(tmpFile.Name()) })

	configContent := `
default:
  provider: local
  env:
    FOO:
      value: foo
    BAR:
      value: bar
`
	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	showEnvironment = "default"
	showConfigFile = tmpFile.Name()
	showOutput = "json"

	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	runErr := runShowCommand(nil, false)
	require.NoError(t, runErr)

	require.NoError(t, w.Close())
	os.Stdout = originalStdout

	outputBytes, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())

	var output map[string]string
	require.NoError(t, json.Unmarshal(outputBytes, &output))

	assert.Equal(t, map[string]string{
		"BAR": "bar",
		"FOO": "foo",
	}, output)
}

func TestRunShowCommandOutputsShell(t *testing.T) {
	t.Cleanup(func() {
		showEnvironment = "default"
		showConfigFile = ""
		showSensitive = false
		showOutput = "dotenv"
	})

	tmpFile, err := os.CreateTemp("", "kuba-show-*.yaml")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(tmpFile.Name()) })

	configContent := `
default:
  provider: local
  env:
    FOO:
      value: foo
    BAR:
      value: bar
`
	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	showEnvironment = "default"
	showConfigFile = tmpFile.Name()
	showOutput = "shell"

	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	runErr := runShowCommand(nil, false)
	require.NoError(t, runErr)

	require.NoError(t, w.Close())
	os.Stdout = originalStdout

	outputBytes, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())

	output := strings.Split(strings.TrimSpace(string(outputBytes)), "\n")
	assert.Equal(t, []string{
		"export BAR=bar",
		"export FOO=foo",
	}, output)
}
