package guiapi

import "strings"

// MaskValue returns a masked representation of a secret value.
func MaskValue(v string) string {
	if v == "" {
		return ""
	}
	if len(v) <= 4 {
		return strings.Repeat("•", len(v))
	}
	return strings.Repeat("•", 8)
}
