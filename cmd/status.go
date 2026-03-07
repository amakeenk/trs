package cmd

import (
	"encoding/json"
	"fmt"

	"altlinux.space/amakeenk/trs/internal/trash"
	"altlinux.space/amakeenk/trs/internal/ui"
	"github.com/spf13/cobra"
)

var statusVerbose bool

// NewStatusCmd creates the status command
func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show trash statistics",
		Long: `Show trash size and file count.

With -v/--verbose, also shows oldest/newest dates and largest files.`,
		Run: runStatus,
	}

	cmd.Flags().BoolVarP(&statusVerbose, "verbose", "v", false, "show detailed statistics")

	return cmd
}

// StatusResult for JSON output
type StatusResult struct {
	Count     int           `json:"count"`
	Size      string        `json:"size"`
	SizeBytes int64         `json:"size_bytes"`
	Oldest    string        `json:"oldest,omitempty"`
	Newest    string        `json:"newest,omitempty"`
	Largest   []LargestItem `json:"largest,omitempty"`
}

// LargestItem for JSON output
type LargestItem struct {
	Name string `json:"name"`
	Size string `json:"size"`
}

func runStatus(cmd *cobra.Command, args []string) {
	mgr, err := trash.NewManager()
	if err != nil {
		exitWithError(fmt.Sprintf("Error: %v", err), 1)
	}

	count, totalSize, err := mgr.Status()
	if err != nil {
		exitWithError(fmt.Sprintf("Error: %v", err), 1)
	}

	if flagJSON {
		outputJSONStatus(mgr, count, totalSize)
	} else {
		outputTextStatus(mgr, count, totalSize)
	}
}

func outputJSONStatus(mgr *trash.Manager, count int, totalSize int64) {
	result := StatusResult{
		Count:     count,
		Size:      ui.FormatSize(totalSize),
		SizeBytes: totalSize,
	}

	if statusVerbose && count > 0 {
		oldest, newest, _ := mgr.GetOldestAndNewest()
		result.Oldest = oldest.Format("2006-01-02 15:04")
		result.Newest = newest.Format("2006-01-02 15:04")

		largest, _ := mgr.GetLargest(3)
		result.Largest = make([]LargestItem, len(largest))
		for i, item := range largest {
			result.Largest[i] = LargestItem{
				Name: item.Name,
				Size: ui.FormatSize(item.Size),
			}
		}
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
}

func outputTextStatus(mgr *trash.Manager, count int, totalSize int64) {
	// Basic output
	fmt.Printf("Trash: %d files, %s\n", count, ui.FormatSize(totalSize))

	if !statusVerbose || count == 0 {
		return
	}

	// Verbose output
	oldest, newest, err := mgr.GetOldestAndNewest()
	if err == nil {
		fmt.Printf("Oldest: %s  Newest: %s\n",
			oldest.Format("2006-01-02 15:04"),
			newest.Format("2006-01-02 15:04"))
	}

	largest, err := mgr.GetLargest(3)
	if err == nil && len(largest) > 0 {
		names := make([]string, len(largest))
		for i, item := range largest {
			names[i] = fmt.Sprintf("%s (%s)", item.Name, ui.FormatSize(item.Size))
		}
		fmt.Printf("Largest: %s", names[0])
		for i := 1; i < len(names); i++ {
			fmt.Printf(", %s", names[i])
		}
		fmt.Println()
	}
}
