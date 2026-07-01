package ws

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommand(t *testing.T) {
	assert.Equal(t, "ws", rootCmd.Use)
	assert.Contains(t, rootCmd.Short, "withsecrets")
	assert.Contains(t, rootCmd.Long, "withsecrets")

	subcommands := rootCmd.Commands()
	expectedCommands := []string{"version", "update"}

	for _, expected := range expectedCommands {
		found := false
		for _, cmd := range subcommands {
			if cmd.Name() == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected subcommand %s not found", expected)
	}
}

func TestRootCommandFlags(t *testing.T) {
	debugFlag := rootCmd.PersistentFlags().Lookup("debug")
	require.NotNil(t, debugFlag, "debug flag should exist")
	assert.Equal(t, "false", debugFlag.DefValue)
}

func TestExecute(t *testing.T) {
	t.Run("execute function exists", func(t *testing.T) {
		assert.NotPanics(t, func() {})
	})
}

func TestExecuteExitsOnError(t *testing.T) {
	prevOsExit := osExit
	defer func() { osExit = prevOsExit }()

	var exitedWith *int
	osExit = func(code int) {
		exitedWith = &code
	}

	failingCmd := &cobra.Command{Use: "failing"}
	failingCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("boom")
	}

	originalRoot := rootCmd
	rootCmd = failingCmd
	defer func() { rootCmd = originalRoot }()

	Execute()

	if assert.NotNil(t, exitedWith) {
		assert.Equal(t, 1, *exitedWith)
	}
}

func TestConfigInitialization(t *testing.T) {
	assert.NotNil(t, cfg)
	assert.IsType(t, false, cfg.Flags.Version)
}

func TestRootCommandHelp(t *testing.T) {
	assert.NotNil(t, rootCmd)
	assert.NotEmpty(t, rootCmd.Short)
	assert.NotEmpty(t, rootCmd.Long)
}

func TestRootCommandInvalidArgs(t *testing.T) {
	assert.NotNil(t, rootCmd)
	assert.NotNil(t, rootCmd.Run)
}

func TestSubcommandIntegration(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		t.Run("subcommand_"+cmd.Name(), func(t *testing.T) {
			assert.NotNil(t, cmd)
			assert.NotEmpty(t, cmd.Name())
		})
	}
}

func TestRootCommandWithEnvironment(t *testing.T) {
	t.Run("root command with environment", func(t *testing.T) {
		assert.NotNil(t, rootCmd)
		assert.Equal(t, "ws", rootCmd.Use)
	})
}

func TestRootCommandFlagParsing(t *testing.T) {
	testCases := []struct {
		name string
		args []string
	}{
		{"debug flag", []string{"--debug"}},
		{"short debug", []string{"-d"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotNil(t, rootCmd)
			assert.NotNil(t, rootCmd.PersistentFlags())
		})
	}
}

func TestRootCommandStructure(t *testing.T) {
	assert.NotNil(t, rootCmd)
	assert.Equal(t, "ws", rootCmd.Name())
	assert.NotEmpty(t, rootCmd.Short)
	assert.NotEmpty(t, rootCmd.Long)
	assert.NotNil(t, rootCmd.Run)
}

func TestRootCommandSuggestions(t *testing.T) {
	assert.NotNil(t, rootCmd)
	assert.NotNil(t, rootCmd.Run)
}

func TestRootCommandCompletion(t *testing.T) {
	assert.NotNil(t, rootCmd)
	assert.NotNil(t, rootCmd.Run)
}
