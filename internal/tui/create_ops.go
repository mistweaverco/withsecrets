package tui

import (
	"strings"

	"github.com/mistweaverco/withsecrets/internal/guiapi"
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

	locs, err := guiapi.GCPLocations(m.ctx, m.selectedEnv.Project)
	if err != nil {
		return err
	}
	m.gcpLocationsAll = locs
	m.gcpLocations = append([]string(nil), locs...)
	return nil
}

func (m *Model) doCreateFromForm(in createInput) error {
	return guiapi.CreateSecret(m.ctx, guiapi.CreateInput{
		ConfigPath:  in.configPath,
		EnvName:     in.envName,
		EnvVar:      in.envVar,
		SecretKey:   in.secretKey,
		Value:       in.value,
		Description: in.desc,
		Replication: in.replication,
		Locations:   in.locations,
	})
}
