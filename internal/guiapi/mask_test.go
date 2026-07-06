package guiapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskValue(t *testing.T) {
	assert.Equal(t, "", MaskValue(""))
	assert.Equal(t, "••••", MaskValue("abcd"))
	assert.Equal(t, "••••••••", MaskValue("longer-secret-value"))
}
