package secrets

import (
	"context"
	"fmt"
	"net/http"
	"time"

	vault "github.com/openbao/openbao/api/v2"
)

// OpenBaoManager handles OpenBao operations
type OpenBaoManager struct {
	client *vault.Client
	ctx    context.Context
}

// NewOpenBaoManager creates a new OpenBao client
func NewOpenBaoManager(ctx context.Context, address string, token string, namespace string) (*OpenBaoManager, error) {
	// Create the client configuration
	config := vault.DefaultConfig()

	// Set the OpenBao server address
	if address != "" {
		config.Address = address
	}

	// Configure HTTP client with reasonable timeouts
	config.HttpClient = &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create the client
	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenBao client: %w", err)
	}

	// Set the token if provided
	if token != "" {
		client.SetToken(token)
	}

	// Set the namespace if provided
	if namespace != "" {
		client.SetNamespace(namespace)
	}

	return &OpenBaoManager{
		client: client,
		ctx:    ctx,
	}, nil
}

// GetSecret retrieves a secret from OpenBao
func (o *OpenBaoManager) GetSecret(projectID, secretID string) (string, error) {
	// In OpenBao, we use the secret path (secretID) to retrieve the secret
	// The projectID can be used as a namespace prefix if needed
	secretPath := secretID
	if projectID != "" {
		secretPath = fmt.Sprintf("%s/%s", projectID, secretID)
	}

	// Read the secret from OpenBao
	secret, err := o.client.Logical().Read(secretPath)
	if err != nil {
		return "", fmt.Errorf("failed to read secret '%s': %w", secretPath, err)
	}

	if secret == nil {
		return "", fmt.Errorf("secret '%s' not found", secretPath)
	}

	// OpenBao secrets are stored as key-value pairs
	// We'll return the first value we find, or an error if no values exist
	if len(secret.Data) == 0 {
		return "", fmt.Errorf("secret '%s' has no data", secretPath)
	}

	// If there's only one key-value pair, return its value
	if len(secret.Data) == 1 {
		for _, value := range secret.Data {
			if str, ok := value.(string); ok {
				return str, nil
			}
			return fmt.Sprintf("%v", value), nil
		}
	}

	// If there are multiple key-value pairs, return the first string value
	for _, value := range secret.Data {
		if str, ok := value.(string); ok {
			return str, nil
		}
	}

	// If no string values found, return an error
	return "", fmt.Errorf("secret '%s' contains no string values", secretPath)
}

// GetSecrets retrieves multiple secrets from OpenBao
func (o *OpenBaoManager) GetSecrets(projectID string, secretIDs []string) (map[string]string, error) {
	secrets := make(map[string]string)

	for _, secretID := range secretIDs {
		secret, err := o.GetSecret(projectID, secretID)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret '%s': %w", secretID, err)
		}
		secrets[secretID] = secret
	}

	return secrets, nil
}

// GetSecretsByPath retrieves all secrets that start with the given path prefix
func (o *OpenBaoManager) GetSecretsByPath(projectID, secretPath string) (map[string]string, error) {
	secrets := make(map[string]string)

	// List all secrets at the path
	secretNames, err := o.ListSecrets(secretPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets at path '%s': %w", secretPath, err)
	}

	// Get each secret and add it to the result
	for _, secretName := range secretNames {
		// Get the actual secret value
		secretValue, err := o.GetSecret(projectID, secretName)
		if err != nil {
			// Log warning but continue with other secrets
			fmt.Printf("Warning: failed to get secret '%s': %v\n", secretName, err)
			continue
		}

		// Sanitize the secret name for use as an environment variable name
		envVarName := relativeEnvVarName(secretPath, secretName)
		secrets[envVarName] = secretValue
	}

	return secrets, nil
}

// Close closes the OpenBao client
func (o *OpenBaoManager) Close() error {
	// OpenBao client doesn't require explicit closing
	return nil
}

// ListSecrets lists all available secrets in a given path (OpenBao-specific method)
func (o *OpenBaoManager) ListSecrets(path string) ([]string, error) {
	// List secrets at the specified path
	secrets, err := o.client.Logical().List(path)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets at path '%s': %w", path, err)
	}

	if secrets == nil || secrets.Data == nil {
		return []string{}, nil
	}

	// Extract secret names from the response
	var secretNames []string
	if keys, ok := secrets.Data["keys"]; ok {
		if keyList, ok := keys.([]interface{}); ok {
			for _, key := range keyList {
				if keyStr, ok := key.(string); ok {
					secretNames = append(secretNames, keyStr)
				}
			}
		}
	}

	return secretNames, nil
}

// CreateSecret creates a new secret in OpenBao (OpenBao-specific method)
func (o *OpenBaoManager) CreateSecret(path string, data map[string]interface{}) error {
	_, err := o.client.Logical().Write(path, data)
	if err != nil {
		return fmt.Errorf("failed to create secret at path '%s': %w", path, err)
	}

	return nil
}

// UpdateSecret updates an existing secret in OpenBao (OpenBao-specific method)
func (o *OpenBaoManager) UpdateSecret(path string, data map[string]interface{}) error {
	// In OpenBao, writing to an existing path updates the secret
	return o.CreateSecret(path, data)
}

// DeleteSecret deletes a secret from OpenBao (OpenBao-specific method)
func (o *OpenBaoManager) DeleteSecret(path string) error {
	_, err := o.client.Logical().Delete(path)
	if err != nil {
		return fmt.Errorf("failed to delete secret at path '%s': %w", path, err)
	}

	return nil
}
