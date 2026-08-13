package secrets

import (
	"context"
	"fmt"
	"sort"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
)

// GCPSecretManager handles GCP Secret Manager operations
type GCPSecretManager struct {
	client          *secretmanager.Client
	ctx             context.Context
	projectID       string
	createLocations []string
}

// NewGCPSecretManager creates a new GCP Secret Manager client
func NewGCPSecretManager(ctx context.Context, credentialsFile string, projectID string) (*GCPSecretManager, error) {
	var opts []option.ClientOption

	if credentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsFile))
	}

	client, err := secretmanager.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret manager client: %w", err)
	}

	return &GCPSecretManager{
		client:    client,
		ctx:       ctx,
		projectID: projectID,
	}, nil
}

// SetCreateLocations configures user-managed replication locations to use for
// subsequent CreateSecret calls on this manager instance. If empty/nil, CreateSecret
// uses automatic (global) replication.
func (g *GCPSecretManager) SetCreateLocations(locs []string) {
	if len(locs) == 0 {
		g.createLocations = nil
		return
	}
	cp := append([]string(nil), locs...)
	sort.Strings(cp)
	g.createLocations = cp
}

// GetSecret retrieves a secret from GCP Secret Manager
func (g *GCPSecretManager) GetSecret(projectID, secretID string) (string, error) {
	// Build the resource name
	name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretID)

	// Access the secret version
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}

	result, err := g.client.AccessSecretVersion(g.ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to access secret version: %w", err)
	}

	return string(result.Payload.Data), nil
}

// Close closes the GCP Secret Manager client
func (g *GCPSecretManager) Close() error {
	return g.client.Close()
}

// GetSecrets retrieves multiple secrets from GCP Secret Manager
func (g *GCPSecretManager) GetSecrets(projectID string, secretIDs []string) (map[string]string, error) {
	secrets := make(map[string]string)

	for _, secretID := range secretIDs {
		secret, err := g.GetSecret(projectID, secretID)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret '%s': %w", secretID, err)
		}
		secrets[secretID] = secret
	}

	return secrets, nil
}

// GetSecretsByPath retrieves all secrets that start with the given path prefix
func (g *GCPSecretManager) GetSecretsByPath(projectID, secretPath string) (map[string]string, error) {
	secrets := make(map[string]string)

	// List all secrets in the project
	req := &secretmanagerpb.ListSecretsRequest{
		Parent: fmt.Sprintf("projects/%s", projectID),
	}

	it := g.client.ListSecrets(g.ctx, req)
	for {
		secret, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate secrets: %w", err)
		}

		// Check if the secret name starts with the path prefix
		secretName := secret.Name
		if strings.HasPrefix(secretName, secretPath) {
			// Extract just the secret ID from the full path
			secretID := extractSecretNameFromPath(secretName)

			// Get the actual secret value
			secretValue, err := g.GetSecret(projectID, secretID)
			if err != nil {
				// Log warning but continue with other secrets
				fmt.Printf("Warning: failed to get secret '%s': %v\n", secretID, err)
				continue
			}

			// Sanitize the secret ID for use as an environment variable name
			secrets[secretID] = secretValue
		}
	}

	return secrets, nil
}

// SupportedLocations returns Secret Manager-supported locations for the project.
// Note: this does not account for org-policy constraints; it's just the service
// location catalog for the project.
func (g *GCPSecretManager) SupportedLocations(projectID string) ([]string, error) {
	if projectID == "" {
		projectID = g.projectID
	}
	if projectID == "" {
		return nil, fmt.Errorf("gcp projectID is required")
	}

	name := fmt.Sprintf("projects/%s", projectID)
	it := g.client.ListLocations(g.ctx, &locationpb.ListLocationsRequest{Name: name})
	locs := make([]string, 0, 32)
	for {
		loc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list locations: %w", err)
		}
		if loc == nil || loc.LocationId == "" {
			continue
		}
		locs = append(locs, loc.LocationId)
	}

	sort.Strings(locs)
	return locs, nil
}

// CreateSecret creates a new secret and adds an initial version.
func (g *GCPSecretManager) CreateSecret(secretName, secretValue, description string) error {
	if g.projectID == "" {
		return fmt.Errorf("gcp projectID is required")
	}

	replication, err := g.replicationForCreate()
	if err != nil {
		return err
	}

	createReq := &secretmanagerpb.CreateSecretRequest{
		Parent:   fmt.Sprintf("projects/%s", g.projectID),
		SecretId: secretName,
		Secret: &secretmanagerpb.Secret{
			Replication: replication,
		},
	}
	if description != "" {
		if createReq.Secret.Labels == nil {
			createReq.Secret.Labels = map[string]string{}
		}
		createReq.Secret.Labels["description"] = description
	}

	_, err = g.client.CreateSecret(g.ctx, createReq)
	if err != nil {
		return fmt.Errorf("failed to create secret '%s': %w", secretName, err)
	}

	return g.UpdateSecret(secretName, secretValue)
}

func (g *GCPSecretManager) replicationForCreate() (*secretmanagerpb.Replication, error) {
	if len(g.createLocations) == 0 {
		return &secretmanagerpb.Replication{
			Replication: &secretmanagerpb.Replication_Automatic_{
				Automatic: &secretmanagerpb.Replication_Automatic{},
			},
		}, nil
	}

	replicas := make([]*secretmanagerpb.Replication_UserManaged_Replica, 0, len(g.createLocations))
	for _, loc := range g.createLocations {
		if strings.TrimSpace(loc) == "" {
			continue
		}
		replicas = append(replicas, &secretmanagerpb.Replication_UserManaged_Replica{
			Location: loc,
		})
	}
	if len(replicas) == 0 {
		return nil, fmt.Errorf("no gcp replication locations provided")
	}

	return &secretmanagerpb.Replication{
		Replication: &secretmanagerpb.Replication_UserManaged_{
			UserManaged: &secretmanagerpb.Replication_UserManaged{Replicas: replicas},
		},
	}, nil
}

// UpdateSecret adds a new secret version (GCP does not update in-place).
func (g *GCPSecretManager) UpdateSecret(secretName, secretValue string) error {
	if g.projectID == "" {
		return fmt.Errorf("gcp projectID is required")
	}

	addReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: fmt.Sprintf("projects/%s/secrets/%s", g.projectID, secretName),
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(secretValue),
		},
	}
	_, err := g.client.AddSecretVersion(g.ctx, addReq)
	if err != nil {
		return fmt.Errorf("failed to add secret version for '%s': %w", secretName, err)
	}
	return nil
}

// DeleteSecret deletes a secret by name.
func (g *GCPSecretManager) DeleteSecret(secretName string, forceDelete bool) error {
	_ = forceDelete // GCP delete is immediate; no "force" flag needed here.
	if g.projectID == "" {
		return fmt.Errorf("gcp projectID is required")
	}
	req := &secretmanagerpb.DeleteSecretRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s", g.projectID, secretName),
		Etag: "",
	}
	err := g.client.DeleteSecret(g.ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete secret '%s': %w", secretName, err)
	}
	return nil
}
