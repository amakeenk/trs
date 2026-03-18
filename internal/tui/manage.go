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
	modeConfirm
	modeResults
)

// ActionType represents the type of action performed
type ActionType int

const (
	ActionRestore ActionType = iota
	ActionDelete
)

// ActionResult represents the result of an action on a single item
type ActionResult struct {
	Item    trash.TrashItem
	Action  ActionType
	Success bool
	Error   error
}

// ManageModel is the TUI model for trash management
type ManageModel struct {
	items         []trash.TrashItem
	filtered      []trash.TrashItem
	selected      int
	selectedItems map[string]bool
	search        textinput.Model
	mode          mode
	action        ActionType
	confirmed     bool
	cancelled     bool
	force         bool
	manager       *trash.Manager
	results       []ActionResult
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

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	confirmBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("214")).
			Padding(1, 2).
			MarginTop(2)
)

func itemKey(item trash.TrashItem) string {
	return item.Name + "|" + item.OriginalPath
}

// NewManageModel creates a new manage TUI model
func NewManageModel(items []trash.TrashItem, force bool, manager *trash.Manager) ManageModel {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 100
	ti.Width = 50

	return ManageModel{
		items:         items,
		filtered:      items,
		search:        ti,
		selected:      0,
		selectedItems: make(map[string]bool),
		mode:          modeSelect,
		force:         force,
		manager:       manager,
	}
}

func (m ManageModel) Init() tea.Cmd {
	return nil
}

func (m ManageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.mode {
		case modeSearch:
			return m.updateSearchMode(msg)
		case modeSelect:
			return m.updateSelectMode(msg)
		case modeConfirm:
			return m.updateConfirmMode(msg)
		case modeResults:
			return m.updateResultsMode(msg)
		}
	}

	return m, nil
}

func (m ManageModel) updateSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		// Toggle selection instead of confirming
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			key := itemKey(m.filtered[m.selected])
			m.selectedItems[key] = !m.selectedItems[key]
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
		if len(m.filtered) > 0 {
			if m.selected > 0 {
				m.selected--
			} else {
				m.selected = len(m.filtered) - 1
			}
		}
		return m, nil

	case tea.KeyDown, tea.KeyCtrlN:
		if len(m.filtered) > 0 {
			if m.selected < len(m.filtered)-1 {
				m.selected++
			} else {
				m.selected = 0
			}
		}
		return m, nil
	}

	// Check for action keys 'r' and 'd'
	switch msg.String() {
	case "r":
		if len(m.selectedItems) > 0 {
			m.action = ActionRestore
			m.mode = modeConfirm
			return m, nil
		}
	case "d":
		if len(m.selectedItems) > 0 {
			m.action = ActionDelete
			m.mode = modeConfirm
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	m.filterItems()
	if m.selected >= len(m.filtered) {
		m.selected = max(0, len(m.filtered)-1)
	}
	return m, cmd
}

func (m ManageModel) updateSelectMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.cancelled = true
		return m, tea.Quit

	case tea.KeyEnter:
		// Toggle selection instead of confirming
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			key := itemKey(m.filtered[m.selected])
			m.selectedItems[key] = !m.selectedItems[key]
		}
		return m, nil

	case tea.KeyUp, tea.KeyCtrlP:
		if len(m.filtered) > 0 {
			if m.selected > 0 {
				m.selected--
			} else {
				m.selected = len(m.filtered) - 1
			}
		}

	case tea.KeyDown, tea.KeyCtrlN:
		if len(m.filtered) > 0 {
			if m.selected < len(m.filtered)-1 {
				m.selected++
			} else {
				m.selected = 0
			}
		}

	case tea.KeyTab, tea.KeySpace:
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
	case "r":
		if len(m.selectedItems) > 0 {
			m.action = ActionRestore
			m.mode = modeConfirm
			return m, nil
		}
	case "d":
		if len(m.selectedItems) > 0 {
			m.action = ActionDelete
			m.mode = modeConfirm
			return m, nil
		}
	}

	return m, nil
}

func (m ManageModel) updateConfirmMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.mode = modeSelect
		return m, nil

	case tea.KeyEnter:
		// Execute the action
		m.executeAction()
		m.mode = modeResults
		return m, nil
	}

	switch msg.String() {
	case "y", "Y":
		m.executeAction()
		m.mode = modeResults
		return m, nil
	case "n", "N":
		m.mode = modeSelect
		return m, nil
	}

	return m, nil
}

func (m ManageModel) updateResultsMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc, tea.KeyEnter:
			m.confirmed = true
			return m, tea.Quit
		}
		switch msg.String() {
		case "q", "Q":
			m.confirmed = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *ManageModel) executeAction() {
	selected := m.SelectedItems()
	m.results = make([]ActionResult, 0, len(selected))

	for _, item := range selected {
		result := ActionResult{
			Item:    item,
			Action:  m.action,
			Success: false,
		}

		if m.manager != nil {
			var err error
			if m.action == ActionRestore {
				err = m.manager.Restore(item.Name, m.force)
			} else {
				err = m.manager.Delete(item.Name)
			}
			result.Success = err == nil
			result.Error = err
		}

		m.results = append(m.results, result)
	}
}

func (m *ManageModel) filterItems() {
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

func (m ManageModel) View() string {
	switch m.mode {
	case modeConfirm:
		return m.viewConfirm()
	case modeResults:
		return m.viewResults()
	default:
		return m.viewSelect()
	}
}

func (m ManageModel) viewSelect() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🗑️  Manage Trash"))
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
		maxDisplay := 20
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
				if count >= 10 {
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
			b.WriteString(helpStyle.Render(fmt.Sprintf("Esc clear • ↑/↓ nav • Tab select (%d) • r restore • d delete", selectedCount)))
		} else {
			b.WriteString(helpStyle.Render("Esc clear/exit • ↑/↓ navigate • Tab select • r restore • d delete"))
		}
	} else {
		selectedCount := len(m.selectedItems)
		if selectedCount > 0 {
			b.WriteString(helpStyle.Render(fmt.Sprintf("↑/↓ nav • Tab select (%d) • / search • r restore • d delete • Esc cancel", selectedCount)))
		} else {
			b.WriteString(helpStyle.Render("↑/↓ navigate • Tab select • / search • r restore • d delete • Esc cancel"))
		}
	}

	return b.String()
}

func (m ManageModel) viewConfirm() string {
	var b strings.Builder

	actionText := "restore"
	actionColor := successStyle
	if m.action == ActionDelete {
		actionText = "permanently delete"
		actionColor = errorStyle
	}

	title := fmt.Sprintf("Confirm %s?", actionText)
	b.WriteString(titleStyle.Render("🗑️  " + title))
	b.WriteString("\n\n")

	selected := m.SelectedItems()
	b.WriteString(warningStyle.Render(fmt.Sprintf("You are about to %s %d item(s):", actionText, len(selected))))
	b.WriteString("\n\n")

	for i, item := range selected {
		if i >= 10 {
			b.WriteString(previewStyle.Render(fmt.Sprintf("  ... and %d more", len(selected)-i)))
			b.WriteString("\n")
			break
		}
		name := item.Name
		if item.IsDir {
			name = dirStyle.Render(name + "/")
		}
		b.WriteString(previewStyle.Render("  • " + name))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(confirmBoxStyle.Render(fmt.Sprintf(
		"Press %s to confirm, %s to cancel",
		actionColor.Render("y/Enter"),
		helpStyle.Render("n/Esc"),
	)))

	return b.String()
}

func (m ManageModel) viewResults() string {
	var b strings.Builder

	actionText := "Restored"
	if m.action == ActionDelete {
		actionText = "Deleted"
	}

	b.WriteString(titleStyle.Render(fmt.Sprintf("🗑️  %s Results", actionText)))
	b.WriteString("\n\n")

	successCount := 0
	failCount := 0

	for _, result := range m.results {
		displayPath := result.Item.OriginalPath
		if displayPath == "" {
			displayPath = result.Item.Name
		}
		if result.Item.IsDir && !strings.HasSuffix(displayPath, "/") {
			displayPath = dirStyle.Render(displayPath + "/")
		}

		if result.Success {
			successCount++
			b.WriteString(successStyle.Render("✓ " + displayPath))
			b.WriteString("\n")
		} else {
			failCount++
			b.WriteString(errorStyle.Render("✗ " + displayPath))
			if result.Error != nil {
				b.WriteString(errorStyle.Render(": " + result.Error.Error()))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	if failCount == 0 {
		b.WriteString(successStyle.Render(fmt.Sprintf("Successfully %s %d item(s)", strings.ToLower(actionText), successCount)))
	} else {
		b.WriteString(warningStyle.Render(fmt.Sprintf("%d succeeded, %d failed", successCount, failCount)))
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Press Enter or q to exit"))

	return b.String()
}

// SelectedItem returns the selected item (for single selection compatibility)
func (m ManageModel) SelectedItem() *trash.TrashItem {
	if len(m.filtered) > 0 && m.selected < len(m.filtered) {
		return &m.filtered[m.selected]
	}
	return nil
}

// SelectedItems returns all selected items for multi-restore
func (m ManageModel) SelectedItems() []trash.TrashItem {
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
func (m ManageModel) Confirmed() bool {
	return m.confirmed
}

// Cancelled returns whether user cancelled
func (m ManageModel) Cancelled() bool {
	return m.cancelled
}

// Force returns whether to overwrite existing files
func (m ManageModel) Force() bool {
	return m.force
}

// Results returns the action results
func (m ManageModel) Results() []ActionResult {
	return m.results
}

// Action returns the selected action
func (m ManageModel) Action() ActionType {
	return m.action
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
