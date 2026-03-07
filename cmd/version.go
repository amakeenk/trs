package cmd

import (
	"encoding/json"
	"fmt"

	"altlinux.space/amakeenk/trs/internal/version"
	"github.com/spf13/cobra"
)

// NewVersionCmd creates the version command
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Print version, git commit, build date, and platform information",
		Run: func(cmd *cobra.Command, args []string) {
			if flagJSON {
				output, _ := json.MarshalIndent(version.Get(), "", "  ")
				fmt.Println(string(output))
			} else {
				fmt.Println(version.String())
			}
		},
	}
}

func init() {
	rootCmd.AddCommand(NewVersionCmd())
}
