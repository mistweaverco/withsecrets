package tui

import (
	"fmt"
	"strings"

	"github.com/mistweaverco/withsecrets/internal/config"
	"github.com/mistweaverco/withsecrets/internal/lib/secrets"
)

type createInput struct {
	envVar      string
	secretKey   string
	value       string
	desc        string
	replication string
	locations   []string
	provider    string
	project     string
	envName     string
	configPath  string
}

func (m *Model) snapshotCreateInput() createInput {
	return createInput{
		envVar:      strings.TrimSpace(m.createEnvVar),
		secretKey:   strings.TrimSpace(m.createSecretKey),
		value:       m.createValue,
		desc:        strings.TrimSpace(m.createDesc),
		replication: m.createReplication,
		locations:   append([]string(nil), m.createLocations...),
		provider:    m.selectedEnv.Provider,
		project:     m.selectedEnv.Project,
		envName:     m.selectedEnvName,
		configPath:  m.configPath,
	}
}

func (m *Model) ensureGCPLocationsLoaded() error {
	if m.selectedEnv == nil || m.selectedEnv.Provider != "gcp" {
		return nil
	}
	if len(m.gcpLocationsAll) > 0 {
		return nil
	}

	factory := secrets.NewSecretManagerFactory()
	sm, err := factory.CreateSecretManager(m.ctx, "gcp", m.selectedEnv.Project)
	if err != nil {
		return err
	}
	defer sm.Close()

	gcpSM, ok := sm.(*secrets.GCPSecretManager)
	if !ok {
		return fmt.Errorf("unexpected gcp secret manager type")
	}

	locs, err := gcpSM.SupportedLocations(m.selectedEnv.Project)
	if err != nil {
		return err
	}
	m.gcpLocationsAll = locs
	// By default, show all locations (unless defaults later narrow it).
	m.gcpLocations = append([]string(nil), locs...)
	return nil
}

func (m *Model) doCreateFromForm(in createInput) error {
	envVar := strings.TrimSpace(in.envVar)
	secretKey := strings.TrimSpace(in.secretKey)
	val := in.value
	desc := strings.TrimSpace(in.desc)

	if envVar == "" || secretKey == "" {
		return fmt.Errorf("env var and secret key are required")
	}

	// Create secret in provider for this environment.
	provider := in.provider
	project := in.project

	factory := secrets.NewSecretManagerFactory()
	sm, err := factory.CreateSecretManager(m.ctx, provider, project)
	if err != nil {
		return err
	}
	defer sm.Close()

	mut, err := secrets.AsMutator(sm)
	if err != nil {
		return err
	}

	// If GCP, optionally set per-create locations for replication on the manager instance.
	if provider == "gcp" {
		if gcpSM, ok := sm.(*secrets.GCPSecretManager); ok {
			if in.replication == "user-managed" && len(in.locations) > 0 {
				gcpSM.SetCreateLocations(in.locations)
			} else {
				gcpSM.SetCreateLocations(nil)
			}
		}
	}

	if err := mut.CreateSecret(secretKey, val, desc); err != nil {
		return err
	}

	// Add mapping to ws.yaml.
	if err := config.AddOrUpdateEnvSecretKeyMapping(in.configPath, in.envName, envVar, secretKey); err != nil {
		return err
	}

	return nil
}
