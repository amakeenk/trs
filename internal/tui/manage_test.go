package tui

import (
	"fmt"
	"os"
	"testing"
	"time"

	"altlinux.space/amakeenk/trs/internal/trash"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTime(offset int) time.Time {
	return time.Date(2025, 1, 1, 12, 0, offset, 0, time.UTC)
}

func TestNewManageModel(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(1), Size: 100, IsDir: false},
		{Name: "file2.txt", OriginalPath: "/path/to/file2.txt", DeletionDate: makeTime(0), Size: 200, IsDir: false},
	}

	model := NewManageModel(items, false, nil)

	assert.Len(t, model.items, 2)
	assert.Len(t, model.filtered, 2)
	assert.Equal(t, 0, model.selected)
	assert.False(t, model.force)
	assert.False(t, model.confirmed)
	assert.False(t, model.cancelled)
}

func TestNewManageModelWithForce(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file.txt", OriginalPath: "/path/to/file.txt", DeletionDate: makeTime(1), Size: 100, IsDir: false},
	}

	model := NewManageModel(items, true, nil)

	assert.True(t, model.force)
}

func TestManageModelSelectAll(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(2), Size: 100, IsDir: false},
		{Name: "file2.txt", OriginalPath: "/path/to/file2.txt", DeletionDate: makeTime(3), Size: 200, IsDir: false},
		{Name: "other.txt", OriginalPath: "/path/to/other.txt", DeletionDate: makeTime(4), Size: 300, IsDir: false},
	}

	t.Run("select all with 'a'", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}

		updatedModel, _ := model.Update(msg)
		m := updatedModel.(ManageModel)

		assert.Len(t, m.selectedItems, 3)
		assert.True(t, m.selectedItems[itemKey(items[0])])
		assert.True(t, m.selectedItems[itemKey(items[1])])
		assert.True(t, m.selectedItems[itemKey(items[2])])
	})

	t.Run("deselect all with 'A'", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.selectedItems[itemKey(items[0])] = true
		model.selectedItems[itemKey(items[1])] = true

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("A")}

		updatedModel, _ := model.Update(msg)
		m := updatedModel.(ManageModel)

		assert.Len(t, m.selectedItems, 0)
	})

	t.Run("select all with Ctrl+a", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		msg := tea.KeyMsg{Type: tea.KeyCtrlA}

		updatedModel, _ := model.Update(msg)
		m := updatedModel.(ManageModel)

		assert.Len(t, m.selectedItems, 3)
	})

	t.Run("select all in search mode", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeSearch
		model.search.SetValue("file")
		model.filterItems() // "file1.txt" and "file2.txt"

		msg := tea.KeyMsg{Type: tea.KeyCtrlA}

		updatedModel, _ := model.Update(msg)
		m := updatedModel.(ManageModel)

		assert.Len(t, m.selectedItems, 2)
		assert.True(t, m.selectedItems[itemKey(items[0])])
		assert.True(t, m.selectedItems[itemKey(items[1])])
		assert.False(t, m.selectedItems[itemKey(items[2])])
	})
}

func TestManageModelInit(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(5), Size: 100, IsDir: false},
	}
	model := NewManageModel(items, false, nil)
	cmd := model.Init()

	assert.Nil(t, cmd)
}

func TestManageModelSelectedItem(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(2), Size: 100, IsDir: false},
		{Name: "file2.txt", OriginalPath: "/path/to/file2.txt", DeletionDate: makeTime(1), Size: 200, IsDir: false},
	}

	t.Run("valid selection", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.selected = 1

		item := model.SelectedItem()

		require.NotNil(t, item)
		assert.Equal(t, "file2.txt", item.Name)
	})

	t.Run("no items", func(t *testing.T) {
		model := NewManageModel([]trash.TrashItem{}, false, nil)

		item := model.SelectedItem()

		assert.Nil(t, item)
	})

	t.Run("selection out of bounds", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.selected = 10

		item := model.SelectedItem()

		assert.Nil(t, item)
	})
}

func TestManageModelConfirmed(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(8), Size: 100, IsDir: false},
	}
	model := NewManageModel(items, false, nil)
	assert.False(t, model.Confirmed())

	model.confirmed = true
	assert.True(t, model.Confirmed())
}

func TestManageModelCancelled(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(9), Size: 100, IsDir: false},
	}
	model := NewManageModel(items, false, nil)
	assert.False(t, model.Cancelled())

	model.cancelled = true
	assert.True(t, model.Cancelled())
}

func TestManageModelForce(t *testing.T) {
	model := NewManageModel([]trash.TrashItem{}, true, nil)
	assert.True(t, model.Force())

	model2 := NewManageModel([]trash.TrashItem{}, false, nil)
	assert.False(t, model2.Force())
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		query    string
		expected bool
	}{
		{
			name:     "empty query matches everything",
			s:        "anything",
			query:    "",
			expected: true,
		},
		{
			name:     "exact match",
			s:        "hello",
			query:    "hello",
			expected: true,
		},
		{
			name:     "prefix match",
			s:        "hello world",
			query:    "hel",
			expected: true,
		},
		{
			name:     "characters in order",
			s:        "hello world",
			query:    "hwd",
			expected: true,
		},
		{
			name:     "characters not in order",
			s:        "hello world",
			query:    "dh",
			expected: false,
		},
		{
			name:     "case sensitive match",
			s:        "Hello",
			query:    "h",
			expected: false,
		},
		{
			name:     "character not in string",
			s:        "hello",
			query:    "z",
			expected: false,
		},
		{
			name:     "partial match then fail",
			s:        "hello",
			query:    "hex",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fuzzyMatch(tt.s, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterItems(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(3), Size: 100, IsDir: false},
		{Name: "document.pdf", OriginalPath: "/path/to/document.pdf", DeletionDate: makeTime(2), Size: 200, IsDir: false},
		{Name: "file2.txt", OriginalPath: "/path/to/file2.txt", DeletionDate: makeTime(1), Size: 300, IsDir: false},
	}

	t.Run("empty query shows all", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.search.SetValue("")
		model.filterItems()

		assert.Len(t, model.filtered, 3)
	})

	t.Run("filter by name", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.search.SetValue("file")
		model.filterItems()

		require.Len(t, model.filtered, 2)
		assert.Equal(t, "file1.txt", model.filtered[0].Name)
		assert.Equal(t, "file2.txt", model.filtered[1].Name)
	})

	t.Run("filter by path", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.search.SetValue("document")
		model.filterItems()

		require.Len(t, model.filtered, 1)
		assert.Equal(t, "document.pdf", model.filtered[0].Name)
	})

	t.Run("no matches", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.search.SetValue("nonexistent")
		model.filterItems()

		assert.Empty(t, model.filtered)
	})

	t.Run("fuzzy match", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.search.SetValue("f1t") // matches "file1.txt"
		model.filterItems()

		require.Len(t, model.filtered, 1)
		assert.Equal(t, "file1.txt", model.filtered[0].Name)
	})

	t.Run("cyrillic match", func(t *testing.T) {
		cyrillicItems := []trash.TrashItem{
			{Name: "документ.txt", OriginalPath: "/path/документ.txt", DeletionDate: makeTime(13), Size: 100, IsDir: false},
		}
		model := NewManageModel(cyrillicItems, false, nil)
		model.search.SetValue("ок")
		model.filterItems()

		require.Len(t, model.filtered, 1)
		assert.Equal(t, "документ.txt", model.filtered[0].Name)
	})
}

func TestFilterItemsResetsSelection(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(14), Size: 100, IsDir: false},
		{Name: "document.pdf", OriginalPath: "/path/to/document.pdf", DeletionDate: makeTime(15), Size: 200, IsDir: false},
	}

	model := NewManageModel(items, false, nil)
	model.selected = 1
	model.search.SetValue("file")
	model.filterItems()

	// After filtering, selection reset is handled by Update(), not filterItems()
	// The filterItems() function only filters, it doesn't modify selection
	assert.Len(t, model.filtered, 1)
	assert.Equal(t, "file1.txt", model.filtered[0].Name)
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{0, 10, 0},
		{-1, 1, -1},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		assert.Equal(t, tt.expected, result)
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{1, 2, 2},
		{2, 1, 2},
		{5, 5, 5},
		{0, 10, 10},
		{-1, 1, 1},
	}

	for _, tt := range tests {
		result := max(tt.a, tt.b)
		assert.Equal(t, tt.expected, result)
	}
}

func TestManageModelView(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file.txt", OriginalPath: "/path/to/file.txt", DeletionDate: makeTime(16), Size: 100, IsDir: false, TrashDir: "/mnt/usb/.Trash-1000"},
	}

	t.Run("shows items", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		view := model.View()

		assert.Contains(t, view, "Manage Trash")
		assert.Contains(t, view, "file.txt")
		assert.Contains(t, view, "/path/to/file.txt")
		assert.Contains(t, view, "Volume:")
		assert.Contains(t, view, "/mnt/usb")
	})

	t.Run("shows no matching files", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.search.SetValue("nonexistent")
		model.filterItems()
		view := model.View()

		assert.Contains(t, view, "No matching files")
	})

	t.Run("shows directory indicator", func(t *testing.T) {
		dirItems := []trash.TrashItem{
			{Name: "mydir", OriginalPath: "/path/to/mydir", DeletionDate: makeTime(17), Size: 0, IsDir: true},
		}
		model := NewManageModel(dirItems, false, nil)
		view := model.View()

		assert.Contains(t, view, "mydir")
	})
}

func TestManageModelUpdate(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(3), Size: 100, IsDir: false},
		{Name: "file2.txt", OriginalPath: "/path/to/file2.txt", DeletionDate: makeTime(2), Size: 200, IsDir: false},
		{Name: "file3.txt", OriginalPath: "/path/to/file3.txt", DeletionDate: makeTime(1), Size: 300, IsDir: false},
	}

	t.Run("/ enters search mode", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		assert.Equal(t, modeSelect, model.mode)

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
		m := updatedModel.(ManageModel)
		assert.Equal(t, modeSearch, m.mode)
		assert.True(t, m.search.Focused())
	})

	t.Run("Space toggles selection", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		assert.Equal(t, modeSelect, model.mode)
		key := itemKey(items[0])
		assert.False(t, model.selectedItems[key])

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
		m := updatedModel.(ManageModel)
		assert.Equal(t, modeSelect, m.mode)
		assert.True(t, m.selectedItems[key])
	})

	t.Run("Space deselects item", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		key := itemKey(items[0])
		model.selectedItems[key] = true

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
		m := updatedModel.(ManageModel)
		assert.False(t, m.selectedItems[key])
	})

	t.Run("Space selects multiple items", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		key0 := itemKey(items[0])
		model.selectedItems[key0] = true
		model.selected = 1

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
		m := updatedModel.(ManageModel)
		key1 := itemKey(items[1])
		assert.True(t, m.selectedItems[key0])
		assert.True(t, m.selectedItems[key1])
	})

	t.Run("SelectedItems returns selected", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.selectedItems[itemKey(items[0])] = true
		model.selectedItems[itemKey(items[1])] = true

		selectedItems := model.SelectedItems()
		require.Len(t, selectedItems, 2)
		assert.Equal(t, "file1.txt", selectedItems[0].Name)
		assert.Equal(t, "file2.txt", selectedItems[1].Name)
	})

	t.Run("SelectedItems returns single item when nothing selected", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.selected = 1

		items := model.SelectedItems()
		require.Len(t, items, 1)
		assert.Equal(t, "file2.txt", items[0].Name)
	})

	t.Run("SelectedItems preserves selection across filter", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.selectedItems[itemKey(items[0])] = true
		model.selectedItems[itemKey(items[2])] = true

		model.mode = modeSearch
		model.search.Focus()
		model.search.SetValue("file1")
		model.filterItems()

		require.Len(t, model.filtered, 1)

		selected := model.SelectedItems()
		require.Len(t, selected, 2)
		names := []string{selected[0].Name, selected[1].Name}
		assert.Contains(t, names, "file1.txt")
		assert.Contains(t, names, "file3.txt")
	})

	t.Run("Search input in search mode", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeSearch
		model.search.Focus()

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		m := updatedModel.(ManageModel)
		assert.Equal(t, "f", m.search.Value())
		assert.Len(t, m.filtered, 3)
	})

	t.Run("Backspace in search mode", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeSearch
		model.search.Focus()
		model.search.SetValue("test")

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		m := updatedModel.(ManageModel)
		assert.Equal(t, "tes", m.search.Value())
	})

	t.Run("Backspace in search mode", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeSearch
		model.search.Focus()
		model.search.SetValue("test")

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		m := updatedModel.(ManageModel)
		assert.Equal(t, "tes", m.search.Value())
	})

	t.Run("Non-key message returns unchanged", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		updatedModel, cmd := model.Update(nil)

		m := updatedModel.(ManageModel)
		assert.Equal(t, model, m)
		assert.Nil(t, cmd)
	})

	t.Run("Esc cancels in select mode", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})

		m := updatedModel.(ManageModel)
		assert.True(t, m.Cancelled())
		assert.False(t, m.Confirmed())
		assert.NotNil(t, cmd)
	})

	t.Run("Esc cancels", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})

		m := updatedModel.(ManageModel)
		assert.True(t, m.Cancelled())
		assert.False(t, m.Confirmed())
		assert.NotNil(t, cmd)
	})

	t.Run("Enter toggles selection at cursor", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		key := itemKey(items[0])
		assert.False(t, model.selectedItems[key])

		updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m := updatedModel.(ManageModel)
		assert.True(t, m.selectedItems[key])
		assert.Nil(t, cmd)
	})

	t.Run("Enter does nothing without items", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

		m := updatedModel.(ManageModel)
		assert.False(t, m.Confirmed())
		assert.False(t, m.Cancelled())
		assert.Nil(t, cmd)
	})

	t.Run("Arrow up navigates", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.selected = 1

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyUp})
		m := updatedModel.(ManageModel)
		assert.Equal(t, 0, m.selected)
	})

	t.Run("Arrow up at top wraps to bottom", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.selected = 0

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyUp})
		m := updatedModel.(ManageModel)
		assert.Equal(t, 2, m.selected)
	})

	t.Run("Arrow down navigates", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.selected = 0

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		m := updatedModel.(ManageModel)
		assert.Equal(t, 1, m.selected)
	})

	t.Run("Arrow down at bottom wraps to top", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.selected = 2

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		m := updatedModel.(ManageModel)
		assert.Equal(t, 0, m.selected)
	})

	t.Run("Ctrl+P navigates up", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.selected = 2

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
		m := updatedModel.(ManageModel)
		assert.Equal(t, 1, m.selected)
	})

	t.Run("Ctrl+N navigates down", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.selected = 0

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
		m := updatedModel.(ManageModel)
		assert.Equal(t, 1, m.selected)
	})

	t.Run("Search input character", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeSearch
		model.search.Focus()

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		m := updatedModel.(ManageModel)
		assert.Equal(t, "f", m.search.Value())
		assert.Len(t, m.filtered, 3) // All files contain 'f'
	})

	t.Run("Search input filters items", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeSearch
		model.search.Focus()

		// Type "file1"
		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
		updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
		updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})

		m := updatedModel.(ManageModel)
		assert.Equal(t, "file1", m.search.Value())
		require.Len(t, m.filtered, 1)
		assert.Equal(t, "file1.txt", m.filtered[0].Name)
	})

	t.Run("Backspace removes character", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeSearch
		model.search.Focus()
		model.search.SetValue("test")

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		m := updatedModel.(ManageModel)
		assert.Equal(t, "tes", m.search.Value())
	})

	t.Run("Delete key triggers update", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeSearch
		model.search.Focus()
		model.search.SetValue("test")

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyDelete})
		m := updatedModel.(ManageModel)
		// Delete triggers search update and filtering
		assert.Equal(t, "test", m.search.Value())
	})

	t.Run("Selection resets when filtered list shrinks", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeSearch
		model.search.Focus()
		model.selected = 2 // Select last item

		// Search for "file1" which filters to 1 item
		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
		updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
		updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})

		m := updatedModel.(ManageModel)
		assert.Equal(t, 0, m.selected) // Should reset to 0 since only 1 item
	})

	t.Run("Selection adjusts when filtering", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeSearch
		model.search.Focus()
		model.selected = 2

		// Search for "file" which filters to all 3 items
		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		m := updatedModel.(ManageModel)
		assert.Equal(t, 2, m.selected) // Should stay at 2
	})

	t.Run("Non-key message returns unchanged", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		updatedModel, cmd := model.Update(nil)

		m := updatedModel.(ManageModel)
		assert.Equal(t, model, m)
		assert.Nil(t, cmd)
	})
}

func TestUpdateConfirmMode(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(21), Size: 100, IsDir: false},
		{Name: "file2.txt", OriginalPath: "/path/to/file2.txt", DeletionDate: makeTime(22), Size: 200, IsDir: false},
	}

	t.Run("Escape cancels confirmation", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeConfirm
		model.action = ActionRestore

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m := updatedModel.(ManageModel)
		assert.Equal(t, modeSelect, m.mode)
	})

	t.Run("Ctrl+C cancels confirmation", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeConfirm
		model.action = ActionRestore

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		m := updatedModel.(ManageModel)
		assert.Equal(t, modeSelect, m.mode)
	})

	t.Run("Enter confirms action", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeConfirm
		model.action = ActionRestore
		model.selectedItems[itemKey(items[0])] = true

		updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m := updatedModel.(ManageModel)
		assert.Equal(t, modeProgress, m.mode)
		assert.Empty(t, m.results)
		require.NotNil(t, cmd)

		updatedModel, cmd = m.Update(cmd())
		m = updatedModel.(ManageModel)
		assert.Equal(t, modeResults, m.mode)
		assert.Len(t, m.results, 1)
		assert.Nil(t, cmd)
	})

	t.Run("y confirms action", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeConfirm
		model.action = ActionRestore
		model.selectedItems[itemKey(items[0])] = true

		updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		m := updatedModel.(ManageModel)
		assert.Equal(t, modeProgress, m.mode)
		assert.NotNil(t, cmd)
	})

	t.Run("Y confirms action", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeConfirm
		model.action = ActionRestore
		model.selectedItems[itemKey(items[0])] = true

		updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
		m := updatedModel.(ManageModel)
		assert.Equal(t, modeProgress, m.mode)
		assert.NotNil(t, cmd)
	})

	t.Run("n cancels action", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeConfirm
		model.action = ActionRestore

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		m := updatedModel.(ManageModel)
		assert.Equal(t, modeSelect, m.mode)
	})

	t.Run("N cancels action", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeConfirm
		model.action = ActionRestore

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
		m := updatedModel.(ManageModel)
		assert.Equal(t, modeSelect, m.mode)
	})

	t.Run("unknown key does nothing", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeConfirm
		model.action = ActionRestore

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		m := updatedModel.(ManageModel)
		assert.Equal(t, modeConfirm, m.mode)
	})
}

func TestUpdateResultsMode(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(23), Size: 100, IsDir: false},
	}

	t.Run("Enter exits", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeResults

		updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m := updatedModel.(ManageModel)
		assert.True(t, m.confirmed)
		assert.NotNil(t, cmd)
	})

	t.Run("Escape exits", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeResults

		updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m := updatedModel.(ManageModel)
		assert.True(t, m.confirmed)
		assert.NotNil(t, cmd)
	})

	t.Run("Ctrl+C exits", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeResults

		updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		m := updatedModel.(ManageModel)
		assert.True(t, m.confirmed)
		assert.NotNil(t, cmd)
	})

	t.Run("q exits", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeResults

		updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		m := updatedModel.(ManageModel)
		assert.True(t, m.confirmed)
		assert.NotNil(t, cmd)
	})

	t.Run("Q exits", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeResults

		updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Q'}})
		m := updatedModel.(ManageModel)
		assert.True(t, m.confirmed)
		assert.NotNil(t, cmd)
	})

	t.Run("unknown key does nothing", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeResults

		updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		m := updatedModel.(ManageModel)
		assert.False(t, m.confirmed)
		assert.Nil(t, cmd)
	})

	t.Run("non-key message does nothing", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeResults

		updatedModel, cmd := model.Update(nil)
		m := updatedModel.(ManageModel)
		assert.False(t, m.confirmed)
		assert.Nil(t, cmd)
	})
}

func TestUpdateProgressMode(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(23), Size: 100, IsDir: false},
		{Name: "file2.txt", OriginalPath: "/path/to/file2.txt", DeletionDate: makeTime(24), Size: 200, IsDir: false},
	}

	model := NewManageModel(items, false, nil)
	model.mode = modeConfirm
	model.action = ActionDelete
	model.selectedItems[itemKey(items[0])] = true
	model.selectedItems[itemKey(items[1])] = true

	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m := updatedModel.(ManageModel)
	require.Equal(t, modeProgress, m.mode)
	require.NotNil(t, cmd)
	assert.Contains(t, m.View(), "Permanently deleting")
	assert.Contains(t, m.View(), "0 of 2 item(s)")
	assert.Contains(t, m.View(), "Processing: "+m.pendingItems[0].Name)

	updatedModel, cmd = m.Update(cmd())
	m = updatedModel.(ManageModel)
	require.Equal(t, modeProgress, m.mode)
	require.NotNil(t, cmd)
	assert.Len(t, m.results, 1)
	assert.Contains(t, m.View(), "50%")
	assert.Contains(t, m.View(), "Processing: "+m.pendingItems[1].Name)

	updatedModel, cmd = m.Update(cmd())
	m = updatedModel.(ManageModel)
	assert.Equal(t, modeResults, m.mode)
	assert.Len(t, m.results, 2)
	assert.Nil(t, cmd)
}

func TestProgressIgnoresUnexpectedResult(t *testing.T) {
	model := NewManageModel(nil, false, nil)
	model.mode = modeSelect
	msg := actionFinishedMsg{result: ActionResult{}}

	updatedModel, cmd := model.Update(msg)
	m := updatedModel.(ManageModel)
	assert.Equal(t, modeSelect, m.mode)
	assert.Empty(t, m.results)
	assert.Nil(t, cmd)
}

func TestPerformAction(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(24), Size: 100, IsDir: false},
		{Name: "file2.txt", OriginalPath: "/path/to/file2.txt", DeletionDate: makeTime(25), Size: 200, IsDir: true},
	}

	t.Run("execute with nil manager", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.action = ActionRestore

		result := model.performAction(items[0])

		assert.False(t, result.Success)
		assert.Equal(t, items[0], result.Item)
	})

	t.Run("execute restore action", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.action = ActionRestore

		result := model.performAction(items[0])

		assert.Equal(t, ActionRestore, result.Action)
	})

	t.Run("execute delete action", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.action = ActionDelete

		result := model.performAction(items[0])

		assert.Equal(t, ActionDelete, result.Action)
	})
}

func TestPerformActionWithManager(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpHome+"/.local/share")
	t.Setenv("HOME", tmpHome)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := tmpDir + "/test_file.txt"
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))

	require.NoError(t, mgr.Move(file))

	items, err := mgr.List()
	require.NoError(t, err)
	require.Len(t, items, 1)

	t.Run("restore with manager", func(t *testing.T) {
		model := NewManageModel(items, false, mgr)
		model.action = ActionRestore

		result := model.performAction(items[0])

		assert.True(t, result.Success)
		assert.NoError(t, result.Error)
	})

	t.Run("delete with manager", func(t *testing.T) {
		file2 := tmpDir + "/test_file2.txt"
		require.NoError(t, os.WriteFile(file2, []byte("content2"), 0644))
		require.NoError(t, mgr.Move(file2))

		items2, err := mgr.List()
		require.NoError(t, err)
		require.Len(t, items2, 1)

		model := NewManageModel(items2, false, mgr)
		model.action = ActionDelete

		result := model.performAction(items2[0])

		assert.True(t, result.Success)
		assert.NoError(t, result.Error)
	})
}

func TestViewConfirm(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(26), Size: 100, IsDir: false},
		{Name: "file2.txt", OriginalPath: "/path/to/file2.txt", DeletionDate: makeTime(27), Size: 200, IsDir: true},
	}

	t.Run("restore confirmation", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeConfirm
		model.action = ActionRestore
		model.selectedItems[itemKey(items[0])] = true

		view := model.viewConfirm()

		assert.Contains(t, view, "Confirm restore?")
		assert.Contains(t, view, "restore 1 item")
		assert.Contains(t, view, "file1.txt")
		assert.Contains(t, view, "y/Enter")
	})

	t.Run("delete confirmation", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeConfirm
		model.action = ActionDelete
		model.selectedItems[itemKey(items[0])] = true

		view := model.viewConfirm()

		assert.Contains(t, view, "Confirm permanently delete?")
		assert.Contains(t, view, "permanently delete")
		assert.Contains(t, view, "file1.txt")
	})

	t.Run("multiple items", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeConfirm
		model.action = ActionRestore
		model.selectedItems[itemKey(items[0])] = true
		model.selectedItems[itemKey(items[1])] = true

		view := model.viewConfirm()

		assert.Contains(t, view, "2 item")
		assert.Contains(t, view, "file1.txt")
		assert.Contains(t, view, "file2.txt")
	})

	t.Run("more than 10 items", func(t *testing.T) {
		manyItems := make([]trash.TrashItem, 15)
		for i := 0; i < 15; i++ {
			manyItems[i] = trash.TrashItem{
				Name:         fmt.Sprintf("file%d.txt", i),
				OriginalPath: fmt.Sprintf("/path/to/file%d.txt", i),
				DeletionDate: makeTime(28),
				Size:         100,
				IsDir:        false,
			}
		}

		model := NewManageModel(manyItems, false, nil)
		model.mode = modeConfirm
		model.action = ActionRestore
		for _, item := range manyItems {
			model.selectedItems[itemKey(item)] = true
		}

		view := model.viewConfirm()

		assert.Contains(t, view, "15 item")
		assert.Contains(t, view, "... and 5 more")
	})

	t.Run("directory indicator", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeConfirm
		model.action = ActionRestore
		model.selectedItems[itemKey(items[1])] = true

		view := model.viewConfirm()

		assert.Contains(t, view, "file2.txt")
	})
}

func TestViewResults(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(29), Size: 100, IsDir: false},
		{Name: "file2.txt", OriginalPath: "/path/to/file2.txt", DeletionDate: makeTime(30), Size: 200, IsDir: true},
	}

	t.Run("successful restore results", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeResults
		model.action = ActionRestore
		model.results = []ActionResult{
			{Item: items[0], Action: ActionRestore, Success: true, Error: nil},
		}

		view := model.viewResults()

		assert.Contains(t, view, "Restored Results")
		assert.Contains(t, view, "/path/to/file1.txt")
		assert.Contains(t, view, "Successfully restored 1 item")
	})

	t.Run("successful delete results", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeResults
		model.action = ActionDelete
		model.results = []ActionResult{
			{Item: items[0], Action: ActionDelete, Success: true, Error: nil},
		}

		view := model.viewResults()

		assert.Contains(t, view, "Deleted Results")
		assert.Contains(t, view, "Successfully deleted 1 item")
	})

	t.Run("failed results", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeResults
		model.action = ActionRestore
		model.results = []ActionResult{
			{Item: items[0], Action: ActionRestore, Success: false, Error: fmt.Errorf("test error")},
		}

		view := model.viewResults()

		assert.Contains(t, view, "0 succeeded, 1 failed")
		assert.Contains(t, view, "test error")
	})

	t.Run("mixed results", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeResults
		model.action = ActionRestore
		model.results = []ActionResult{
			{Item: items[0], Action: ActionRestore, Success: true, Error: nil},
			{Item: items[1], Action: ActionRestore, Success: false, Error: fmt.Errorf("failed")},
		}

		view := model.viewResults()

		assert.Contains(t, view, "1 succeeded, 1 failed")
	})

	t.Run("directory with path display", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeResults
		model.action = ActionRestore
		model.results = []ActionResult{
			{Item: items[1], Action: ActionRestore, Success: true, Error: nil},
		}

		view := model.viewResults()

		assert.Contains(t, view, "/path/to/file2.txt")
	})

	t.Run("item with empty original path uses name", func(t *testing.T) {
		emptyPathItem := trash.TrashItem{
			Name:         "noname.txt",
			OriginalPath: "",
			DeletionDate: makeTime(31),
			Size:         100,
			IsDir:        false,
		}

		model := NewManageModel([]trash.TrashItem{emptyPathItem}, false, nil)
		model.mode = modeResults
		model.action = ActionRestore
		model.results = []ActionResult{
			{Item: emptyPathItem, Action: ActionRestore, Success: true, Error: nil},
		}

		view := model.viewResults()

		assert.Contains(t, view, "noname.txt")
	})

	t.Run("exit prompt", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.mode = modeResults
		model.action = ActionRestore
		model.results = []ActionResult{
			{Item: items[0], Action: ActionRestore, Success: true, Error: nil},
		}

		view := model.viewResults()

		assert.Contains(t, view, "Press Enter or q to exit")
	})
}

func TestResults(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(32), Size: 100, IsDir: false},
	}

	t.Run("returns results", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.results = []ActionResult{
			{Item: items[0], Action: ActionRestore, Success: true, Error: nil},
		}

		results := model.Results()
		require.Len(t, results, 1)
		assert.Equal(t, items[0], results[0].Item)
		assert.True(t, results[0].Success)
	})

	t.Run("returns empty slice when no results", func(t *testing.T) {
		model := NewManageModel(items, false, nil)

		results := model.Results()
		assert.Empty(t, results)
	})
}

func TestAction(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(33), Size: 100, IsDir: false},
	}

	t.Run("returns restore action", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.action = ActionRestore

		assert.Equal(t, ActionRestore, model.Action())
	})

	t.Run("returns delete action", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.action = ActionDelete

		assert.Equal(t, ActionDelete, model.Action())
	})

	t.Run("default action is zero", func(t *testing.T) {
		model := NewManageModel(items, false, nil)

		assert.Equal(t, ActionType(0), model.Action())
	})
}

func TestManageModelUpdateConfirmWithR(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(34), Size: 100, IsDir: false},
	}

	model := NewManageModel(items, false, nil)
	model.mode = modeConfirm
	model.action = ActionRestore

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m := updatedModel.(ManageModel)
	assert.Equal(t, modeConfirm, m.mode)
}

func TestManageModelUpdateResultsNonKeyMsg(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(35), Size: 100, IsDir: false},
	}

	model := NewManageModel(items, false, nil)
	model.mode = modeResults

	updatedModel, cmd := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := updatedModel.(ManageModel)
	assert.False(t, m.confirmed)
	assert.Nil(t, cmd)
}

func TestSortItems(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "b.txt", OriginalPath: "/path/b.txt", DeletionDate: makeTime(1), Size: 200, IsDir: false, FileCount: 0},
		{Name: "a.txt", OriginalPath: "/path/a.txt", DeletionDate: makeTime(3), Size: 100, IsDir: false, FileCount: 0},
		{Name: "c.txt", OriginalPath: "/path/c.txt", DeletionDate: makeTime(2), Size: 300, IsDir: false, FileCount: 0},
	}

	t.Run("sort by deletion date descending", func(t *testing.T) {
		sorted := sortItems(items, sortByDeletionDate, false)
		require.Len(t, sorted, 3)
		assert.Equal(t, "a.txt", sorted[0].Name) // makeTime(3) newest
		assert.Equal(t, "c.txt", sorted[1].Name) // makeTime(2)
		assert.Equal(t, "b.txt", sorted[2].Name) // makeTime(1) oldest
	})

	t.Run("sort by deletion date ascending", func(t *testing.T) {
		sorted := sortItems(items, sortByDeletionDate, true)
		require.Len(t, sorted, 3)
		assert.Equal(t, "b.txt", sorted[0].Name) // makeTime(1) oldest
		assert.Equal(t, "c.txt", sorted[1].Name) // makeTime(2)
		assert.Equal(t, "a.txt", sorted[2].Name) // makeTime(3) newest
	})

	t.Run("sort by name ascending", func(t *testing.T) {
		sorted := sortItems(items, sortByName, true)
		require.Len(t, sorted, 3)
		assert.Equal(t, "a.txt", sorted[0].Name)
		assert.Equal(t, "b.txt", sorted[1].Name)
		assert.Equal(t, "c.txt", sorted[2].Name)
	})

	t.Run("sort by name descending", func(t *testing.T) {
		sorted := sortItems(items, sortByName, false)
		require.Len(t, sorted, 3)
		assert.Equal(t, "c.txt", sorted[0].Name)
		assert.Equal(t, "b.txt", sorted[1].Name)
		assert.Equal(t, "a.txt", sorted[2].Name)
	})

	t.Run("sort by size ascending", func(t *testing.T) {
		sorted := sortItems(items, sortBySize, true)
		require.Len(t, sorted, 3)
		assert.Equal(t, "a.txt", sorted[0].Name) // 100
		assert.Equal(t, "b.txt", sorted[1].Name) // 200
		assert.Equal(t, "c.txt", sorted[2].Name) // 300
	})

	t.Run("sort by size descending", func(t *testing.T) {
		sorted := sortItems(items, sortBySize, false)
		require.Len(t, sorted, 3)
		assert.Equal(t, "c.txt", sorted[0].Name) // 300
		assert.Equal(t, "b.txt", sorted[1].Name) // 200
		assert.Equal(t, "a.txt", sorted[2].Name) // 100
	})

	t.Run("sort by file count ascending", func(t *testing.T) {
		dirItems := []trash.TrashItem{
			{Name: "big", IsDir: true, FileCount: 100, DeletionDate: makeTime(1)},
			{Name: "small", IsDir: true, FileCount: 5, DeletionDate: makeTime(2)},
			{Name: "medium", IsDir: true, FileCount: 50, DeletionDate: makeTime(3)},
			{Name: "file", IsDir: false, FileCount: 0, DeletionDate: makeTime(4)},
		}
		sorted := sortItems(dirItems, sortByFileCount, true)
		require.Len(t, sorted, 4)
		assert.Equal(t, "file", sorted[0].Name)   // 0
		assert.Equal(t, "small", sorted[1].Name)  // 5
		assert.Equal(t, "medium", sorted[2].Name) // 50
		assert.Equal(t, "big", sorted[3].Name)    // 100
	})

	t.Run("sort by file count descending", func(t *testing.T) {
		dirItems := []trash.TrashItem{
			{Name: "big", IsDir: true, FileCount: 100, DeletionDate: makeTime(1)},
			{Name: "small", IsDir: true, FileCount: 5, DeletionDate: makeTime(2)},
			{Name: "file", IsDir: false, FileCount: 0, DeletionDate: makeTime(3)},
		}
		sorted := sortItems(dirItems, sortByFileCount, false)
		require.Len(t, sorted, 3)
		assert.Equal(t, "big", sorted[0].Name)   // 100
		assert.Equal(t, "small", sorted[1].Name) // 5
		assert.Equal(t, "file", sorted[2].Name)  // 0
	})

	t.Run("sort does not modify original", func(t *testing.T) {
		original := make([]trash.TrashItem, len(items))
		copy(original, items)
		sortItems(items, sortByName, true)
		for i := range items {
			assert.Equal(t, original[i].Name, items[i].Name)
		}
	})
}

func TestSortLabel(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file.txt", OriginalPath: "/path/file.txt", DeletionDate: makeTime(1), Size: 100, IsDir: false},
	}

	t.Run("default sort label", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		label := model.sortLabel()
		assert.Contains(t, label, "Date")
		assert.Contains(t, label, "↓") // descending
	})

	t.Run("name sort ascending", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.sortMode = sortByName
		model.sortAsc = true
		label := model.sortLabel()
		assert.Contains(t, label, "Name")
		assert.Contains(t, label, "↑")
	})

	t.Run("size sort descending", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.sortMode = sortBySize
		model.sortAsc = false
		label := model.sortLabel()
		assert.Contains(t, label, "Size")
		assert.Contains(t, label, "↓")
	})

	t.Run("file count sort ascending", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.sortMode = sortByFileCount
		model.sortAsc = true
		label := model.sortLabel()
		assert.Contains(t, label, "Count")
		assert.Contains(t, label, "↑")
	})
}

func TestCycleSortMode(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file.txt", OriginalPath: "/path/file.txt", DeletionDate: makeTime(1), Size: 100, IsDir: false},
	}

	t.Run("cycles through all sort modes", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		assert.Equal(t, sortByDeletionDate, model.sortMode)
		assert.False(t, model.sortAsc) // desc

		model.cycleSortMode()
		assert.Equal(t, sortByName, model.sortMode)
		assert.True(t, model.sortAsc) // asc

		model.cycleSortMode()
		assert.Equal(t, sortBySize, model.sortMode)
		assert.False(t, model.sortAsc) // desc

		model.cycleSortMode()
		assert.Equal(t, sortByFileCount, model.sortMode)
		assert.False(t, model.sortAsc) // desc

		model.cycleSortMode()
		assert.Equal(t, sortByDeletionDate, model.sortMode)
		assert.False(t, model.sortAsc) // desc, wraps around
	})

	t.Run("resets selection to 0", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.selected = 5
		model.cycleSortMode()
		assert.Equal(t, 0, model.selected)
	})
}

func TestToggleSortDirection(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file.txt", OriginalPath: "/path/file.txt", DeletionDate: makeTime(1), Size: 100, IsDir: false},
	}

	t.Run("toggles direction", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		assert.False(t, model.sortAsc)

		model.toggleSortDirection()
		assert.True(t, model.sortAsc)

		model.toggleSortDirection()
		assert.False(t, model.sortAsc)
	})

	t.Run("resets selection to 0", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		model.selected = 5
		model.toggleSortDirection()
		assert.Equal(t, 0, model.selected)
	})
}

func TestSortKeyBindings(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: makeTime(3), Size: 100, IsDir: false},
		{Name: "file2.txt", OriginalPath: "/path/to/file2.txt", DeletionDate: makeTime(2), Size: 200, IsDir: false},
		{Name: "file3.txt", OriginalPath: "/path/to/file3.txt", DeletionDate: makeTime(1), Size: 300, IsDir: false},
	}

	t.Run("s key cycles sort mode in select mode", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		assert.Equal(t, sortByDeletionDate, model.sortMode)

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		m := updatedModel.(ManageModel)
		assert.Equal(t, sortByName, m.sortMode)
		assert.Equal(t, 0, m.selected)
	})

	t.Run("S key toggles sort direction in select mode", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		assert.False(t, model.sortAsc)

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
		m := updatedModel.(ManageModel)
		assert.True(t, m.sortAsc)
	})

	t.Run("sort persists across filter", func(t *testing.T) {
		model := NewManageModel(items, false, nil)

		// Switch to sort by name
		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		model = updatedModel.(ManageModel)

		// Enter search and filter
		updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
		model = updatedModel.(ManageModel)
		model.search.SetValue("file")
		model.filterItems()

		assert.Equal(t, sortByName, model.sortMode)
	})
}

func TestSortInView(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file.txt", OriginalPath: "/path/file.txt", DeletionDate: makeTime(1), Size: 100, IsDir: false},
	}

	t.Run("view contains sort indicator", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		view := model.View()
		assert.Contains(t, view, "Sort: Date ↓")
	})

	t.Run("view contains sort keybinding hint", func(t *testing.T) {
		model := NewManageModel(items, false, nil)
		view := model.View()
		assert.Contains(t, view, "s/S sort")
	})
}

func TestDefaultSortIsDeletionDateDesc(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "old.txt", OriginalPath: "/path/old.txt", DeletionDate: makeTime(1), Size: 100, IsDir: false},
		{Name: "new.txt", OriginalPath: "/path/new.txt", DeletionDate: makeTime(5), Size: 100, IsDir: false},
		{Name: "mid.txt", OriginalPath: "/path/mid.txt", DeletionDate: makeTime(3), Size: 100, IsDir: false},
	}

	model := NewManageModel(items, false, nil)

	require.Len(t, model.items, 3)
	assert.Equal(t, "new.txt", model.items[0].Name) // newest first
	assert.Equal(t, "mid.txt", model.items[1].Name)
	assert.Equal(t, "old.txt", model.items[2].Name) // oldest last
}
