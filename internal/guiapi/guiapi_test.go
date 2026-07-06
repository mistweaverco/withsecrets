package guiapi

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const testConfig = `---
default:
  provider: gcp
  project: test-project
  env:
    TEST_VAR:
      value: inline-value
    SECRET_VAR:
      secret-key: my-secret
staging:
  provider: aws
  project: aws-project
  env:
    AWS_VAR:
      secret-path: /path/to/secret
`

func writeTestConfig(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "ws-*.yaml")
	require.NoError(t, err)
	_, err = f.WriteString(testConfig)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func TestListEnvironments(t *testing.T) {
	path := writeTestConfig(t)
	envs, err := ListEnvironments(path)
	require.NoError(t, err)
	require.Len(t, envs, 2)
	assertEnv(t, envs[0], "default", "gcp", "test-project")
	assertEnv(t, envs[1], "staging", "aws", "aws-project")
}

func assertEnv(t *testing.T, got EnvironmentSummary, name, provider, project string) {
	t.Helper()
	if got.Name != name || got.Provider != provider || got.Project != project {
		t.Fatalf("expected %+v, got %+v", EnvironmentSummary{name, provider, project}, got)
	}
}

func TestListSecretsReturnsPlainAndMaskedValues(t *testing.T) {
	path := writeTestConfig(t)
	rows, err := ListSecrets(t.Context(), path, "default")
	require.NoError(t, err)
	require.Len(t, rows, 2)

	var inline SecretRow
	for _, row := range rows {
		if row.EnvVar == "TEST_VAR" {
			inline = row
			break
		}
	}
	require.Equal(t, "inline-value", inline.Value)
	require.Equal(t, MaskValue("inline-value"), inline.MaskedValue)
}

func TestUpdateSecretRejectsNonSecretKey(t *testing.T) {
	path := writeTestConfig(t)
	err := UpdateSecret(t.Context(), path, "default", "TEST_VAR", "new")
	require.Error(t, err)
	require.Contains(t, err.Error(), "secret-key")
}

func TestDeleteSecretRejectsNonSecretKey(t *testing.T) {
	path := writeTestConfig(t)
	err := DeleteSecret(t.Context(), path, "staging", "AWS_VAR")
	require.Error(t, err)
	require.Contains(t, err.Error(), "secret-key")
}

func TestCreateSecretRequiresFields(t *testing.T) {
	path := writeTestConfig(t)
	err := CreateSecret(t.Context(), CreateInput{
		ConfigPath: path,
		EnvName:    "default",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "required")
}
