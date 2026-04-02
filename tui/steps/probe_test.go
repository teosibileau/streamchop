package steps

import (
	"fmt"
	"testing"

	"github.com/teosibileau/streamchop/tui/onvif"
)

func TestProbeModelCollectsResults(t *testing.T) {
	cameras := []onvif.Camera{
		{Name: "Cam1", IP: "192.168.1.10", XAddr: "http://192.168.1.10/onvif"},
		{Name: "Cam2", IP: "192.168.1.20", XAddr: "http://192.168.1.20/onvif"},
	}
	creds := []onvif.Credentials{
		{Username: "admin", Password: "pass"},
		{Username: "admin", Password: "pass"},
	}

	m := NewProbeModel(cameras, creds)

	// Simulate results arriving
	m, _ = m.Update(probeResultMsg{
		index:   0,
		streams: []onvif.StreamInfo{{ProfileToken: "p1", ProfileName: "Main", URI: "rtsp://a"}},
	})
	if m.state != probeRunning {
		t.Error("should still be running with 1 pending")
	}

	m, _ = m.Update(probeResultMsg{
		index:   1,
		streams: []onvif.StreamInfo{{ProfileToken: "p1", ProfileName: "Main", URI: "rtsp://b"}},
	})
	if m.state != probeResults {
		t.Error("should be in results state after all probes complete")
	}

	configured := m.Configured()
	if len(configured) != 2 {
		t.Errorf("expected 2 configured cameras, got %d", len(configured))
	}
}

func TestProbeModelSkipsFailedCameras(t *testing.T) {
	cameras := []onvif.Camera{
		{Name: "Cam1", IP: "192.168.1.10", XAddr: "http://192.168.1.10/onvif"},
		{Name: "Cam2", IP: "192.168.1.20", XAddr: "http://192.168.1.20/onvif"},
	}
	creds := []onvif.Credentials{
		{Username: "admin", Password: "pass"},
		{Username: "admin", Password: "wrong"},
	}

	m := NewProbeModel(cameras, creds)

	m, _ = m.Update(probeResultMsg{
		index:   0,
		streams: []onvif.StreamInfo{{ProfileToken: "p1", ProfileName: "Main", URI: "rtsp://a"}},
	})
	m, _ = m.Update(probeResultMsg{
		index: 1,
		err:   fmt.Errorf("auth failed"),
	})

	configured := m.Configured()
	if len(configured) != 1 {
		t.Errorf("expected 1 configured camera (skipping failed), got %d", len(configured))
	}
}
