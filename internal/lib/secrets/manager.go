package secrets

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mistweaverco/withsecrets/internal/config"
	"github.com/mistweaverco/withsecrets/internal/lib/cache"
	"github.com/mistweaverco/withsecrets/internal/lib/log"
)

// SecretManager defines the interface for secret management operations
type SecretManager interface {
	GetSecret(projectID, secretID string) (string, error)
	GetSecrets(projectID string, secretIDs []string) (map[string]string, error)
	GetSecretsByPath(projectID, secretPath string) (map[string]string, error)
	Close() error
}

// SecretMutator is an optional interface implemented by providers that support
// creating/updating/deleting secrets (used by interactive tooling like the TUI).
type SecretMutator interface {
	CreateSecret(secretName, secretValue, description string) error
	UpdateSecret(secretName, secretValue string) error
	DeleteSecret(secretName string, forceDelete bool) error
}

// SecretManagerFactory creates secret managers for different cloud providers
type SecretManagerFactory struct{}

// NewSecretManagerFactory creates a new secret manager factory
func NewSecretManagerFactory() *SecretManagerFactory {
	return &SecretManagerFactory{}
}

// CreateSecretManager creates a secret manager for the specified provider
func (f *SecretManagerFactory) CreateSecretManager(ctx context.Context, provider string, projectID string) (SecretManager, error) {
	switch provider {
	case "gcp":
		// Check for GCP credentials
		credentialsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		return NewGCPSecretManager(ctx, credentialsFile, projectID)
	case "aws":
		// Check for AWS region and profile
		region := os.Getenv("AWS_REGION")
		profile := os.Getenv("AWS_PROFILE")
		return NewAWSSecretsManager(ctx, region, profile)
	case "azure":
		// Check for Azure Key Vault configuration
		vaultURL := os.Getenv("AZURE_KEY_VAULT_URL")
		if vaultURL == "" {
			return nil, fmt.Errorf("AZURE_KEY_VAULT_URL environment variable is required for Azure Key Vault")
		}

		// Optional: tenant ID, client ID, and client secret for service principal auth
		tenantID := os.Getenv("AZURE_TENANT_ID")
		clientID := os.Getenv("AZURE_CLIENT_ID")
		clientSecret := os.Getenv("AZURE_CLIENT_SECRET")

		return NewAzureKeyVaultManager(ctx, vaultURL, tenantID, clientID, clientSecret)
	case "openbao":
		// Check for OpenBao configuration
		address := os.Getenv("OPENBAO_ADDR")
		if address == "" {
			return nil, fmt.Errorf("OPENBAO_ADDR environment variable is required for OpenBao")
		}

		// Optional: token and namespace
		token := os.Getenv("OPENBAO_TOKEN")
		namespace := os.Getenv("OPENBAO_NAMESPACE")

		return NewOpenBaoManager(ctx, address, token, namespace)
	case "local":
		// Local provider doesn't require any external configuration
		return NewLocalManager(ctx)
	case "bitwarden":
		// Bitwarden uses its own organization concept; projectID is treated
		// as the Bitwarden organization ID (or falls back to env vars).
		return NewBitwardenManager(ctx, projectID)
	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", provider)
	}
}

// GetSecretsForEnvironment retrieves all secrets and values for a given environment configuration
func (f *SecretManagerFactory) GetSecretsForEnvironment(ctx context.Context, env *config.Environment) (map[string]string, error) {
	return f.GetSecretsForEnvironmentWithCache(ctx, env, "", "")
}

// GetSecretsForEnvironmentWithCache retrieves all secrets and values for a given environment configuration with caching
func (f *SecretManagerFactory) GetSecretsForEnvironmentWithCache(ctx context.Context, env *config.Environment, configPath, envName string) (map[string]string, error) {
	logger := log.NewLogger()

	// Initialize cache manager if config path is provided
	var cacheManager *cache.Manager
	var cacheEnabled bool
	var cacheTTL time.Duration

	if configPath != "" {
		// Load global config
		globalConfig, err := config.LoadGlobalConfig()
		if err != nil {
			logger.Debug("Failed to load global config, using defaults", "error", err)
			globalConfig = config.DefaultGlobalConfig()
		}

		// Check if caching should be enabled (global or environment level)
		shouldEnableCache := globalConfig.Cache.Enabled
		if env.Cache != nil {
			shouldEnableCache = env.Cache.Enabled
		}

		// Only create cache manager if caching is enabled
		if shouldEnableCache {
			// Convert to cache types
			cacheGlobalConfig := &cache.GlobalConfig{
				Cache: cache.CacheConfig{
					Enabled: globalConfig.Cache.Enabled,
					TTL:     globalConfig.Cache.TTL,
				},
			}

			cacheManager, err = cache.NewManager(cacheGlobalConfig)
			if err != nil {
				logger.Debug("Failed to initialize cache manager", "error", err)
			} else {
				// Convert env cache config
				var envCache *cache.CacheConfig
				if env.Cache != nil {
					envCache = &cache.CacheConfig{
						Enabled: env.Cache.Enabled,
						TTL:     env.Cache.TTL,
					}
				}

				cacheEnabled, cacheTTL = cacheManager.GetCacheConfig(envCache)
				logger.Debug("Cache configuration", "enabled", cacheEnabled, "ttl", cacheTTL)
			}
		} else {
			logger.Debug("Caching disabled", "global_enabled", globalConfig.Cache.Enabled, "env_cache", env.Cache != nil)
		}
	}

	// Try to retrieve all secrets from cache first
	if cacheManager != nil && cacheEnabled && configPath != "" && envName != "" {
		logger.Debug("Attempting to retrieve secrets from cache", "config_path", configPath, "env_name", envName)

		// Get all env items to know what to look for
		envItems := env.GetEnvItems()
		cachedSecrets := make(map[string]string)
		allCached := true

		for _, envItem := range envItems {
			// Skip value-based mappings as they don't need caching
			if envItem.Value != nil {
				continue
			}

			// Try to get from cache
			if value, found, err := cacheManager.Get(configPath, envName, envItem.EnvironmentVariable); err != nil {
				logger.Debug("Failed to get secret from cache", "env_var", envItem.EnvironmentVariable, "error", err)
				allCached = false
				break
			} else if found {
				cachedSecrets[envItem.EnvironmentVariable] = value
				logger.Debug("Retrieved secret from cache", "env_var", envItem.EnvironmentVariable)
			} else {
				logger.Debug("Secret not found in cache", "env_var", envItem.EnvironmentVariable)
				allCached = false
				break
			}
		}

		// If all secrets were found in cache, combine with static values
		if allCached && len(cachedSecrets) > 0 {
			logger.Debug("All secrets retrieved from cache", "count", len(cachedSecrets))

			// Combine cached secrets with static values
			allSecrets := make(map[string]string)

			// Add cached secrets
			for envVar, value := range cachedSecrets {
				allSecrets[envVar] = value
			}

			// Add static values
			for _, envItem := range envItems {
				if envItem.Value != nil {
					allSecrets[envItem.EnvironmentVariable] = fmt.Sprintf("%v", envItem.Value)
				}
			}

			// Interpolate all values
			for key, value := range allSecrets {
				if strings.Contains(value, "${") {
					interpolatedValue := config.InterpolateEnvVars(value, allSecrets)
					allSecrets[key] = interpolatedValue
				}
			}

			// Clean up cache manager
			cacheManager.Close()

			return allSecrets, nil
		}

		logger.Debug("Not all secrets found in cache, fetching from providers", "cached_count", len(cachedSecrets))
	}

	// Group mappings by provider and project for secret-based mappings
	providerGroups := make(map[string]map[string][]string)

	// Group mappings by provider and project for path-based mappings
	pathGroups := make(map[string]map[string]string)

	// Get all env items (from map)
	envItems := env.GetEnvItems()
	logger.Debug("Processing environment mappings", "total_mappings", len(envItems))

	// Process all env items to separate secret-based and value-based ones
	for i, envItem := range envItems {
		logger.Debug("Processing mapping", "index", i, "env_var", envItem.EnvironmentVariable, "has_secret_key", envItem.SecretKey != "", "has_secret_path", envItem.SecretPath != "", "has_value", envItem.Value != nil)

		// Handle direct values first
		if envItem.Value != nil {
			logger.Debug("Skipping secret processing for value-based mapping", "env_var", envItem.EnvironmentVariable)
			continue // Skip secret processing for value-based mappings
		}

		// Process secret-based mappings (single key)
		if envItem.SecretKey != "" {
			provider := envItem.Provider
			if provider == "" {
				provider = env.Provider
			}

			project := envItem.Project
			if project == "" {
				project = env.Project
			}

			// For AWS, Azure, OpenBao, Bitwarden, and local, we use a default project key since they don't use projects in the same way as GCP
			if (provider == "aws" || provider == "azure" || provider == "openbao" || provider == "bitwarden" || provider == "local") && project == "" {
				project = "default"
			}

			logger.Debug("Adding secret-based mapping to provider group", "provider", provider, "project", project, "secret_key", envItem.SecretKey)

			if providerGroups[provider] == nil {
				providerGroups[provider] = make(map[string][]string)
			}

			providerGroups[provider][project] = append(providerGroups[provider][project], envItem.SecretKey)
		}

		// Process path-based mappings
		if envItem.SecretPath != "" {
			provider := envItem.Provider
			if provider == "" {
				provider = env.Provider
			}

			project := envItem.Project
			if project == "" {
				project = env.Project
			}

			// For AWS, Azure, OpenBao, Bitwarden, and local, we use a default project key since they don't use projects in the same way as GCP
			if (provider == "aws" || provider == "azure" || provider == "openbao" || provider == "bitwarden" || provider == "local") && project == "" {
				project = "default"
			}

			logger.Debug("Adding path-based mapping to provider group", "provider", provider, "project", project, "secret_path", envItem.SecretPath)

			// Create a separate group for path-based lookups
			pathKey := fmt.Sprintf("%s:%s", provider, project)
			if pathGroups[pathKey] == nil {
				pathGroups[pathKey] = make(map[string]string)
			}
			pathGroups[pathKey][envItem.EnvironmentVariable] = envItem.SecretPath
		}
	}

	logger.Debug("Provider groups created", "secret_providers", len(providerGroups), "path_providers", len(pathGroups))

	// Fetch secrets from each provider
	allSecrets := make(map[string]string)

	for provider, projects := range providerGroups {
		for project, secretIDs := range projects {
			logger.Debug("Creating secret manager", "provider", provider, "project", project, "secret_count", len(secretIDs))

			secretManager, err := f.CreateSecretManager(ctx, provider, project)
			if err != nil {
				logger.Debug("Failed to create secret manager", "provider", provider, "project", project, "error", err)
				// Log warning but continue with other providers
				fmt.Printf("Warning: failed to create secret manager for %s: %v\n", provider, err)
				continue
			}
			defer secretManager.Close()

			logger.Debug("Fetching secrets from provider", "provider", provider, "project", project, "secret_ids", secretIDs)
			secrets, err := secretManager.GetSecrets(project, secretIDs)
			if err != nil {
				logger.Debug("Failed to get secrets from provider", "provider", provider, "project", project, "error", err)
				// Log warning but continue with other providers
				fmt.Printf("Warning: failed to get secrets from %s project %s: %v\n", provider, project, err)
				continue
			}

			logger.Debug("Successfully retrieved secrets from provider", "provider", provider, "project", project, "retrieved_count", len(secrets))

			// Map secrets to environment variables
			for _, envItem := range envItems {
				if envItem.SecretKey != "" {
					envItemProvider := envItem.Provider
					if envItemProvider == "" {
						envItemProvider = env.Provider
					}

					envItemProject := envItem.Project
					if envItemProject == "" {
						envItemProject = env.Project
					}

					// For AWS, Azure, OpenBao, Bitwarden, and local, we use a default project key since they don't use projects in the same way as GCP
					if (envItemProvider == "aws" || envItemProvider == "azure" || envItemProvider == "openbao" || envItemProvider == "bitwarden" || envItemProvider == "local") && envItemProject == "" {
						envItemProject = "default"
					}

					// Only process mappings that match the current provider and project
					if envItemProvider == provider && envItemProject == project {
						if secretValue, exists := secrets[envItem.SecretKey]; exists {
							allSecrets[envItem.EnvironmentVariable] = secretValue
							logger.Debug("Mapped secret to environment variable", "env_var", envItem.EnvironmentVariable, "secret_key", envItem.SecretKey, "provider", provider, "project", project)
						} else {
							logger.Debug("Secret key not found in provider response", "env_var", envItem.EnvironmentVariable, "secret_key", envItem.SecretKey, "provider", provider, "project", project)
						}
					}
				}
			}
		}
	}

	// Process path-based mappings
	for pathKey, pathMappings := range pathGroups {
		// Parse the path key to get provider and project
		parts := strings.Split(pathKey, ":")
		if len(parts) != 2 {
			fmt.Printf("Warning: invalid path key format: %s\n", pathKey)
			continue
		}

		provider := parts[0]
		project := parts[1]

		secretManager, err := f.CreateSecretManager(ctx, provider, project)
		if err != nil {
			// Log warning but continue with other providers
			fmt.Printf("Warning: failed to create secret manager for %s: %v\n", provider, err)
			continue
		}
		defer secretManager.Close()

		// Process each path mapping
		for envVar, secretPath := range pathMappings {
			secrets, err := secretManager.GetSecretsByPath(project, secretPath)
			if err != nil {
				// Log warning but continue with other paths
				fmt.Printf("Warning: failed to get secrets from path '%s': %v\n", secretPath, err)
				continue
			}

			// Add all secrets from this path to the result
			// The environment variable name from the mapping is used as a prefix
			for secretName, secretValue := range secrets {
				// Create a unique environment variable name by combining the mapping's env var and the secret name
				finalEnvVarName := envVar + "_" + secretName
				allSecrets[finalEnvVarName] = secretValue
			}
		}
	}

	// Process value-based mappings (no bare items allowed anymore)
	for _, envItem := range envItems {
		if envItem.Value != nil {
			// Convert value to string
			var strValue string
			switch v := envItem.Value.(type) {
			case string:
				strValue = v
			case int, int32, int64:
				strValue = fmt.Sprintf("%d", v)
			case float32, float64:
				strValue = fmt.Sprintf("%g", v)
			default:
				strValue = fmt.Sprintf("%v", v)
			}
			allSecrets[envItem.EnvironmentVariable] = strValue
		}
	}

	// Perform interpolation on all values now that we have all secrets and values
	// This allows values to reference other environment variables that were just resolved
	for key, value := range allSecrets {
		if strings.Contains(value, "${") {
			interpolatedValue := config.InterpolateEnvVars(value, allSecrets)
			allSecrets[key] = interpolatedValue
		}
	}

	// Cache the results if caching is enabled (only cache secrets, not static values)
	if cacheManager != nil && cacheEnabled && configPath != "" && envName != "" {
		cachedCount := 0
		for _, envItem := range envItems {
			// Only cache secrets (not static values)
			if envItem.Value == nil && (envItem.SecretKey != "" || envItem.SecretPath != "") {
				envVar := envItem.EnvironmentVariable
				if value, exists := allSecrets[envVar]; exists {
					if err := cacheManager.Set(configPath, envName, envVar, value, cacheTTL); err != nil {
						logger.Debug("Failed to cache secret", "env_var", envVar, "error", err)
					} else {
						cachedCount++
					}
				}
			}
		}
		logger.Debug("Cached secrets", "count", cachedCount, "ttl", cacheTTL)
	}

	// Clean up cache manager
	if cacheManager != nil {
		cacheManager.Close()
	}

	return allSecrets, nil
}
