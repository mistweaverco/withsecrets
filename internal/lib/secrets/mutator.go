package secrets

import "fmt"

// AsMutator converts a SecretManager into a SecretMutator when supported.
//
// Not all providers support secret mutation, and some providers expose mutation
// methods that don't match the generic SecretMutator signature. This helper
// normalizes those differences for interactive tooling (e.g. `ws tui`).
func AsMutator(sm SecretManager) (SecretMutator, error) {
	if sm == nil {
		return nil, fmt.Errorf("nil secret manager")
	}

	// Providers that already match SecretMutator.
	if m, ok := sm.(SecretMutator); ok {
		return m, nil
	}

	// Providers needing adapters.
	switch v := sm.(type) {
	case *OpenBaoManager:
		return &openBaoMutator{m: v}, nil
	default:
		return nil, fmt.Errorf("provider does not support secret mutation")
	}
}

type openBaoMutator struct {
	m *OpenBaoManager
}

func (o *openBaoMutator) CreateSecret(secretName, secretValue, description string) error {
	_ = description // OpenBao doesn't have a standard description field
	return o.m.CreateSecret(secretName, map[string]interface{}{"value": secretValue})
}

func (o *openBaoMutator) UpdateSecret(secretName, secretValue string) error {
	return o.m.UpdateSecret(secretName, map[string]interface{}{"value": secretValue})
}

func (o *openBaoMutator) DeleteSecret(secretName string, forceDelete bool) error {
	_ = forceDelete // OpenBao does not have soft delete here
	return o.m.DeleteSecret(secretName)
}
