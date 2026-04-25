package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"altlinux.space/amakeenk/trs/internal/trash"
	"altlinux.space/amakeenk/trs/internal/ui"
	"github.com/spf13/cobra"
)

// NewListCmd creates the list command
func NewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List files in trash",
		Long:  "List all files in the trash with their index, name, size, and deletion date",
		Run:   runList,
	}
}

// ListItem for JSON output
type ListItem struct {
	Index     int    `json:"index"`
	Name      string `json:"name"`
	Size      string `json:"size"`
	SizeBytes int64  `json:"size_bytes"`
	Deleted   string `json:"deleted"`
	IsDir     bool   `json:"is_dir"`
	Original  string `json:"original_path"`
}

func runList(cmd *cobra.Command, args []string) {
	mgr, err := trash.NewManager()
	if err != nil {
		exitWithError(fmt.Sprintf("Error: %v", err), 1)
	}

	items, err := mgr.List()
	if err != nil {
		exitWithError(fmt.Sprintf("Error listing trash: %v", err), 1)
	}

	if len(items) == 0 {
		if flagJSON {
			fmt.Println("[]")
		} else {
			fmt.Println("Trash is empty")
		}
		return
	}

	if flagJSON {
		outputJSON(items)
	} else {
		outputTable(items)
	}
}

func outputJSON(items []trash.TrashItem) {
	result := make([]ListItem, len(items))
	for i, item := range items {
		result[i] = ListItem{
			Index:     i + 1,
			Name:      item.Name,
			Size:      ui.FormatSize(item.Size),
			SizeBytes: item.Size,
			Deleted:   item.DeletionDate.Format("2006-01-02 15:04"),
			IsDir:     item.IsDir,
			Original:  item.OriginalPath,
		}
	}
	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
}

const (
	colIndexWidth    = 5
	colNameWidth     = 60
	colOriginalWidth = 70
	colSizeWidth     = 10
	colDeletedWidth  = 17
)

func formatColumn(s string, width int) string {
	dw := ui.DisplayWidth(s)
	if dw > width {
		runes := []rune(s)
		w := 0
		for i, r := range runes {
			rw := 1
			if r >= 0x1100 {
				rw = 2
			}
			if w+rw > width-3 {
				return string(runes[:i]) + "..."
			}
			w += rw
		}
		return s
	}
	return s + strings.Repeat(" ", width-dw)
}

func outputTable(items []trash.TrashItem) {
	header := fmt.Sprintf("%s  %s  %s  %s  %s",
		ui.BoldText(formatColumn("#", colIndexWidth)),
		ui.BoldText(formatColumn("NAME", colNameWidth)),
		ui.BoldText(formatColumn("ORIGINAL", colOriginalWidth)),
		ui.BoldText(formatColumn("SIZE", colSizeWidth)),
		ui.BoldText(formatColumn("DELETED", colDeletedWidth)),
	)
	fmt.Println(header)
	fmt.Printf("%s  %s  %s  %s  %s\n",
		strings.Repeat("─", colIndexWidth),
		strings.Repeat("─", colNameWidth),
		strings.Repeat("─", colOriginalWidth),
		strings.Repeat("─", colSizeWidth),
		strings.Repeat("─", colDeletedWidth),
	)

	homeDir, _ := os.UserHomeDir()

	for i, item := range items {
		name := ui.Truncate(item.Name, colNameWidth-1)
		if item.IsDir {
			name = ui.Directory(formatColumn(name+"/", colNameWidth))
		} else {
			name = formatColumn(name, colNameWidth)
		}

		origPath := item.OriginalPath
		if homeDir != "" && strings.HasPrefix(origPath, homeDir) {
			origPath = "~" + strings.TrimPrefix(origPath, homeDir)
		}

		orig := ui.Truncate(origPath, colOriginalWidth-1)
		orig = formatColumn(orig, colOriginalWidth)

		row := fmt.Sprintf("%s  %s  %s  %s  %s",
			formatColumn(fmt.Sprintf("%d", i+1), colIndexWidth),
			name,
			orig,
			formatColumn(ui.FormatSize(item.Size), colSizeWidth),
			formatColumn(item.DeletionDate.Format("2006-01-02 15:04"), colDeletedWidth),
		)
		fmt.Println(row)
	}
}
