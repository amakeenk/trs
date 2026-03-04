package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/amakeenk/trs/internal/trash"
	"github.com/amakeenk/trs/internal/ui"
	"github.com/spf13/cobra"
)

var (
	flagDays int
)

// NewEmptyCmd creates the empty command
func NewEmptyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "empty [--days N]",
		Short: "Empty the trash",
		Long: `Empty the trash.

Without arguments, clears all files.
With --days N, only clears files older than N days.`,
		Run: runEmpty,
	}

	cmd.Flags().IntVar(&flagDays, "days", 0, "only remove files older than N days")

	return cmd
}

// EmptyResult for JSON output
type EmptyResult struct {
	Removed   int    `json:"removed"`
	Remaining int    `json:"remaining"`
	Days      int    `json:"days,omitempty"`
	Message   string `json:"message,omitempty"`
}

func runEmpty(cmd *cobra.Command, args []string) {
	mgr, err := trash.NewManager()
	if err != nil {
		exitWithError(fmt.Sprintf("Error: %v", err), 1)
	}

	// Get current count
	items, err := mgr.List()
	if err != nil {
		exitWithError(fmt.Sprintf("Error: %v", err), 1)
	}

	beforeCount := len(items)

	if beforeCount == 0 {
		if flagJSON {
			output, _ := json.Marshal(EmptyResult{Message: "Trash is empty"})
			fmt.Println(string(output))
		} else {
			fmt.Println("Trash is empty")
		}
		return
	}

	// Confirm unless --force or --json
	if !flagForce && !flagJSON {
		msg := "Empty trash?"
		if flagDays > 0 {
			msg = fmt.Sprintf("Remove files older than %d days?", flagDays)
		}
		fmt.Printf("%s [y/N]: ", msg)

		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled")
			return
		}
	}

	// Perform empty
	if flagDays > 0 {
		err = mgr.EmptyOlderThan(flagDays)
	} else {
		err = mgr.Empty()
	}

	if err != nil {
		if flagJSON {
			output, _ := json.Marshal(EmptyResult{Message: err.Error()})
			fmt.Println(string(output))
		} else {
			exitWithError(fmt.Sprintf("\x1b[31mError: %v\x1b[0m", err), 1)
		}
		return
	}

	// Get remaining count
	items, _ = mgr.List()
	afterCount := len(items)
	removed := beforeCount - afterCount

	if flagJSON {
		result := EmptyResult{
			Removed:   removed,
			Remaining: afterCount,
		}
		if flagDays > 0 {
			result.Days = flagDays
		}
		output, _ := json.Marshal(result)
		fmt.Println(string(output))
	} else {
		if flagDays > 0 {
			fmt.Printf("\x1b[32mRemoved %d files older than %d days\x1b[0m\n", removed, flagDays)
		} else {
			fmt.Printf("\x1b[32mTrash emptied: %d files removed\x1b[0m\n", removed)
		}
	}
}

// Helper to parse days from string
func parseDays(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}

// Used in verbose mode to show what would be deleted
func listOldFiles(mgr *trash.Manager, days int) {
	items, _ := mgr.List()
	for _, item := range items {
		// Simple check - in real implementation, compare with cutoff
		fmt.Printf("  %s - %s\n", item.Name, ui.FormatSize(item.Size))
	}
}
