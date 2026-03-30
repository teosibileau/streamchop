package steps

import (
	"fmt"
	"testing"

	"github.com/teosibileau/streamchop/tui/onvif"
)

func TestDiscoveryModelTransitionsOnCamerasFound(t *testing.T) {
	m := NewDiscoveryModel()

	cameras := []onvif.Camera{
		{Name: "Cam1", IP: "192.168.1.10", Port: "80", XAddr: "http://192.168.1.10/onvif"},
	}

	m, _ = m.Update(camerasFoundMsg{cameras: cameras})

	if !m.Done() {
		t.Error("expected done after cameras found")
	}
	if len(m.Cameras()) != 1 {
		t.Errorf("expected 1 camera, got %d", len(m.Cameras()))
	}
}

func TestDiscoveryModelFailsOnNoCameras(t *testing.T) {
	m := NewDiscoveryModel()

	m, _ = m.Update(camerasFoundMsg{cameras: nil})

	if m.Done() {
		t.Error("should not be done when no cameras found — should show retry options")
	}
	if m.state != discoveryFailed {
		t.Errorf("expected discoveryFailed state, got %d", m.state)
	}
	if m.Err() == nil {
		t.Error("expected an error")
	}
}

func TestDiscoveryModelFailsOnError(t *testing.T) {
	m := NewDiscoveryModel()

	m, _ = m.Update(camerasFoundMsg{err: fmt.Errorf("network error")})

	if m.state != discoveryFailed {
		t.Errorf("expected discoveryFailed state, got %d", m.state)
	}
	if m.Err().Error() != "network error" {
		t.Errorf("unexpected error: %v", m.Err())
	}
}
