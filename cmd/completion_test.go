package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionCommand(t *testing.T) {
	tests := []struct {
		name     string
		shell    string
		contains string
	}{
		{
			name:     "bash completion",
			shell:    "bash",
			contains: "__start_trs",
		},
		{
			name:     "zsh completion",
			shell:    "zsh",
			contains: "#compdef",
		},
		{
			name:     "fish completion",
			shell:    "fish",
			contains: "complete -c trs",
		},
		{
			name:     "powershell completion",
			shell:    "powershell",
			contains: "Register-ArgumentCompleter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			// Use rootCmd so cmd.Root() returns the actual root
			rootCmd.SetOut(buf)
			rootCmd.SetArgs([]string{"completion", tt.shell})
			err := rootCmd.Execute()
			require.NoError(t, err)

			output := buf.String()
			assert.Contains(t, output, tt.contains)
		})
	}
}

func TestCompletionInvalidShell(t *testing.T) {
	rootCmd.SetArgs([]string{"completion", "invalid"})
	err := rootCmd.Execute()
	assert.Error(t, err)
}

func TestCompletionNoArgs(t *testing.T) {
	rootCmd.SetArgs([]string{"completion"})
	err := rootCmd.Execute()
	assert.Error(t, err)
}

func TestFileCompletion(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		toComplete string
	}{
		{
			name:       "empty args",
			args:       []string{},
			toComplete: "",
		},
		{
			name:       "with existing args",
			args:       []string{"file1.txt"},
			toComplete: "file",
		},
		{
			name:       "partial path",
			args:       []string{},
			toComplete: "/home/user/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions, directive := fileCompletion(rootCmd, tt.args, tt.toComplete)

			// fileCompletion should return empty completions and ShellCompDirectiveDefault
			// which tells the shell to use its built-in file completion
			assert.Nil(t, completions)
			assert.Equal(t, cobra.ShellCompDirectiveDefault, directive)
		})
	}
}

func TestCompletionCommandExists(t *testing.T) {
	// Verify completion command is registered
	completionCmd, _, err := rootCmd.Find([]string{"completion"})
	require.NoError(t, err)
	assert.Equal(t, "completion", completionCmd.Name())
}

func TestCompletionCommandValidArgs(t *testing.T) {
	completionCmd, _, err := rootCmd.Find([]string{"completion"})
	require.NoError(t, err)

	validArgs := completionCmd.ValidArgs
	assert.Contains(t, validArgs, "bash")
	assert.Contains(t, validArgs, "zsh")
	assert.Contains(t, validArgs, "fish")
	assert.Contains(t, validArgs, "powershell")
}

func TestCompletionBashOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion", "bash"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Check for key bash completion elements
	assert.Contains(t, output, "trs", "completion should reference 'trs' command")

	// Check for subcommands in completion
	expectedSubcmds := []string{"list", "restore", "empty", "status", "version"}
	for _, subcmd := range expectedSubcmds {
		assert.Contains(t, output, subcmd, "completion should include '%s' subcommand", subcmd)
	}
}

func TestCompletionZshOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion", "zsh"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// zsh completion should start with #compdef
	assert.Contains(t, output, "#compdef")
}

func TestCompletionFishOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion", "fish"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// fish completion should use 'complete -c trs'
	assert.Contains(t, output, "complete -c trs")
}

func TestCompletionPowershellOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion", "powershell"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// powershell should contain Register-ArgumentCompleter
	assert.Contains(t, output, "Register-ArgumentCompleter")
}

func TestCompletionCommandHelp(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion", "--help"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Check that help contains instructions for all shells
	assert.True(t, strings.Contains(output, "Bash:") ||
		strings.Contains(output, "bash"), "help should mention bash")
	assert.True(t, strings.Contains(output, "Zsh:") ||
		strings.Contains(output, "zsh"), "help should mention zsh")
	assert.True(t, strings.Contains(output, "Fish:") ||
		strings.Contains(output, "fish"), "help should mention fish")
}

