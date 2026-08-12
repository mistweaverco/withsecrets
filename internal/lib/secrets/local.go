package secrets

import (
	"context"
	"fmt"
	"os"
)

// LocalManager implements SecretManager for local environment variables
type LocalManager struct {
	ctx context.Context
}

// NewLocalManager creates a new local secret manager
func NewLocalManager(ctx context.Context) (SecretManager, error) {
	return &LocalManager{
		ctx: ctx,
	}, nil
}

// GetSecret retrieves a single secret from environment variables
func (l *LocalManager) GetSecret(projectID, secretID string) (string, error) {
	// For local provider, we just return the environment variable value
	value := os.Getenv(secretID)
	if value == "" {
		return "", fmt.Errorf("environment variable '%s' not found", secretID)
	}
	return value, nil
}

// GetSecrets retrieves multiple secrets from environment variables
func (l *LocalManager) GetSecrets(projectID string, secretIDs []string) (map[string]string, error) {
	secrets := make(map[string]string)

	for _, secretID := range secretIDs {
		value := os.Getenv(secretID)
		if value != "" {
			secrets[secretID] = value
		}
		// Note: We don't return an error if a secret is not found,
		// we just skip it to be consistent with other providers
	}

	return secrets, nil
}

// GetSecretsByPath retrieves all environment variables that start with the given path
func (l *LocalManager) GetSecretsByPath(projectID, secretPath string) (map[string]string, error) {
	secrets := make(map[string]string)

	// Get all environment variables
	for _, env := range os.Environ() {
		// Split on first '=' to get key and value
		parts := splitEnvVar(env)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]

			// Check if this environment variable starts with our path
			if len(key) >= len(secretPath) && key[:len(secretPath)] == secretPath {
				envVarName := relativeEnvVarName(secretPath, key)
				secrets[envVarName] = value
			}
		}
	}

	return secrets, nil
}

// Close closes the local manager (no-op for local provider)
func (l *LocalManager) Close() error {
	return nil
}

// splitEnvVar splits an environment variable string on the first '=' character
func splitEnvVar(env string) []string {
	for i, char := range env {
		if char == '=' {
			return []string{env[:i], env[i+1:]}
		}
	}
	return []string{env}
}
