package ws

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGuiCommandRegistered(t *testing.T) {
	var found bool
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "gui" {
			found = true
			require.Contains(t, cmd.Short, "GUI")
			configFlag := cmd.Flags().Lookup("config")
			require.NotNil(t, configFlag)
			noBrowserFlag := cmd.Flags().Lookup("no-browser")
			require.NotNil(t, noBrowserFlag)
			break
		}
	}
	require.True(t, found, "gui command should be registered")
}
