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

func NewManageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manage [name|index]",
		Short: "Manage files in trash (restore or delete)",
		Long: `Manage files in trash with an interactive TUI.

Without arguments, shows interactive selection where you can:
- Restore selected files back to their original locations
- Permanently delete selected files

With --last, restores the most recently trashed file.
With a name or index, restores that specific file.`,
		Run: runManage,
	}

	cmd.Flags().BoolVar(&flagLast, "last", false, "restore the most recently trashed file")
	cmd.Flags().BoolVarP(&flagOverwrite, "force", "f", false, "overwrite existing files")

	return cmd
}

type ManageResult struct {
	Name     string `json:"name,omitempty"`
	Original string `json:"original_path,omitempty"`
	Action   string `json:"action,omitempty"`
	Success  bool   `json:"success,omitempty"`
	Error    string `json:"error,omitempty"`
}

func runManage(cmd *cobra.Command, args []string) {
	mgr, err := trash.NewManager()
	if err != nil {
		exitWithError(fmt.Sprintf("Error: %v", err), 1)
	}

	if flagLast {
		restoreLast(mgr)
		return
	}

	if len(args) == 0 {
		manageInteractive(mgr)
		return
	}

	restoreSpecific(mgr, args[0])
}

func restoreLast(mgr *trash.Manager) {
	item, err := mgr.GetLast()
	if err != nil {
		if flagJSON {
			output, _ := json.Marshal(ManageResult{Error: err.Error()})
			fmt.Println(string(output))
		} else {
			exitWithError(fmt.Sprintf("Error: %v", err), 1)
		}
		return
	}

	err = mgr.Restore(item.Name, flagOverwrite)
	if flagJSON {
		result := ManageResult{
			Name:     item.Name,
			Original: item.OriginalPath,
			Action:   "restore",
			Success:  err == nil,
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

func manageInteractive(mgr *trash.Manager) {
	items, err := mgr.List()
	if err != nil {
		exitWithError(fmt.Sprintf("Error: %v", err), 1)
	}

	if len(items) == 0 {
		fmt.Println("Trash is empty")
		return
	}

	if !ui.IsTerminal() {
		manageInteractiveSimple(mgr, items)
		return
	}

	model := tui.NewManageModel(items, flagOverwrite, mgr)
	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		exitWithError(fmt.Sprintf("TUI error: %v", err), 1)
	}

	result := finalModel.(tui.ManageModel)

	if result.Cancelled() {
		fmt.Println("Cancelled")
		return
	}

	if !result.Confirmed() {
		fmt.Println("Cancelled")
		return
	}

	if flagJSON {
		outputResultsJSON(result.Results())
	} else {
		outputResults(result.Results())
	}
}

func outputResults(results []tui.ActionResult) {
	if len(results) == 0 {
		fmt.Println("No action taken")
		return
	}

	successCount := 0
	failCount := 0

	for _, r := range results {
		name := r.Item.Name
		if r.Item.IsDir {
			name += "/"
		}

		if r.Success {
			successCount++
			actionText := "Restored"
			if r.Item.IsDir {
				actionText = "Restored"
			}
			fmt.Printf("\x1b[32m%s: %s → %s\x1b[0m\n", actionText, name, r.Item.OriginalPath)
		} else {
			failCount++
			errMsg := ""
			if r.Error != nil {
				errMsg = r.Error.Error()
			}
			fmt.Printf("\x1b[31mError: %s - %s\x1b[0m\n", name, errMsg)
		}
	}

	if successCount > 1 {
		fmt.Printf("\x1b[32mSuccessfully processed %d items\x1b[0m\n", successCount)
	}
	if failCount > 0 {
		fmt.Printf("\x1b[31mFailed: %d items\x1b[0m\n", failCount)
	}
}

func outputResultsJSON(results []tui.ActionResult) {
	output := make([]ManageResult, len(results))
	for i, r := range results {
		output[i] = ManageResult{
			Name:     r.Item.Name,
			Original: r.Item.OriginalPath,
			Success:  r.Success,
		}
		if r.Error != nil {
			output[i].Error = r.Error.Error()
		}
	}
	data, _ := json.Marshal(output)
	fmt.Println(string(data))
}

func manageInteractiveSimple(mgr *trash.Manager, items []trash.TrashItem) {
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

	if idx, err := strconv.Atoi(input); err == nil {
		if idx < 1 || idx > len(items) {
			exitWithError(fmt.Sprintf("Invalid index: %d (must be 1-%d)", idx, len(items)), 2)
		}
		target = &items[idx-1]
	} else {
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
		result := ManageResult{
			Name:     target.Name,
			Original: target.OriginalPath,
			Action:   "restore",
			Success:  err == nil,
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

func NewRestoreCmd() *cobra.Command {
	return NewManageCmd()
}
