package steps

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/teosibileau/streamchop/tui/onvif"
)

func TestGenerateModelCancel(t *testing.T) {
	cameras := []ConfiguredCamera{
		{
			Camera: onvif.Camera{Name: "Cam1", IP: "192.168.1.10"},
			Creds:  onvif.Credentials{Username: "admin", Password: "pass"},
			Stream: onvif.StreamInfo{URI: "rtsp://admin:pass@192.168.1.10:554/stream1"},
		},
	}

	m := NewGenerateModel(cameras, MQTTConfig{Enabled: true, Host: "mqtt", Port: "1883"}, "192.168.1.5")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !m.Done() {
		t.Error("expected done after cancel")
	}
}

func TestGenerateModelConfirmTriggersWrite(t *testing.T) {
	cameras := []ConfiguredCamera{
		{
			Camera: onvif.Camera{Name: "Cam1", IP: "192.168.1.10"},
			Creds:  onvif.Credentials{Username: "admin", Password: "pass"},
			Stream: onvif.StreamInfo{URI: "rtsp://admin:pass@192.168.1.10:554/stream1"},
		},
	}

	m := NewGenerateModel(cameras, MQTTConfig{Enabled: true, Host: "mqtt", Port: "1883"}, "192.168.1.5")

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if m.state != generateWriting {
		t.Error("expected state to be generateWriting after confirm")
	}
	if cmd == nil {
		t.Error("expected a command to be returned for file generation")
	}
}

func TestGenerateModelHandlesError(t *testing.T) {
	m := GenerateModel{state: generateWriting}

	m, _ = m.Update(generateDoneMsg{err: fmt.Errorf("write failed")})
	if m.state != generateError {
		t.Error("expected error state")
	}
	if m.Done() {
		t.Error("should not be done on error")
	}
}
