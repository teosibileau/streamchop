package steps

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/teosibileau/streamchop/tui/onvif"
)

func TestSelectionPreselects(t *testing.T) {
	existing := []string{"192.168.1.10", "192.168.1.30"}
	m := NewSelectionModel(existing)

	cameras := []onvif.Camera{
		{Name: "Cam1", IP: "192.168.1.10", Port: "80", XAddr: "http://192.168.1.10/onvif"},
		{Name: "Cam2", IP: "192.168.1.20", Port: "80", XAddr: "http://192.168.1.20/onvif"},
		{Name: "Cam3", IP: "192.168.1.30", Port: "80", XAddr: "http://192.168.1.30/onvif"},
	}
	m.SetCameras(cameras)

	if !m.selected[0] {
		t.Error("expected camera 0 (192.168.1.10) to be pre-selected")
	}
	if m.selected[1] {
		t.Error("expected camera 1 (192.168.1.20) to NOT be pre-selected")
	}
	if !m.selected[2] {
		t.Error("expected camera 2 (192.168.1.30) to be pre-selected")
	}
}

func TestSelectionToggle(t *testing.T) {
	m := NewSelectionModel(nil)
	m.SetCameras([]onvif.Camera{
		{Name: "Cam1", IP: "192.168.1.10"},
		{Name: "Cam2", IP: "192.168.1.20"},
	})

	// Toggle first camera on
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !m.selected[0] {
		t.Error("expected camera 0 to be selected after toggle")
	}

	// Toggle first camera off
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if m.selected[0] {
		t.Error("expected camera 0 to be deselected after second toggle")
	}
}

func TestSelectionNavigate(t *testing.T) {
	m := NewSelectionModel(nil)
	m.SetCameras([]onvif.Camera{
		{Name: "Cam1", IP: "192.168.1.10"},
		{Name: "Cam2", IP: "192.168.1.20"},
	})

	if m.cursor != 0 {
		t.Fatalf("expected cursor at 0, got %d", m.cursor)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 after j, got %d", m.cursor)
	}

	// Can't go past the end
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.cursor != 1 {
		t.Errorf("expected cursor to stay at 1, got %d", m.cursor)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0 after k, got %d", m.cursor)
	}
}

func TestSelectionRequiresAtLeastOne(t *testing.T) {
	m := NewSelectionModel(nil)
	m.SetCameras([]onvif.Camera{
		{Name: "Cam1", IP: "192.168.1.10"},
	})

	// Try to confirm with nothing selected
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.Done() {
		t.Error("should not be done with no selection")
	}

	// Select and confirm
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.Done() {
		t.Error("should be done after selecting and confirming")
	}

	selected := m.Selected()
	if len(selected) != 1 {
		t.Errorf("expected 1 selected, got %d", len(selected))
	}
}
