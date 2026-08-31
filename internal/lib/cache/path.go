package cache

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// PathMappingVersion is the schema version stored in CachedPathMapping records.
	PathMappingVersion   = 1
	pathMappingKeyPrefix = "ws:path:v1:"
)

// CachedPathEntry is one expanded environment variable from a path mapping.
type CachedPathEntry struct {
	EnvironmentVariable string `json:"environment_variable"`
	SourceRef           string `json:"source_ref"`
	Value               string `json:"value"`
}

// CachedPathMapping is the complete resolved result of one secret-path or param-path mapping.
type CachedPathMapping struct {
	Version int               `json:"version"`
	Entries []CachedPathEntry `json:"entries"`
}

// PathMappingCacheKey returns the reserved secrets.env key for a path mapping identity.
func PathMappingCacheKey(mappingID string) string {
	return pathMappingKeyPrefix + mappingID
}

// IsPathMappingKey reports whether env is a reserved path-mapping cache key.
func IsPathMappingKey(env string) bool {
	return strings.HasPrefix(env, pathMappingKeyPrefix)
}

// SetPathMapping stores a complete path mapping record.
func (c *Cache) SetPathMapping(path, configEnv, mappingID string, mapping CachedPathMapping, ttl time.Duration) error {
	data, err := marshalPathMapping(mapping)
	if err != nil {
		return err
	}
	return c.Set(path, configEnv, PathMappingCacheKey(mappingID), string(data), ttl)
}

// GetPathMapping retrieves a complete path mapping record.
// Missing rows, expired rows, malformed JSON, and unsupported versions are cache misses.
func (c *Cache) GetPathMapping(path, configEnv, mappingID string) (CachedPathMapping, bool, error) {
	value, found, err := c.Get(path, configEnv, PathMappingCacheKey(mappingID))
	if err != nil || !found {
		return CachedPathMapping{}, found, err
	}
	mapping, ok := unmarshalPathMapping(value)
	if !ok {
		return CachedPathMapping{}, false, nil
	}
	return mapping, true, nil
}

// SetPathMapping stores a complete path mapping record.
func (m *Manager) SetPathMapping(configPath, envName, mappingID string, mapping CachedPathMapping, ttl time.Duration) error {
	if !m.IsEnabled() {
		return nil
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	return m.cache.SetPathMapping(absPath, envName, mappingID, mapping, ttl)
}

// GetPathMapping retrieves a complete path mapping record.
func (m *Manager) GetPathMapping(configPath, envName, mappingID string) (CachedPathMapping, bool, error) {
	if !m.IsEnabled() {
		return CachedPathMapping{}, false, nil
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return CachedPathMapping{}, false, fmt.Errorf("failed to get absolute path: %w", err)
	}

	return m.cache.GetPathMapping(absPath, envName, mappingID)
}

// PatchPathMappingEntry updates the value of envVar inside any path-mapping
// blobs for this environment. Missing or expired blobs are skipped.
func (m *Manager) PatchPathMappingEntry(configPath, envName, envVar, newValue string) error {
	if !m.IsEnabled() {
		return nil
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	return m.cache.PatchPathMappingEntry(absPath, envName, envVar, newValue)
}

// RemovePathMappingEntry drops envVar from any path-mapping blobs for this
// environment. Empty blobs are deleted.
func (m *Manager) RemovePathMappingEntry(configPath, envName, envVar string) error {
	if !m.IsEnabled() {
		return nil
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	return m.cache.RemovePathMappingEntry(absPath, envName, envVar)
}

func remainingTTL(expiresAt time.Time) (time.Duration, bool) {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return 0, false
	}
	return ttl, true
}

func pathMappingIDFromEnv(env string) (string, bool) {
	if !IsPathMappingKey(env) {
		return "", false
	}
	return strings.TrimPrefix(env, pathMappingKeyPrefix), true
}

func (c *Cache) forEachPathMapping(path, configEnv string, fn func(mapping CachedPathMapping) (CachedPathMapping, bool)) error {
	entries, err := c.List()
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Path != path || entry.ConfigEnv != configEnv {
			continue
		}
		mappingID, ok := pathMappingIDFromEnv(entry.Env)
		if !ok {
			continue
		}
		ttl, ok := remainingTTL(entry.ExpiresAt)
		if !ok {
			continue
		}
		mapping, ok := unmarshalPathMapping(entry.Value)
		if !ok {
			continue
		}
		updated, changed := fn(mapping)
		if !changed {
			continue
		}
		if len(updated.Entries) == 0 {
			if err := c.Delete(path, configEnv, entry.Env); err != nil {
				return err
			}
			continue
		}
		if err := c.SetPathMapping(path, configEnv, mappingID, updated, ttl); err != nil {
			return err
		}
	}
	return nil
}

// PatchPathMappingEntry updates envVar's value in matching path-mapping blobs.
func (c *Cache) PatchPathMappingEntry(path, configEnv, envVar, newValue string) error {
	return c.forEachPathMapping(path, configEnv, func(mapping CachedPathMapping) (CachedPathMapping, bool) {
		changed := false
		for i := range mapping.Entries {
			if mapping.Entries[i].EnvironmentVariable == envVar {
				mapping.Entries[i].Value = newValue
				changed = true
			}
		}
		return mapping, changed
	})
}

// RemovePathMappingEntry drops envVar from matching path-mapping blobs.
func (c *Cache) RemovePathMappingEntry(path, configEnv, envVar string) error {
	return c.forEachPathMapping(path, configEnv, func(mapping CachedPathMapping) (CachedPathMapping, bool) {
		filtered := make([]CachedPathEntry, 0, len(mapping.Entries))
		changed := false
		for _, item := range mapping.Entries {
			if item.EnvironmentVariable == envVar {
				changed = true
				continue
			}
			filtered = append(filtered, item)
		}
		mapping.Entries = filtered
		return mapping, changed
	})
}

func marshalPathMapping(mapping CachedPathMapping) ([]byte, error) {
	mapping.Version = PathMappingVersion
	if mapping.Entries == nil {
		mapping.Entries = []CachedPathEntry{}
	}
	sort.Slice(mapping.Entries, func(i, j int) bool {
		if mapping.Entries[i].EnvironmentVariable != mapping.Entries[j].EnvironmentVariable {
			return mapping.Entries[i].EnvironmentVariable < mapping.Entries[j].EnvironmentVariable
		}
		return mapping.Entries[i].SourceRef < mapping.Entries[j].SourceRef
	})
	return json.Marshal(mapping)
}

// ExpandListEntries replaces path-mapping cache rows with one entry per
// expanded environment variable so listing and stats match the resolved env.
// Malformed path records are omitted rather than shown as JSON blobs.
func ExpandListEntries(entries []CacheEntry) []CacheEntry {
	out := make([]CacheEntry, 0, len(entries))
	for _, entry := range entries {
		if !IsPathMappingKey(entry.Env) {
			out = append(out, entry)
			continue
		}
		mapping, ok := unmarshalPathMapping(entry.Value)
		if !ok {
			continue
		}
		for _, item := range mapping.Entries {
			out = append(out, CacheEntry{
				Path:      entry.Path,
				ConfigEnv: entry.ConfigEnv,
				Env:       item.EnvironmentVariable,
				Value:     item.Value,
				CreatedAt: entry.CreatedAt,
				ExpiresAt: entry.ExpiresAt,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		if out[i].ConfigEnv != out[j].ConfigEnv {
			return out[i].ConfigEnv < out[j].ConfigEnv
		}
		return out[i].Env < out[j].Env
	})
	return out
}

func unmarshalPathMapping(value string) (CachedPathMapping, bool) {
	var mapping CachedPathMapping
	if err := json.Unmarshal([]byte(value), &mapping); err != nil {
		return CachedPathMapping{}, false
	}
	if mapping.Version != PathMappingVersion {
		return CachedPathMapping{}, false
	}
	if mapping.Entries == nil {
		mapping.Entries = []CachedPathEntry{}
	}
	return mapping, true
}
