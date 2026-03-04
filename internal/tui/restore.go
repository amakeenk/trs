package tui

import (
	"fmt"
	"strings"

	"github.com/amakeenk/trs/internal/trash"
	"github.com/amakeenk/trs/internal/ui"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// RestoreModel is the TUI model for restore selection
type RestoreModel struct {
	items     []trash.TrashItem
	filtered  []trash.TrashItem
	selected  int
	search    textinput.Model
	confirmed bool
	cancelled bool
	force     bool
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			MarginBottom(1)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedStyle = lipgloss.NewStyle().
			PaddingLeft(1).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("63"))

	searchStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color("14"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	previewStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246")).
			PaddingLeft(2)

	dirStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("81"))
)

// NewRestoreModel creates a new restore TUI model
func NewRestoreModel(items []trash.TrashItem, force bool) RestoreModel {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50

	return RestoreModel{
		items:    items,
		filtered: items,
		search:   ti,
		selected: 0,
		force:    force,
	}
}

func (m RestoreModel) Init() tea.Cmd {
	return nil
}

func (m RestoreModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			return m, tea.Quit

		case tea.KeyEnter:
			if len(m.filtered) > 0 {
				m.confirmed = true
				return m, tea.Quit
			}

		case tea.KeyUp, tea.KeyCtrlP:
			if m.selected > 0 {
				m.selected--
			}

		case tea.KeyDown, tea.KeyCtrlN:
			if m.selected < len(m.filtered)-1 {
				m.selected++
			}
		}

		// Handle search input
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete {
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			m.filterItems()
			if m.selected >= len(m.filtered) {
				m.selected = max(0, len(m.filtered)-1)
			}
			return m, cmd
		}
	}

	return m, nil
}

func (m *RestoreModel) filterItems() {
	query := strings.ToLower(m.search.Value())
	if query == "" {
		m.filtered = m.items
		return
	}

	var filtered []trash.TrashItem
	for _, item := range m.items {
		if fuzzyMatch(strings.ToLower(item.Name), query) ||
			fuzzyMatch(strings.ToLower(item.OriginalPath), query) {
			filtered = append(filtered, item)
		}
	}
	m.filtered = filtered
}

// fuzzyMatch checks if query chars appear in order in s
func fuzzyMatch(s, query string) bool {
	if len(query) == 0 {
		return true
	}

	qi := 0
	for _, c := range s {
		if qi < len(query) && byte(c) == query[qi] {
			qi++
		}
	}
	return qi == len(query)
}

func (m RestoreModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("🗑️  Restore from Trash"))
	b.WriteString("\n")

	// Search input
	b.WriteString(searchStyle.Render("> " + m.search.View()))
	b.WriteString("\n\n")

	// Items list
	if len(m.filtered) == 0 {
		b.WriteString(itemStyle.Render("No matching files"))
	} else {
		maxDisplay := 10
		start := 0
		if m.selected >= maxDisplay {
			start = m.selected - maxDisplay + 1
		}
		end := min(start+maxDisplay, len(m.filtered))

		for i := start; i < end; i++ {
			item := m.filtered[i]
			name := item.Name
			if item.IsDir {
				name = dirStyle.Render(name + "/")
			}

			line := fmt.Sprintf("%s  %s", ui.FormatSize(item.Size), name)

			if i == m.selected {
				line = "▶ " + selectedStyle.Render(line)
			} else {
				line = "  " + itemStyle.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}

		if len(m.filtered) > maxDisplay {
			b.WriteString(helpStyle.Render(fmt.Sprintf("  ... %d more", len(m.filtered)-maxDisplay)))
			b.WriteString("\n")
		}
	}

	// Preview
	if len(m.filtered) > 0 && m.selected < len(m.filtered) {
		item := m.filtered[m.selected]
		b.WriteString("\n")
		b.WriteString(previewStyle.Render("Original: " + item.OriginalPath))
		b.WriteString("\n")
		b.WriteString(previewStyle.Render("Deleted:  " + item.DeletionDate.Format("2006-01-02 15:04")))
	}

	// Help
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓ navigate • Enter restore • Esc cancel"))

	return b.String()
}

// SelectedItem returns the selected item
func (m RestoreModel) SelectedItem() *trash.TrashItem {
	if len(m.filtered) > 0 && m.selected < len(m.filtered) {
		return &m.filtered[m.selected]
	}
	return nil
}

// Confirmed returns whether user confirmed selection
func (m RestoreModel) Confirmed() bool {
	return m.confirmed
}

// Cancelled returns whether user cancelled
func (m RestoreModel) Cancelled() bool {
	return m.cancelled
}

// Force returns whether to overwrite existing files
func (m RestoreModel) Force() bool {
	return m.force
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
