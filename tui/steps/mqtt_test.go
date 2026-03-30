package steps

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMQTTModelSkip(t *testing.T) {
	m := NewMQTTModel("192.168.1.5")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !m.Done() {
		t.Fatal("expected done after skip")
	}
	if m.Config().Enabled {
		t.Error("expected MQTT disabled after skip")
	}
}

func TestMQTTModelManualEntry(t *testing.T) {
	m := NewMQTTModel("192.168.1.5")

	// Enter manual mode
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if m.state != mqttManualEntry {
		t.Fatalf("expected manual entry state, got %d", m.state)
	}

	// Type host
	for _, r := range "10.0.0.50" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Tab to port
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})

	// Type port
	for _, r := range "1884" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Confirm
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.Done() {
		t.Fatal("expected done after manual entry confirm")
	}

	config := m.Config()
	if !config.Enabled {
		t.Error("expected MQTT enabled")
	}
	if config.Host != "10.0.0.50" {
		t.Errorf("expected host 10.0.0.50, got %s", config.Host)
	}
	if config.Port != "1884" {
		t.Errorf("expected port 1884, got %s", config.Port)
	}
}

func TestMQTTModelManualEntryDefaultPort(t *testing.T) {
	m := NewMQTTModel("192.168.1.5")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})

	for _, r := range "10.0.0.50" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Confirm without entering port
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.Done() {
		t.Fatal("expected done")
	}
	if m.Config().Port != "1883" {
		t.Errorf("expected default port 1883, got %s", m.Config().Port)
	}
}

func TestMQTTModelManualEntryRejectsEmpty(t *testing.T) {
	m := NewMQTTModel("192.168.1.5")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.Done() {
		t.Error("should not accept empty host")
	}
}

func TestMQTTModelManualEntryEscGoesBack(t *testing.T) {
	m := NewMQTTModel("192.168.1.5")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if m.state != mqttManualEntry {
		t.Fatal("expected manual entry state")
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.state != mqttMenu {
		t.Errorf("expected menu state after esc, got %d", m.state)
	}
}

func TestMQTTModelScanShowsSelection(t *testing.T) {
	m := NewMQTTModel("192.168.1.5")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if m.state != mqttScanning {
		t.Fatalf("expected scanning state, got %d", m.state)
	}

	// Simulate scan result with multiple brokers
	m, _ = m.Update(mqttScanResultMsg{hosts: []string{"192.168.1.100", "192.168.1.200"}})
	if m.state != mqttSelectBroker {
		t.Fatalf("expected select broker state, got %d", m.state)
	}
	if len(m.foundHosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(m.foundHosts))
	}

	// Navigate and select
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1, got %d", m.cursor)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.Done() {
		t.Fatal("expected done after selection")
	}
	if m.Config().Host != "192.168.1.200" {
		t.Errorf("expected host 192.168.1.200, got %s", m.Config().Host)
	}
}

func TestMQTTModelScanSingleBrokerShowsSelection(t *testing.T) {
	m := NewMQTTModel("192.168.1.5")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m, _ = m.Update(mqttScanResultMsg{hosts: []string{"192.168.1.100"}})

	if m.state != mqttSelectBroker {
		t.Fatalf("expected select broker state even with 1 result, got %d", m.state)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.Done() {
		t.Fatal("expected done after selection")
	}
	if m.Config().Host != "192.168.1.100" {
		t.Errorf("expected host 192.168.1.100, got %s", m.Config().Host)
	}
}

func TestMQTTModelScanFailureReturnsToMenu(t *testing.T) {
	m := NewMQTTModel("192.168.1.5")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m, _ = m.Update(mqttScanResultMsg{err: fmt.Errorf("not found")})

	if m.Done() {
		t.Error("should not be done after scan failure")
	}
	if m.state != mqttMenu {
		t.Errorf("expected menu state after scan failure, got %d", m.state)
	}
	if m.err == nil {
		t.Error("expected error to be set")
	}
}
