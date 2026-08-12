package guiapi

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/mistweaverco/withsecrets/internal/config"
	"github.com/mistweaverco/withsecrets/internal/lib/secrets"
)

var errUnexpectedGCPManager = fmt.Errorf("unexpected gcp secret manager type")

// ListEnvironments returns all configured environment names with provider/project.
func ListEnvironments(configPath string) ([]EnvironmentSummary, error) {
	cfg, err := config.LoadSecretsConfig(configPath)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(cfg.Environments))
	for name := range cfg.Environments {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]EnvironmentSummary, 0, len(names))
	for _, name := range names {
		env := cfg.Environments[name]
		out = append(out, EnvironmentSummary{
			Name:     name,
			Provider: env.Provider,
			Project:  env.Project,
		})
	}
	return out, nil
}

// ListSecrets returns secret rows for an environment with plain and masked values.
func ListSecrets(ctx context.Context, configPath, envName string) ([]SecretRow, error) {
	cfg, err := config.LoadSecretsConfig(configPath)
	if err != nil {
		return nil, err
	}
	env, err := cfg.GetEnvironment(envName)
	if err != nil {
		return nil, err
	}

	factory := secrets.NewSecretManagerFactory()
	values, err := factory.GetSecretsForEnvironmentWithCache(ctx, env, configPath, envName)
	if err != nil {
		return nil, err
	}

	items := env.GetEnvItems()
	rows := make([]SecretRow, 0, len(items))
	for _, it := range items {
		provider := it.Provider
		if provider == "" {
			provider = env.Provider
		}
		project := it.Project
		if project == "" {
			project = env.Project
		}
		region := it.Region
		if region == "" {
			region = env.Region
		}

		refKind := "value"
		ref := ""
		if it.SecretKey != "" {
			refKind = "secret-key"
			ref = it.SecretKey
		} else if it.SecretPath != "" {
			refKind = "secret-path"
			ref = it.SecretPath
		} else if it.ParamKey != "" {
			refKind = "param-key"
			ref = it.ParamKey
		} else if it.ParamPath != "" {
			refKind = "param-path"
			ref = it.ParamPath
		}

		val := values[it.EnvironmentVariable]

		// For "*" bulk path mappings, skip the parent "*" row and only surface expanded vars
		if it.EnvironmentVariable != "*" {
			rows = append(rows, SecretRow{
				EnvVar:      it.EnvironmentVariable,
				Value:       val,
				MaskedValue: MaskValue(val),
				RefKind:     refKind,
				Ref:         ref,
				Provider:    provider,
				Project:     project,
				Region:      region,
			})
		}

		// For path / "*" bulk mappings, also surface expanded env vars
		if it.SecretPath != "" || it.ParamPath != "" {
			prefix := ""
			if it.EnvironmentVariable != "*" {
				prefix = it.EnvironmentVariable + "_"
			}
			for k, v := range values {
				if it.EnvironmentVariable == "*" {
					// Skip keys that are themselves configured mappings
					if _, isMapped := env.Env[k]; isMapped {
						continue
					}
					rows = append(rows, SecretRow{
						EnvVar:      k,
						Value:       v,
						MaskedValue: MaskValue(v),
						RefKind:     refKind,
						Ref:         ref,
						Provider:    provider,
						Project:     project,
						Region:      region,
					})
				} else if strings.HasPrefix(k, prefix) && k != it.EnvironmentVariable {
					rows = append(rows, SecretRow{
						EnvVar:      k,
						Value:       v,
						MaskedValue: MaskValue(v),
						RefKind:     refKind,
						Ref:         ref,
						Provider:    provider,
						Project:     project,
						Region:      region,
					})
				}
			}
		}
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].EnvVar < rows[j].EnvVar })
	return rows, nil
}

// GetSecret returns the unmasked value for a single env var.
func GetSecret(ctx context.Context, configPath, envName, envVar string) (string, error) {
	rows, err := ListSecrets(ctx, configPath, envName)
	if err != nil {
		return "", err
	}
	for _, r := range rows {
		if r.EnvVar == envVar {
			return r.Value, nil
		}
	}
	return "", fmt.Errorf("secret '%s' not found in environment '%s'", envVar, envName)
}

// CreateSecret creates a provider secret and adds a ws.yaml mapping.
func CreateSecret(ctx context.Context, in CreateInput) error {
	envVar := strings.TrimSpace(in.EnvVar)
	secretKey := strings.TrimSpace(in.SecretKey)
	if envVar == "" || secretKey == "" {
		return fmt.Errorf("env var and secret key are required")
	}

	cfg, err := config.LoadSecretsConfig(in.ConfigPath)
	if err != nil {
		return err
	}
	env, err := cfg.GetEnvironment(in.EnvName)
	if err != nil {
		return err
	}

	factory := secrets.NewSecretManagerFactory()
	sm, err := factory.CreateSecretManager(ctx, env.Provider, env.Project, env.Region)
	if err != nil {
		return err
	}
	defer sm.Close()

	if env.Provider == "gcp" {
		if gcpSM, ok := sm.(*secrets.GCPSecretManager); ok {
			if in.Replication == "user-managed" && len(in.Locations) > 0 {
				gcpSM.SetCreateLocations(in.Locations)
			} else {
				gcpSM.SetCreateLocations(nil)
			}
		}
	}

	mut, err := secrets.AsMutator(sm)
	if err != nil {
		return err
	}

	desc := strings.TrimSpace(in.Description)
	if err := mut.CreateSecret(secretKey, in.Value, desc); err != nil {
		return err
	}

	return config.AddOrUpdateEnvSecretKeyMapping(in.ConfigPath, in.EnvName, envVar, secretKey)
}

// UpdateSecret updates a provider secret value (secret-key mappings only).
func UpdateSecret(ctx context.Context, configPath, envName, envVar, newValue string) error {
	row, err := findSecretRow(ctx, configPath, envName, envVar)
	if err != nil {
		return err
	}
	if row.RefKind != "secret-key" {
		return fmt.Errorf("edit is only supported for secret-key mappings")
	}

	factory := secrets.NewSecretManagerFactory()
	sm, err := factory.CreateSecretManager(ctx, row.Provider, row.Project, row.Region)
	if err != nil {
		return err
	}
	defer sm.Close()

	mut, err := secrets.AsMutator(sm)
	if err != nil {
		return err
	}
	return mut.UpdateSecret(row.Ref, newValue)
}

// DeleteSecret deletes a provider secret and removes the ws.yaml mapping.
func DeleteSecret(ctx context.Context, configPath, envName, envVar string) error {
	row, err := findSecretRow(ctx, configPath, envName, envVar)
	if err != nil {
		return err
	}
	if row.RefKind != "secret-key" {
		return fmt.Errorf("delete is only supported for secret-key mappings")
	}

	factory := secrets.NewSecretManagerFactory()
	sm, err := factory.CreateSecretManager(ctx, row.Provider, row.Project, row.Region)
	if err != nil {
		return err
	}
	defer sm.Close()

	mut, err := secrets.AsMutator(sm)
	if err != nil {
		return err
	}
	if err := mut.DeleteSecret(row.Ref, true); err != nil {
		return err
	}
	return config.RemoveEnvMapping(configPath, envName, envVar)
}

func findSecretRow(ctx context.Context, configPath, envName, envVar string) (*SecretRow, error) {
	rows, err := ListSecrets(ctx, configPath, envName)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if rows[i].EnvVar == envVar {
			return &rows[i], nil
		}
	}
	return nil, fmt.Errorf("secret '%s' not found in environment '%s'", envVar, envName)
}
