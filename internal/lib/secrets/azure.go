package secrets

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

// AzureKeyVaultManager handles Azure Key Vault operations
type AzureKeyVaultManager struct {
	client *azsecrets.Client
	ctx    context.Context
}

// NewAzureKeyVaultManager creates a new Azure Key Vault client
func NewAzureKeyVaultManager(ctx context.Context, vaultURL string, tenantID string, clientID string, clientSecret string) (*AzureKeyVaultManager, error) {
	var cred azcore.TokenCredential
	var err error

	// Try different authentication methods in order of preference
	if clientID != "" && clientSecret != "" && tenantID != "" {
		// Use service principal authentication
		cred, err = azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create client secret credential: %w", err)
		}
	} else if os.Getenv("AZURE_CLIENT_ID") != "" && os.Getenv("AZURE_CLIENT_SECRET") != "" && os.Getenv("AZURE_TENANT_ID") != "" {
		// Use environment variables for service principal
		cred, err = azidentity.NewClientSecretCredential(
			os.Getenv("AZURE_TENANT_ID"),
			os.Getenv("AZURE_CLIENT_ID"),
			os.Getenv("AZURE_CLIENT_SECRET"),
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create client secret credential from environment: %w", err)
		}
	} else {
		// Try managed identity or default Azure credential
		cred, err = azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create default Azure credential: %w", err)
		}
	}

	// Create the Key Vault client
	client, err := azsecrets.NewClient(vaultURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Key Vault client: %w", err)
	}

	return &AzureKeyVaultManager{
		client: client,
		ctx:    ctx,
	}, nil
}

// GetSecret retrieves a secret from Azure Key Vault
// Note: In Azure, projectID is not used, but we keep the interface consistent
func (a *AzureKeyVaultManager) GetSecret(projectID, secretID string) (string, error) {
	// Get the secret value
	resp, err := a.client.GetSecret(a.ctx, secretID, "", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get secret '%s': %w", secretID, err)
	}

	// Convert the secret value to string
	if resp.Value != nil {
		return *resp.Value, nil
	}

	return "", fmt.Errorf("secret '%s' has no value", secretID)
}

// Close closes the Azure Key Vault client
// Note: Azure SDK doesn't require explicit closing, but we keep the interface consistent
func (a *AzureKeyVaultManager) Close() error {
	// Azure SDK clients are stateless and don't need to be closed
	return nil
}

// GetSecrets retrieves multiple secrets from Azure Key Vault
// Note: In Azure, projectID is not used, but we keep the interface consistent
func (a *AzureKeyVaultManager) GetSecrets(projectID string, secretIDs []string) (map[string]string, error) {
	secrets := make(map[string]string)

	for _, secretID := range secretIDs {
		secret, err := a.GetSecret(projectID, secretID)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret '%s': %w", secretID, err)
		}
		secrets[secretID] = secret
	}

	return secrets, nil
}

// GetSecretsByPath retrieves all secrets that start with the given path prefix
func (a *AzureKeyVaultManager) GetSecretsByPath(projectID, secretPath string) (map[string]string, error) {
	secrets := make(map[string]string)

	// List all secrets
	secretNames, err := a.ListSecrets()
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	// Filter secrets that start with the path prefix
	for _, secretName := range secretNames {
		if strings.HasPrefix(secretName, secretPath) {
			// Get the actual secret value
			secretValue, err := a.GetSecret(projectID, secretName)
			if err != nil {
				// Log warning but continue with other secrets
				fmt.Printf("Warning: failed to get secret '%s': %v\n", secretName, err)
				continue
			}

			// Sanitize the secret name for use as an environment variable name
			envVarName := relativeEnvVarName(secretPath, secretName)
			secrets[envVarName] = secretValue
		}
	}

	return secrets, nil
}

// ListSecrets lists all available secrets (Azure-specific method)
func (a *AzureKeyVaultManager) ListSecrets() ([]string, error) {
	var secretNames []string

	// List all secrets
	pager := a.client.NewListSecretPropertiesPager(nil)
	for pager.More() {
		page, err := pager.NextPage(a.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}

		for _, secret := range page.Value {
			if secret.ID != nil {
				secretNames = append(secretNames, secret.ID.Name())
			}
		}
	}

	return secretNames, nil
}

// CreateSecret creates a new secret in Azure Key Vault (Azure-specific method)
func (a *AzureKeyVaultManager) CreateSecret(secretName, secretValue, description string) error {
	// Set secret parameters
	parameters := azsecrets.SetSecretParameters{
		Value: &secretValue,
	}

	if description != "" {
		parameters.Tags = map[string]*string{
			"description": &description,
		}
	}

	// Create the secret
	_, err := a.client.SetSecret(a.ctx, secretName, parameters, nil)
	if err != nil {
		return fmt.Errorf("failed to create secret '%s': %w", secretName, err)
	}

	return nil
}

// UpdateSecret updates an existing secret in Azure Key Vault (Azure-specific method)
func (a *AzureKeyVaultManager) UpdateSecret(secretName, secretValue string) error {
	// Update the secret
	_, err := a.client.SetSecret(a.ctx, secretName, azsecrets.SetSecretParameters{
		Value: &secretValue,
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to update secret '%s': %w", secretName, err)
	}

	return nil
}

// DeleteSecret deletes a secret from Azure Key Vault (Azure-specific method)
func (a *AzureKeyVaultManager) DeleteSecret(secretName string, forceDelete bool) error {
	// Delete the secret
	_, err := a.client.DeleteSecret(a.ctx, secretName, nil)
	if err != nil {
		return fmt.Errorf("failed to delete secret '%s': %w", secretName, err)
	}

	// If force delete is requested, purge the secret immediately
	if forceDelete {
		_, err = a.client.PurgeDeletedSecret(a.ctx, secretName, nil)
		if err != nil {
			return fmt.Errorf("failed to purge deleted secret '%s': %w", secretName, err)
		}
	}

	return nil
}
