package secrets

import (
	"regexp"
	"strings"
)

// sanitizeEnvVarName sanitizes a string to be a valid POSIX environment variable name
// POSIX rules: must begin with a letter or underscore, and contain only letters, numbers, or underscores
// Also make sure it's all uppercase for convention
func sanitizeEnvVarName(name string) string {
	if name == "" {
		return "_"
	}

	// Replace any non-alphanumeric characters with underscores
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	sanitized := re.ReplaceAllString(name, "_")

	// Ensure it starts with a letter or underscore
	if len(sanitized) > 0 {
		firstChar := sanitized[0]
		if (firstChar < 'a' || firstChar > 'z') && (firstChar < 'A' || firstChar > 'Z') && firstChar != '_' {
			sanitized = "_" + sanitized
		}
	}

	// If the result is empty after sanitization, return underscore
	if sanitized == "" {
		return "_"
	}

	// Convert to uppercase for convention
	return strings.ToUpper(sanitized)
}

// relativeEnvVarName returns a sanitized env var name from the portion of fullName
// after basePath. Nested remainders like "db/user" become "DB_USER".
func relativeEnvVarName(basePath, fullName string) string {
	base := strings.TrimRight(basePath, "/")
	name := strings.TrimRight(fullName, "/")

	var relative string
	switch {
	case base == "":
		relative = strings.TrimLeft(name, "/")
	case name == base:
		relative = extractSecretNameFromPath(name)
	case strings.HasPrefix(name, base+"/"):
		relative = strings.TrimPrefix(name, base+"/")
	case strings.HasPrefix(name, base):
		// Prefix match without separator (e.g. "database" + "database-username")
		relative = strings.TrimPrefix(name, base)
		relative = strings.TrimLeft(relative, "/-_.")
		if relative == "" {
			relative = extractSecretNameFromPath(name)
		}
	default:
		relative = extractSecretNameFromPath(name)
	}

	if relative == "" {
		relative = name
	}
	return sanitizeEnvVarName(relative)
}

// extractSecretNameFromPath extracts the secret name from a full secret path
// This is useful for providers where the path contains additional metadata
func extractSecretNameFromPath(path string) string {
	// Split by common separators and take the last part
	parts := strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == '\\' || r == ':'
	})

	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return path
}
