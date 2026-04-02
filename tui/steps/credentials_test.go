package steps

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/teosibileau/streamchop/tui/onvif"
)

func TestCredentialsSameForAll(t *testing.T) {
	cameras := []onvif.Camera{
		{Name: "Cam1", IP: "192.168.1.10"},
		{Name: "Cam2", IP: "192.168.1.20"},
		{Name: "Cam3", IP: "192.168.1.30"},
	}

	m := NewCredentialsModel(cameras, nil)

	// Type username
	for _, r := range "admin" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Tab to password
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})

	// Type password
	for _, r := range "secret" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Toggle same-for-all
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	if !m.sameForAll {
		t.Fatal("expected sameForAll to be true")
	}

	// Confirm
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.Done() {
		t.Fatal("expected done after confirm with same-for-all")
	}

	creds := m.Credentials()
	for i, c := range creds {
		if c.Username != "admin" || c.Password != "secret" {
			t.Errorf("camera %d: expected admin/secret, got %s/%s", i, c.Username, c.Password)
		}
	}
}

func TestCredentialsPerCamera(t *testing.T) {
	cameras := []onvif.Camera{
		{Name: "Cam1", IP: "192.168.1.10"},
		{Name: "Cam2", IP: "192.168.1.20"},
	}

	m := NewCredentialsModel(cameras, nil)

	// First camera: user1/pass1
	for _, r := range "user1" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	for _, r := range "pass1" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.Done() {
		t.Fatal("should not be done after first camera")
	}
	if m.camIndex != 1 {
		t.Fatalf("expected camIndex 1, got %d", m.camIndex)
	}

	// Second camera: user2/pass2
	for _, r := range "user2" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	for _, r := range "pass2" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !m.Done() {
		t.Fatal("expected done after all cameras")
	}

	creds := m.Credentials()
	if creds[0].Username != "user1" || creds[0].Password != "pass1" {
		t.Errorf("cam 0: got %s/%s", creds[0].Username, creds[0].Password)
	}
	if creds[1].Username != "user2" || creds[1].Password != "pass2" {
		t.Errorf("cam 1: got %s/%s", creds[1].Username, creds[1].Password)
	}
}

func TestCredentialsRejectsEmptyUsername(t *testing.T) {
	cameras := []onvif.Camera{
		{Name: "Cam1", IP: "192.168.1.10"},
	}

	m := NewCredentialsModel(cameras, nil)

	// Try to confirm with empty username
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.Done() {
		t.Error("should not accept empty username")
	}
}

func TestCredentialsPrefillFromExisting(t *testing.T) {
	cameras := []onvif.Camera{
		{Name: "Cam1", IP: "192.168.1.10"},
		{Name: "Cam2", IP: "192.168.1.20"},
	}

	existing := map[string][2]string{
		"192.168.1.10": {"olduser", "oldpass"},
		"192.168.1.20": {"admin", "secret"},
	}

	m := NewCredentialsModel(cameras, existing)

	// First camera should be pre-filled
	if m.username.Value() != "olduser" {
		t.Errorf("expected pre-filled username 'olduser', got '%s'", m.username.Value())
	}

	// Confirm first camera with pre-filled values
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.camIndex != 1 {
		t.Fatalf("expected to advance to camera 1, got %d", m.camIndex)
	}

	// Second camera should be pre-filled
	if m.username.Value() != "admin" {
		t.Errorf("expected pre-filled username 'admin', got '%s'", m.username.Value())
	}
}
