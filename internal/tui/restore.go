package tui

import (
	"fmt"
	"strings"

	"altlinux.space/amakeenk/trs/internal/trash"
	"altlinux.space/amakeenk/trs/internal/ui"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type mode int

const (
	modeSelect mode = iota
	modeSearch
)

// RestoreModel is the TUI model for restore selection
type RestoreModel struct {
	items         []trash.TrashItem
	filtered      []trash.TrashItem
	selected      int
	selectedItems map[string]bool
	search        textinput.Model
	mode          mode
	confirmed     bool
	cancelled     bool
	force         bool
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

	checkedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))
)

func itemKey(item trash.TrashItem) string {
	return item.Name + "|" + item.OriginalPath
}

// NewRestoreModel creates a new restore TUI model
func NewRestoreModel(items []trash.TrashItem, force bool) RestoreModel {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 100
	ti.Width = 50

	return RestoreModel{
		items:         items,
		filtered:      items,
		search:        ti,
		selected:      0,
		selectedItems: make(map[string]bool),
		mode:          modeSelect,
		force:         force,
	}
}

func (m RestoreModel) Init() tea.Cmd {
	return nil
}

func (m RestoreModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.mode {
		case modeSearch:
			return m.updateSearchMode(msg)
		case modeSelect:
			return m.updateSelectMode(msg)
		}
	}

	return m, nil
}

func (m RestoreModel) updateSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		if m.search.Value() != "" {
			m.search.SetValue("")
			m.filtered = m.items
			m.selected = 0
			return m, nil
		}
		m.mode = modeSelect
		m.search.Blur()
		return m, nil

	case tea.KeyEnter:
		if len(m.filtered) > 0 {
			m.confirmed = true
			return m, tea.Quit
		}
		return m, nil

	case tea.KeySpace:
		if m.search.Value() == "" && len(m.filtered) > 0 && m.selected < len(m.filtered) {
			key := itemKey(m.filtered[m.selected])
			m.selectedItems[key] = !m.selectedItems[key]
		}
		return m, nil

	case tea.KeyTab:
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			key := itemKey(m.filtered[m.selected])
			m.selectedItems[key] = !m.selectedItems[key]
		}
		return m, nil

	case tea.KeyUp, tea.KeyCtrlP:
		if m.selected > 0 {
			m.selected--
		}
		return m, nil

	case tea.KeyDown, tea.KeyCtrlN:
		if m.selected < len(m.filtered)-1 {
			m.selected++
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	m.filterItems()
	if m.selected >= len(m.filtered) {
		m.selected = max(0, len(m.filtered)-1)
	}
	return m, cmd
}

func (m RestoreModel) updateSelectMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	case tea.KeySpace:
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			key := itemKey(m.filtered[m.selected])
			m.selectedItems[key] = !m.selectedItems[key]
		}
		return m, nil
	}

	switch msg.String() {
	case "/":
		m.mode = modeSearch
		m.search.Focus()
		return m, nil
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

	b.WriteString(titleStyle.Render("🗑️  Restore from Trash"))
	b.WriteString("\n")

	if m.mode == modeSearch {
		b.WriteString(searchStyle.Render("> " + m.search.View()))
		b.WriteString("\n\n")
	} else {
		b.WriteString(previewStyle.Render("/ to search"))
		b.WriteString("\n\n")
	}

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

			check := " "
			if m.selectedItems[itemKey(item)] {
				check = checkedStyle.Render("✓")
			}

			line := fmt.Sprintf("[%s] %s  %s", check, ui.FormatSize(item.Size), name)

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

	if len(m.selectedItems) > 0 {
		b.WriteString("\n")
		b.WriteString(helpStyle.Render(fmt.Sprintf("Selected (%d):", len(m.selectedItems))))
		b.WriteString("\n")
		count := 0
		for _, item := range m.items {
			if m.selectedItems[itemKey(item)] {
				name := item.Name
				if item.IsDir {
					name = dirStyle.Render(name + "/")
				}
				b.WriteString(previewStyle.Render("  • " + name))
				b.WriteString("\n")
				count++
				if count >= 5 {
					remaining := len(m.selectedItems) - count
					if remaining > 0 {
						b.WriteString(previewStyle.Render(fmt.Sprintf("  ... and %d more", remaining)))
						b.WriteString("\n")
					}
					break
				}
			}
		}
	}

	if len(m.filtered) > 0 && m.selected < len(m.filtered) {
		item := m.filtered[m.selected]
		b.WriteString("\n")
		b.WriteString(previewStyle.Render("Original: " + item.OriginalPath))
		b.WriteString("\n")
		b.WriteString(previewStyle.Render("Deleted:  " + item.DeletionDate.Format("2006-01-02 15:04")))
	}

	b.WriteString("\n")
	if m.mode == modeSearch {
		selectedCount := len(m.selectedItems)
		if selectedCount > 0 {
			b.WriteString(helpStyle.Render(fmt.Sprintf("Esc clear • ↑/↓ nav • Tab select (%d) • Enter restore", selectedCount)))
		} else {
			b.WriteString(helpStyle.Render("Esc clear/exit • ↑/↓ navigate • Tab select • Enter restore"))
		}
	} else {
		selectedCount := len(m.selectedItems)
		if selectedCount > 0 {
			b.WriteString(helpStyle.Render(fmt.Sprintf("↑/↓ navigate • Space select (%d selected) • / search • Enter restore • Esc cancel", selectedCount)))
		} else {
			b.WriteString(helpStyle.Render("↑/↓ navigate • Space select • / search • Enter restore • Esc cancel"))
		}
	}

	return b.String()
}

// SelectedItem returns the selected item (for single selection compatibility)
func (m RestoreModel) SelectedItem() *trash.TrashItem {
	if len(m.filtered) > 0 && m.selected < len(m.filtered) {
		return &m.filtered[m.selected]
	}
	return nil
}

// SelectedItems returns all selected items for multi-restore
func (m RestoreModel) SelectedItems() []trash.TrashItem {
	if len(m.selectedItems) == 0 {
		if item := m.SelectedItem(); item != nil {
			return []trash.TrashItem{*item}
		}
		return nil
	}

	items := make([]trash.TrashItem, 0, len(m.selectedItems))
	for _, item := range m.items {
		if m.selectedItems[itemKey(item)] {
			items = append(items, item)
		}
	}
	return items
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
