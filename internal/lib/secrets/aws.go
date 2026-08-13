package secrets

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// AWSSecretsManager handles AWS Secrets Manager and Parameter Store operations
type AWSSecretsManager struct {
	client    *secretsmanager.Client
	ssmClient *ssm.Client
	ctx       context.Context
}

// NewAWSSecretsManager creates a new AWS Secrets Manager / Parameter Store client
func NewAWSSecretsManager(ctx context.Context, region string, profile string) (*AWSSecretsManager, error) {
	var cfg aws.Config
	var err error

	if profile != "" {
		// Load config with specific profile
		cfg, err = config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile))
	} else {
		// Load default config (uses environment variables, IAM roles, etc.)
		cfg, err = config.LoadDefaultConfig(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Override region if specified
	if region != "" {
		cfg.Region = region
	}

	client := secretsmanager.NewFromConfig(cfg)
	ssmClient := ssm.NewFromConfig(cfg)

	return &AWSSecretsManager{
		client:    client,
		ssmClient: ssmClient,
		ctx:       ctx,
	}, nil
}

// GetSecret retrieves a secret from AWS Secrets Manager
// Note: In AWS, projectID is not used, but we keep the interface consistent
func (a *AWSSecretsManager) GetSecret(projectID, secretID string) (string, error) {
	// In AWS, we only need the secret name/ID
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretID),
	}

	result, err := a.client.GetSecretValue(a.ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get secret '%s': %w", secretID, err)
	}

	// Check if the secret is binary or string
	if result.SecretBinary != nil {
		return string(result.SecretBinary), nil
	}

	if result.SecretString != nil {
		return *result.SecretString, nil
	}

	return "", fmt.Errorf("secret '%s' has no value", secretID)
}

// Close closes the AWS Secrets Manager client
// Note: AWS SDK v2 doesn't require explicit closing, but we keep the interface consistent
func (a *AWSSecretsManager) Close() error {
	// AWS SDK v2 clients are stateless and don't need to be closed
	return nil
}

// GetSecrets retrieves multiple secrets from AWS Secrets Manager
// Note: In AWS, projectID is not used, but we keep the interface consistent
func (a *AWSSecretsManager) GetSecrets(projectID string, secretIDs []string) (map[string]string, error) {
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
func (a *AWSSecretsManager) GetSecretsByPath(projectID, secretPath string) (map[string]string, error) {
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

			secrets[secretName] = secretValue
		}
	}

	return secrets, nil
}

// GetParameter retrieves a parameter from AWS Systems Manager Parameter Store
func (a *AWSSecretsManager) GetParameter(projectID, paramName string) (string, error) {
	input := &ssm.GetParameterInput{
		Name:           aws.String(paramName),
		WithDecryption: aws.Bool(true),
	}

	result, err := a.ssmClient.GetParameter(a.ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get parameter '%s': %w", paramName, err)
	}

	if result.Parameter == nil || result.Parameter.Value == nil {
		return "", fmt.Errorf("parameter '%s' has no value", paramName)
	}

	return *result.Parameter.Value, nil
}

// GetParameters retrieves multiple parameters from AWS Systems Manager Parameter Store
func (a *AWSSecretsManager) GetParameters(projectID string, paramNames []string) (map[string]string, error) {
	params := make(map[string]string)

	for _, paramName := range paramNames {
		value, err := a.GetParameter(projectID, paramName)
		if err != nil {
			return nil, fmt.Errorf("failed to get parameter '%s': %w", paramName, err)
		}
		params[paramName] = value
	}

	return params, nil
}

// GetParametersByPath retrieves all parameters under the given path prefix
func (a *AWSSecretsManager) GetParametersByPath(projectID, paramPath string) (map[string]string, error) {
	params := make(map[string]string)

	input := &ssm.GetParametersByPathInput{
		Path:           aws.String(paramPath),
		Recursive:      aws.Bool(true),
		WithDecryption: aws.Bool(true),
	}

	paginator := ssm.NewGetParametersByPathPaginator(a.ssmClient, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(a.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list parameters under '%s': %w", paramPath, err)
		}

		for _, parameter := range page.Parameters {
			if parameter.Name == nil || parameter.Value == nil {
				continue
			}
			params[*parameter.Name] = *parameter.Value
		}
	}

	return params, nil
}

// ListParameters lists available Parameter Store parameter names (AWS-specific method)
func (a *AWSSecretsManager) ListParameters() ([]string, error) {
	input := &ssm.DescribeParametersInput{}

	var names []string
	paginator := ssm.NewDescribeParametersPaginator(a.ssmClient, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(a.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list parameters: %w", err)
		}
		for _, parameter := range page.Parameters {
			if parameter.Name != nil {
				names = append(names, *parameter.Name)
			}
		}
	}

	return names, nil
}

// ListSecrets lists all available secrets (AWS-specific method)
func (a *AWSSecretsManager) ListSecrets() ([]string, error) {
	input := &secretsmanager.ListSecretsInput{}

	var secretNames []string
	paginator := secretsmanager.NewListSecretsPaginator(a.client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(a.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}

		for _, secret := range page.SecretList {
			if secret.Name != nil {
				secretNames = append(secretNames, *secret.Name)
			}
		}
	}

	return secretNames, nil
}

// CreateSecret creates a new secret in AWS Secrets Manager (AWS-specific method)
func (a *AWSSecretsManager) CreateSecret(secretName, secretValue, description string) error {
	input := &secretsmanager.CreateSecretInput{
		Name:         aws.String(secretName),
		SecretString: aws.String(secretValue),
	}

	if description != "" {
		input.Description = aws.String(description)
	}

	_, err := a.client.CreateSecret(a.ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create secret '%s': %w", secretName, err)
	}

	return nil
}

// UpdateSecret updates an existing secret in AWS Secrets Manager (AWS-specific method)
func (a *AWSSecretsManager) UpdateSecret(secretName, secretValue string) error {
	input := &secretsmanager.UpdateSecretInput{
		SecretId:     aws.String(secretName),
		SecretString: aws.String(secretValue),
	}

	_, err := a.client.UpdateSecret(a.ctx, input)
	if err != nil {
		return fmt.Errorf("failed to update secret '%s': %w", secretName, err)
	}

	return nil
}

// DeleteSecret deletes a secret from AWS Secrets Manager (AWS-specific method)
func (a *AWSSecretsManager) DeleteSecret(secretName string, forceDelete bool) error {
	input := &secretsmanager.DeleteSecretInput{
		SecretId: aws.String(secretName),
	}

	if forceDelete {
		input.ForceDeleteWithoutRecovery = aws.Bool(true)
	}

	_, err := a.client.DeleteSecret(a.ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete secret '%s': %w", secretName, err)
	}

	return nil
}

// PutParameter creates or overwrites an SSM parameter.
func (a *AWSSecretsManager) PutParameter(projectID, paramName, paramValue string) error {
	input := &ssm.PutParameterInput{
		Name:      aws.String(paramName),
		Value:     aws.String(paramValue),
		Overwrite: aws.Bool(true),
	}
	_, err := a.ssmClient.PutParameter(a.ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put parameter '%s': %w", paramName, err)
	}
	return nil
}

// DeleteParameter deletes an SSM parameter.
func (a *AWSSecretsManager) DeleteParameter(projectID, paramName string) error {
	_, err := a.ssmClient.DeleteParameter(a.ctx, &ssm.DeleteParameterInput{
		Name: aws.String(paramName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete parameter '%s': %w", paramName, err)
	}
	return nil
}
