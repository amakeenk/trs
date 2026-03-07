package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

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

func outputTable(items []trash.TrashItem) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Header
		fmt.Fprintln(w, ui.BoldText("INDEX\tNAME\tSIZE\tDELETED"))

	// Rows
	for i, item := range items {
		name := item.Name
		if item.IsDir {
			name = ui.Directory(name + "/")
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
			i+1,
			name,
			ui.FormatSize(item.Size),
			item.DeletionDate.Format("2006-01-02 15:04"),
		)
	}

	w.Flush()
}
