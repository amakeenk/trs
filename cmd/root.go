package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	flagVerbose bool
	flagJSON    bool
	flagForce   bool
)

var rootCmd = &cobra.Command{
	Use:   "trs <file> [file...]",
	Short: "Safe rm replacement - moves files to trash",
	Long: `trs (trash) is a safe replacement for rm that moves files to the 
system trash instead of permanently deleting them.

Files are moved according to the XDG Trash specification:
- $XDG_DATA_HOME/Trash or ~/.local/share/Trash/
- Cross-device files go to $VOLUME/.Trash-$UID/`,
	Run:  runTrash,
	Args: cobra.ArbitraryArgs,
	ValidArgsFunction: fileCompletion,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "machine-readable JSON output")
	rootCmd.PersistentFlags().BoolVarP(&flagForce, "force", "f", false, "ignore nonexistent files")

	// Add subcommands
	rootCmd.AddCommand(NewListCmd())
	rootCmd.AddCommand(NewRestoreCmd())
	rootCmd.AddCommand(NewEmptyCmd())
	rootCmd.AddCommand(NewStatusCmd())
	rootCmd.AddCommand(newCompletionCmd())
}

// fileCompletion enables shell file completion for arguments
func fileCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveDefault
}

// exitWithError prints error and exits with code
func exitWithError(msg string, code int) {
	os.Stderr.WriteString(msg + "\n")
	os.Exit(code)
}

// validateCLIInput validates user input from CLI prompts
// It checks for:
// - Maximum length (4096 characters)
// - Null bytes (security risk)
func validateCLIInput(input string) error {
	if len(input) > 4096 {
		return fmt.Errorf("input too long (max 4096 characters)")
	}
	if strings.ContainsRune(input, '\x00') {
		return fmt.Errorf("input contains null byte")
	}
	return nil
}
