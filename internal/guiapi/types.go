package guiapi

// EnvironmentSummary describes a configured environment.
type EnvironmentSummary struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Project  string `json:"project"`
}

// SecretRow is a single secret mapping with resolved plain and masked values.
type SecretRow struct {
	EnvVar      string `json:"envVar"`
	Value       string `json:"value"`
	MaskedValue string `json:"maskedValue"`
	RefKind     string `json:"refKind"` // secret-key | secret-path | value
	Ref         string `json:"ref"`
	Provider    string `json:"provider"`
	Project     string `json:"project"`
}

// CreateInput holds fields for creating a secret and ws.yaml mapping.
type CreateInput struct {
	ConfigPath  string
	EnvName     string
	EnvVar      string
	SecretKey   string
	Value       string
	Description string
	Replication string   // "global" | "user-managed"
	Locations   []string // GCP user-managed locations
}

// CreateOptions holds pre-filled values for the create form.
type CreateOptions struct {
	Provider            string   `json:"provider"`
	Replication         string   `json:"replication"` // "global" | "user-managed"
	Locations           []string `json:"locations"`   // filtered locations for multiselect
	AllLocations        []string `json:"allLocations,omitempty"`
	SelectedLocations   []string `json:"selectedLocations,omitempty"`
	SupportsReplication bool     `json:"supportsReplication"`
}
