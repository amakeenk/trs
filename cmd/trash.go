package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"altlinux.space/amakeenk/trs/internal/trash"
	"github.com/spf13/cobra"
)

var (
	flagRecursive bool
)

// TrashResult for JSON output
type TrashResult struct {
	Files   []string `json:"files,omitempty"`
	Errors  []string `json:"errors,omitempty"`
	Success int      `json:"success"`
	Failed  int      `json:"failed"`
}

func init() {
	rootCmd.Flags().BoolVarP(&flagRecursive, "recursive", "r", false, "remove directories recursively")
}

func runTrash(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.Help()
		os.Exit(2)
	}

	mgr, err := trash.NewManager()
	if err != nil {
		exitWithError(fmt.Sprintf("Error: %v", err), 1)
	}

	if flagVerbose {
		mgr.SetVerboseCallback(func(path string, itemType trash.ItemType) {
			switch itemType {
			case trash.ItemTypeFile:
				fmt.Printf("\x1b[32mtrashed file %s\x1b[0m\n", path)
			case trash.ItemTypeSymlink:
				fmt.Printf("\x1b[36mtrashed symlink %s\x1b[0m\n", path)
			case trash.ItemTypeDirectory:
				fmt.Printf("\x1b[34mtrashed directory %s/\x1b[0m\n", path)
			}
		})
	}

	result := TrashResult{}

	for _, path := range args {
		err := trashFile(mgr, path)

		if err != nil {
			if os.IsNotExist(err) && flagForce {
				continue // Ignore missing files with -f
			}
			result.Errors = append(result.Errors, err.Error())
			result.Failed++
			if flagVerbose {
				fmt.Fprintf(os.Stderr, "\x1b[31mError: %v\x1b[0m\n", err)
			}
		} else {
			result.Success++
			result.Files = append(result.Files, path)
		}
	}

	if flagJSON {
		output, _ := json.Marshal(result)
		fmt.Println(string(output))
	} else if !flagVerbose && len(result.Errors) > 0 {
		// Print summary of errors
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "\x1b[31m%s\x1b[0m\n", e)
		}
	}

	if result.Failed > 0 {
		os.Exit(1)
	}
}

func trashFile(mgr *trash.Manager, path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("cannot trash '%s': No such file or directory", path)
		}
		return fmt.Errorf("cannot trash '%s': %w", path, err)
	}

	// Check if directory without -r flag
	if info.IsDir() && !flagRecursive {
		return fmt.Errorf("cannot trash '%s': Is a directory (use -r)", path)
	}

	return mgr.Move(path)
}
