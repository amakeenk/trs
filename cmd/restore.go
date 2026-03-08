package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"altlinux.space/amakeenk/trs/internal/trash"
	"altlinux.space/amakeenk/trs/internal/tui"
	"altlinux.space/amakeenk/trs/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	flagLast      bool
	flagOverwrite bool
)

// NewRestoreCmd creates the restore command
func NewRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore [name|index]",
		Short: "Restore files from trash",
		Long: `Restore files from trash.

Without arguments, shows interactive selection.
With --last, restores the most recently trashed file.
With a name or index, restores that specific file.`,
		Run: runRestore,
	}

	cmd.Flags().BoolVar(&flagLast, "last", false, "restore the most recently trashed file")
	cmd.Flags().BoolVarP(&flagOverwrite, "force", "f", false, "overwrite existing files")

	return cmd
}

// RestoreResult for JSON output
type RestoreResult struct {
	Name     string `json:"name,omitempty"`
	Original string `json:"original_path,omitempty"`
	Error    string `json:"error,omitempty"`
}

func runRestore(cmd *cobra.Command, args []string) {
	mgr, err := trash.NewManager()
	if err != nil {
		exitWithError(fmt.Sprintf("Error: %v", err), 1)
	}

	// --last flag
	if flagLast {
		restoreLast(mgr)
		return
	}

	// No args - interactive mode
	if len(args) == 0 {
		restoreInteractive(mgr)
		return
	}

	// Specific file or index
	restoreSpecific(mgr, args[0])
}

func restoreLast(mgr *trash.Manager) {
	item, err := mgr.GetLast()
	if err != nil {
		if flagJSON {
			output, _ := json.Marshal(RestoreResult{Error: err.Error()})
			fmt.Println(string(output))
		} else {
			exitWithError(fmt.Sprintf("Error: %v", err), 1)
		}
		return
	}

	err = mgr.Restore(item.Name, flagOverwrite)
	if flagJSON {
		result := RestoreResult{
			Name:     item.Name,
			Original: item.OriginalPath,
		}
		if err != nil {
			result.Error = err.Error()
		}
		output, _ := json.Marshal(result)
		fmt.Println(string(output))
	} else if err != nil {
		exitWithError(fmt.Sprintf("\x1b[31mError: %v\x1b[0m", err), 1)
	} else {
		fmt.Printf("\x1b[32mRestored: %s → %s\x1b[0m\n", item.Name, item.OriginalPath)
	}
}

func restoreInteractive(mgr *trash.Manager) {
	items, err := mgr.List()
	if err != nil {
		exitWithError(fmt.Sprintf("Error: %v", err), 1)
	}

	if len(items) == 0 {
		fmt.Println("Trash is empty")
		return
	}

	// Check if stdout is a terminal
	if !ui.IsTerminal() {
		// Fallback to simple list for non-TTY
		restoreInteractiveSimple(mgr, items)
		return
	}

	// Launch TUI
	model := tui.NewRestoreModel(items, flagOverwrite)
	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		exitWithError(fmt.Sprintf("TUI error: %v", err), 1)
	}

	result := finalModel.(tui.RestoreModel)

	if result.Cancelled() {
		fmt.Println("Cancelled")
		return
	}

	if !result.Confirmed() {
		fmt.Println("Cancelled")
		return
	}

	selected := result.SelectedItem()
	if selected == nil {
		fmt.Println("No file selected")
		return
	}

	err = mgr.Restore(selected.Name, result.Force())
	if flagJSON {
		restoreResult := RestoreResult{
			Name:     selected.Name,
			Original: selected.OriginalPath,
		}
		if err != nil {
			restoreResult.Error = err.Error()
		}
		output, _ := json.Marshal(restoreResult)
		fmt.Println(string(output))
	} else if err != nil {
		exitWithError(fmt.Sprintf("\x1b[31mError: %v\x1b[0m", err), 1)
	} else {
		fmt.Printf("\x1b[32mRestored: %s → %s\x1b[0m\n", selected.Name, selected.OriginalPath)
	}
}

// restoreInteractiveSimple is a fallback for non-TTY environments
func restoreInteractiveSimple(mgr *trash.Manager, items []trash.TrashItem) {
	// Simple numbered list
	for i, item := range items {
		fmt.Printf("%d) %s\n", i+1, item.Name)
	}

	fmt.Print("\nRestore which? [1-", len(items), "]: ")

	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(input)

	if input == "" {
		fmt.Println("Cancelled")
		return
	}

	// Validate input for security
	if err := validateCLIInput(input); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	restoreByInput(mgr, items, input)
}

func restoreSpecific(mgr *trash.Manager, nameOrIndex string) {
	items, err := mgr.List()
	if err != nil {
		exitWithError(fmt.Sprintf("Error: %v", err), 1)
	}

	restoreByInput(mgr, items, nameOrIndex)
}

func restoreByInput(mgr *trash.Manager, items []trash.TrashItem, input string) {
	var target *trash.TrashItem

	// Try as index first
	if idx, err := strconv.Atoi(input); err == nil {
		if idx < 1 || idx > len(items) {
			exitWithError(fmt.Sprintf("Invalid index: %d (must be 1-%d)", idx, len(items)), 2)
		}
		target = &items[idx-1]
	} else {
		// Try as name - find exact or partial match
		var matches []trash.TrashItem
		for _, item := range items {
			if item.Name == input {
				target = &item
				break
			}
			if strings.HasPrefix(item.Name, input) {
				matches = append(matches, item)
			}
		}

		if target == nil {
			if len(matches) == 1 {
				target = &matches[0]
			} else if len(matches) > 1 {
				exitWithError(fmt.Sprintf("Ambiguous name '%s' matches %d files. Use full name or index.", input, len(matches)), 2)
			}
		}
	}

	if target == nil {
		exitWithError(fmt.Sprintf("File not found: %s", input), 2)
	}

	err := mgr.Restore(target.Name, flagOverwrite)
	if flagJSON {
		result := RestoreResult{
			Name:     target.Name,
			Original: target.OriginalPath,
		}
		if err != nil {
			result.Error = err.Error()
		}
		output, _ := json.Marshal(result)
		fmt.Println(string(output))
	} else if err != nil {
		exitWithError(fmt.Sprintf("\x1b[31mError: %v\x1b[0m", err), 1)
	} else {
		fmt.Printf("\x1b[32mRestored: %s → %s\x1b[0m\n", target.Name, target.OriginalPath)
	}
}
