package cmd

import (
	"os"

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
}

// exitWithError prints error and exits with code
func exitWithError(msg string, code int) {
	os.Stderr.WriteString(msg + "\n")
	os.Exit(code)
}
