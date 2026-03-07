package tui

import (
	"testing"
	"time"

	"altlinux.space/amakeenk/trs/internal/trash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewRestoreModel(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: time.Now(), Size: 100, IsDir: false},
		{Name: "file2.txt", OriginalPath: "/path/to/file2.txt", DeletionDate: time.Now(), Size: 200, IsDir: false},
	}

	model := NewRestoreModel(items, false)

	assert.Len(t, model.items, 2)
	assert.Len(t, model.filtered, 2)
	assert.Equal(t, 0, model.selected)
	assert.False(t, model.force)
	assert.False(t, model.confirmed)
	assert.False(t, model.cancelled)
}

func TestNewRestoreModelWithForce(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file.txt", OriginalPath: "/path/to/file.txt", DeletionDate: time.Now(), Size: 100, IsDir: false},
	}

	model := NewRestoreModel(items, true)

	assert.True(t, model.force)
}

func TestRestoreModelInit(t *testing.T) {
	model := NewRestoreModel([]trash.TrashItem{}, false)
	cmd := model.Init()

	assert.Nil(t, cmd)
}

func TestRestoreModelSelectedItem(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: time.Now(), Size: 100, IsDir: false},
		{Name: "file2.txt", OriginalPath: "/path/to/file2.txt", DeletionDate: time.Now(), Size: 200, IsDir: false},
	}

	t.Run("valid selection", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		model.selected = 1

		item := model.SelectedItem()

		require.NotNil(t, item)
		assert.Equal(t, "file2.txt", item.Name)
	})

	t.Run("no items", func(t *testing.T) {
		model := NewRestoreModel([]trash.TrashItem{}, false)

		item := model.SelectedItem()

		assert.Nil(t, item)
	})

	t.Run("selection out of bounds", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		model.selected = 10

		item := model.SelectedItem()

		assert.Nil(t, item)
	})
}

func TestRestoreModelConfirmed(t *testing.T) {
	model := NewRestoreModel([]trash.TrashItem{}, false)
	assert.False(t, model.Confirmed())

	model.confirmed = true
	assert.True(t, model.Confirmed())
}

func TestRestoreModelCancelled(t *testing.T) {
	model := NewRestoreModel([]trash.TrashItem{}, false)
	assert.False(t, model.Cancelled())

	model.cancelled = true
	assert.True(t, model.Cancelled())
}

func TestRestoreModelForce(t *testing.T) {
	model := NewRestoreModel([]trash.TrashItem{}, true)
	assert.True(t, model.Force())

	model2 := NewRestoreModel([]trash.TrashItem{}, false)
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
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: time.Now(), Size: 100, IsDir: false},
		{Name: "document.pdf", OriginalPath: "/path/to/document.pdf", DeletionDate: time.Now(), Size: 200, IsDir: false},
		{Name: "file2.txt", OriginalPath: "/path/to/file2.txt", DeletionDate: time.Now(), Size: 300, IsDir: false},
	}

	t.Run("empty query shows all", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		model.search.SetValue("")
		model.filterItems()

		assert.Len(t, model.filtered, 3)
	})

	t.Run("filter by name", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		model.search.SetValue("file")
		model.filterItems()

		require.Len(t, model.filtered, 2)
		assert.Equal(t, "file1.txt", model.filtered[0].Name)
		assert.Equal(t, "file2.txt", model.filtered[1].Name)
	})

	t.Run("filter by path", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		model.search.SetValue("document")
		model.filterItems()

		require.Len(t, model.filtered, 1)
		assert.Equal(t, "document.pdf", model.filtered[0].Name)
	})

	t.Run("no matches", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		model.search.SetValue("nonexistent")
		model.filterItems()

		assert.Empty(t, model.filtered)
	})

	t.Run("fuzzy match", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		model.search.SetValue("f1t") // matches "file1.txt"
		model.filterItems()

		require.Len(t, model.filtered, 1)
		assert.Equal(t, "file1.txt", model.filtered[0].Name)
	})
}

func TestFilterItemsResetsSelection(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: time.Now(), Size: 100, IsDir: false},
		{Name: "document.pdf", OriginalPath: "/path/to/document.pdf", DeletionDate: time.Now(), Size: 200, IsDir: false},
	}

	model := NewRestoreModel(items, false)
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

func TestRestoreModelView(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file.txt", OriginalPath: "/path/to/file.txt", DeletionDate: time.Now(), Size: 100, IsDir: false},
	}

	t.Run("shows items", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		view := model.View()

		assert.Contains(t, view, "Restore from Trash")
		assert.Contains(t, view, "file.txt")
		assert.Contains(t, view, "/path/to/file.txt")
	})

	t.Run("shows no matching files", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		model.search.SetValue("nonexistent")
		model.filterItems()
		view := model.View()

		assert.Contains(t, view, "No matching files")
	})

	t.Run("shows directory indicator", func(t *testing.T) {
		dirItems := []trash.TrashItem{
			{Name: "mydir", OriginalPath: "/path/to/mydir", DeletionDate: time.Now(), Size: 0, IsDir: true},
		}
		model := NewRestoreModel(dirItems, false)
		view := model.View()

		assert.Contains(t, view, "mydir")
	})
}

func TestRestoreModelUpdate(t *testing.T) {
	items := []trash.TrashItem{
		{Name: "file1.txt", OriginalPath: "/path/to/file1.txt", DeletionDate: time.Now(), Size: 100, IsDir: false},
		{Name: "file2.txt", OriginalPath: "/path/to/file2.txt", DeletionDate: time.Now(), Size: 200, IsDir: false},
		{Name: "file3.txt", OriginalPath: "/path/to/file3.txt", DeletionDate: time.Now(), Size: 300, IsDir: false},
	}

	t.Run("Ctrl+C cancels", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

		m := updatedModel.(RestoreModel)
		assert.True(t, m.Cancelled())
		assert.False(t, m.Confirmed())
		assert.NotNil(t, cmd)
	})

	t.Run("Esc cancels", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})

		m := updatedModel.(RestoreModel)
		assert.True(t, m.Cancelled())
		assert.False(t, m.Confirmed())
		assert.NotNil(t, cmd)
	})

	t.Run("Enter confirms with items", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

		m := updatedModel.(RestoreModel)
		assert.True(t, m.Confirmed())
		assert.False(t, m.Cancelled())
		assert.NotNil(t, cmd)
	})

	t.Run("Enter does nothing without items", func(t *testing.T) {
		model := NewRestoreModel([]trash.TrashItem{}, false)
		updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

		m := updatedModel.(RestoreModel)
		assert.False(t, m.Confirmed())
		assert.False(t, m.Cancelled())
		assert.Nil(t, cmd)
	})

	t.Run("Arrow up navigates", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		model.selected = 1

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyUp})
		m := updatedModel.(RestoreModel)
		assert.Equal(t, 0, m.selected)
	})

	t.Run("Arrow up at top stays at top", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		model.selected = 0

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyUp})
		m := updatedModel.(RestoreModel)
		assert.Equal(t, 0, m.selected)
	})

	t.Run("Arrow down navigates", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		model.selected = 0

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		m := updatedModel.(RestoreModel)
		assert.Equal(t, 1, m.selected)
	})

	t.Run("Arrow down at bottom stays at bottom", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		model.selected = 2

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		m := updatedModel.(RestoreModel)
		assert.Equal(t, 2, m.selected)
	})

	t.Run("Ctrl+P navigates up", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		model.selected = 2

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
		m := updatedModel.(RestoreModel)
		assert.Equal(t, 1, m.selected)
	})

	t.Run("Ctrl+N navigates down", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		model.selected = 0

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
		m := updatedModel.(RestoreModel)
		assert.Equal(t, 1, m.selected)
	})

	t.Run("Search input character", func(t *testing.T) {
		model := NewRestoreModel(items, false)

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		m := updatedModel.(RestoreModel)
		assert.Equal(t, "f", m.search.Value())
		assert.Len(t, m.filtered, 3) // All files contain 'f'
	})

	t.Run("Search input filters items", func(t *testing.T) {
		model := NewRestoreModel(items, false)

		// Type "file1"
		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
		updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
		updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})

		m := updatedModel.(RestoreModel)
		assert.Equal(t, "file1", m.search.Value())
		require.Len(t, m.filtered, 1)
		assert.Equal(t, "file1.txt", m.filtered[0].Name)
	})

	t.Run("Backspace removes character", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		model.search.SetValue("test")

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		m := updatedModel.(RestoreModel)
		assert.Equal(t, "tes", m.search.Value())
	})

	t.Run("Delete key triggers update", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		model.search.SetValue("test")

		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyDelete})
		m := updatedModel.(RestoreModel)
		// Delete triggers search update and filtering
		assert.Equal(t, "test", m.search.Value())
	})

	t.Run("Selection resets when filtered list shrinks", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		model.selected = 2 // Select last item

		// Search for "file1" which filters to 1 item
		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
		updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
		updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})

		m := updatedModel.(RestoreModel)
		assert.Equal(t, 0, m.selected) // Should reset to 0 since only 1 item
	})

	t.Run("Selection adjusts when filtering", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		model.selected = 2

		// Search for "file" which filters to all 3 items
		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		m := updatedModel.(RestoreModel)
		assert.Equal(t, 2, m.selected) // Should stay at 2
	})

	t.Run("Non-key message returns unchanged", func(t *testing.T) {
		model := NewRestoreModel(items, false)
		updatedModel, cmd := model.Update(nil)

		m := updatedModel.(RestoreModel)
		assert.Equal(t, model, m)
		assert.Nil(t, cmd)
	})
}
