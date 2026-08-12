package secrets

import (
	"testing"
)

func TestSanitizeEnvVarName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal-name", "NORMAL_NAME"},
		{"name_with_underscores", "NAME_WITH_UNDERSCORES"},
		{"123-start-with-number", "_123_START_WITH_NUMBER"},
		{"start-with-dash", "START_WITH_DASH"},
		{"has.dots", "HAS_DOTS"},
		{"has spaces", "HAS_SPACES"},
		{"has@special#chars", "HAS_SPECIAL_CHARS"},
		{"", "_"},
		{"a", "A"},
		{"A", "A"},
		{"_", "_"},
		{"1", "_1"},
		{"-", "_"},
		{"my-app-config", "MY_APP_CONFIG"},
		{"database.password", "DATABASE_PASSWORD"},
		{"API_KEY_123", "API_KEY_123"},
		{"MixedCase-Name", "MIXEDCASE_NAME"},
		{"lowercase", "LOWERCASE"},
		{"UPPERCASE", "UPPERCASE"},
		{"camelCase", "CAMELCASE"},
		{"snake_case", "SNAKE_CASE"},
		{"kebab-case", "KEBAB_CASE"},
		{"dot.notation", "DOT_NOTATION"},
		{"path/to/secret", "PATH_TO_SECRET"},
		{"file-name.txt", "FILE_NAME_TXT"},
		{"config.json", "CONFIG_JSON"},
		{"env-var-123", "ENV_VAR_123"},
		{"test@example.com", "TEST_EXAMPLE_COM"},
		{"user-name_123", "USER_NAME_123"},
	}

	for _, test := range tests {
		result := sanitizeEnvVarName(test.input)
		if result != test.expected {
			t.Errorf("sanitizeEnvVarName(%q) = %q, want %q", test.input, result, test.expected)
		}
	}
}

func TestExtractSecretNameFromPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple-name", "simple-name"},
		{"path/to/secret", "secret"},
		{"path\\to\\secret", "secret"},
		{"path:to:secret", "secret"},
		{"projects/myproject/secrets/mysecret", "mysecret"},
		{"", ""},
		{"single", "single"},
	}

	for _, test := range tests {
		result := extractSecretNameFromPath(test.input)
		if result != test.expected {
			t.Errorf("extractSecretNameFromPath(%q) = %q, want %q", test.input, result, test.expected)
		}
	}
}

func TestRelativeEnvVarName(t *testing.T) {
	tests := []struct {
		basePath string
		fullName string
		expected string
	}{
		{"/local/withsecrets/config", "/local/withsecrets/config/USERNAME", "USERNAME"},
		{"/local/withsecrets/config/", "/local/withsecrets/config/PASSWORD", "PASSWORD"},
		{"/local/withsecrets/config", "/local/withsecrets/config/db/user", "DB_USER"},
		{"database", "database-username", "USERNAME"},
		{"database", "database_password", "PASSWORD"},
		{"/app", "/app/api-key", "API_KEY"},
		{"", "/foo/bar", "FOO_BAR"},
		{"/other", "/local/config/USER", "USER"},
	}

	for _, test := range tests {
		result := relativeEnvVarName(test.basePath, test.fullName)
		if result != test.expected {
			t.Errorf("relativeEnvVarName(%q, %q) = %q, want %q", test.basePath, test.fullName, result, test.expected)
		}
	}
}
