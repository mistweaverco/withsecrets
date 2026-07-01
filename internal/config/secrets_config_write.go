package config

import (
	"bytes"
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

const schemaHeader = "# yaml-language-server: $schema=https://withsecrets.com/ws.schema.json\n---\n"

// AddOrUpdateEnvSecretKeyMapping adds/updates an env mapping in ws.yaml for the
// given environment and environment variable name.
func AddOrUpdateEnvSecretKeyMapping(configPath, envName, envVar, secretKey string) error {
	if configPath == "" {
		return fmt.Errorf("configPath is required")
	}
	if envName == "" {
		envName = "default"
	}
	if envVar == "" {
		return fmt.Errorf("envVar is required")
	}
	if secretKey == "" {
		return fmt.Errorf("secretKey is required")
	}

	return editSecretsYAML(configPath, func(root *yaml.Node) error {
		envNode, err := ensureMapKeyMap(root, envName)
		if err != nil {
			return err
		}
		envMap, err := ensureMapKeyMap(envNode, "env")
		if err != nil {
			return err
		}

		// Replace env var mapping with: {secret-key: "<secretKey>"}
		valueNode := &yaml.Node{
			Kind: yaml.MappingNode,
			Tag:  "!!map",
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "secret-key"},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: secretKey},
			},
		}
		setMapKey(envMap, envVar, valueNode)
		return nil
	})
}

// RemoveEnvMapping removes a single env mapping (by env var name) from the given
// environment in ws.yaml.
func RemoveEnvMapping(configPath, envName, envVar string) error {
	if configPath == "" {
		return fmt.Errorf("configPath is required")
	}
	if envName == "" {
		envName = "default"
	}
	if envVar == "" {
		return fmt.Errorf("envVar is required")
	}

	return editSecretsYAML(configPath, func(root *yaml.Node) error {
		envNode, err := getMapValue(root, envName)
		if err != nil {
			return err
		}
		if envNode == nil {
			return fmt.Errorf("environment '%s' not found in config", envName)
		}
		envMapNode, err := getMapValue(envNode, "env")
		if err != nil {
			return err
		}
		if envMapNode == nil {
			return nil
		}
		deleteMapKey(envMapNode, envVar)
		return nil
	})
}

func editSecretsYAML(configPath string, edit func(root *yaml.Node) error) error {
	original, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	indent := detectIndent(original)

	dec := yaml.NewDecoder(bytes.NewReader(original))
	dec.KnownFields(false)

	var doc yaml.Node
	if err := dec.Decode(&doc); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return fmt.Errorf("invalid yaml document")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("expected top-level mapping")
	}

	if err := edit(root); err != nil {
		return err
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(indent)
	if err := enc.Encode(&doc); err != nil {
		_ = enc.Close()
		return fmt.Errorf("failed to encode updated config: %w", err)
	}
	_ = enc.Close()

	out := ensureSchemaHeader(buf.Bytes())

	if err := os.WriteFile(configPath, out, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

func ensureSchemaHeader(out []byte) []byte {
	// Normalize output to always have exactly:
	//   # yaml-language-server: $schema=...
	//   ---
	// before the YAML document.
	trimmed := bytes.TrimLeft(out, "\ufeff \t\r\n")

	// Drop an existing schema header line if it was preserved as a comment.
	if bytes.HasPrefix(trimmed, []byte("# yaml-language-server: $schema=")) {
		if i := bytes.IndexByte(trimmed, '\n'); i >= 0 {
			trimmed = trimmed[i+1:]
		} else {
			trimmed = nil
		}
		trimmed = bytes.TrimLeft(trimmed, " \t\r\n")
	}

	// Drop a leading document start marker if present to avoid duplication.
	trimmed = bytes.TrimPrefix(trimmed, []byte("---\n"))

	return append([]byte(schemaHeader), trimmed...)
}

func detectIndent(content []byte) int {
	// Try to detect indentation from common fields like "provider:" in env blocks.
	// Default matches `ws init` which uses 2-space indentation.
	s := string(content)
	re := regexp.MustCompile(`(?m)^\s*(\w[\w-]*):\s*\n(\s+)provider:`)
	m := re.FindStringSubmatch(s)
	if len(m) >= 3 {
		spaces := countLeadingSpaces(m[2])
		if spaces >= 2 && spaces <= 8 {
			return spaces
		}
	}
	return 2
}

func countLeadingSpaces(s string) int {
	n := 0
	for _, r := range s {
		if r == ' ' {
			n++
			continue
		}
		break
	}
	return n
}

func ensureMapKeyMap(mapNode *yaml.Node, key string) (*yaml.Node, error) {
	if mapNode == nil || mapNode.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected mapping node")
	}
	val, err := getMapValue(mapNode, key)
	if err != nil {
		return nil, err
	}
	if val != nil {
		if val.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("expected '%s' to be a mapping", key)
		}
		return val, nil
	}
	newMap := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	setMapKey(mapNode, key, newMap)
	return newMap, nil
}

func getMapValue(mapNode *yaml.Node, key string) (*yaml.Node, error) {
	if mapNode == nil || mapNode.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected mapping node")
	}
	for i := 0; i < len(mapNode.Content); i += 2 {
		k := mapNode.Content[i]
		v := mapNode.Content[i+1]
		if k.Kind == yaml.ScalarNode && k.Value == key {
			return v, nil
		}
	}
	return nil, nil
}

func setMapKey(mapNode *yaml.Node, key string, value *yaml.Node) {
	// Update if exists, else append to end.
	for i := 0; i < len(mapNode.Content); i += 2 {
		k := mapNode.Content[i]
		if k.Kind == yaml.ScalarNode && k.Value == key {
			mapNode.Content[i+1] = value
			return
		}
	}
	mapNode.Content = append(mapNode.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		value,
	)
}

func deleteMapKey(mapNode *yaml.Node, key string) {
	if mapNode == nil || mapNode.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i < len(mapNode.Content); i += 2 {
		k := mapNode.Content[i]
		if k.Kind == yaml.ScalarNode && k.Value == key {
			mapNode.Content = append(mapNode.Content[:i], mapNode.Content[i+2:]...)
			return
		}
	}
}
