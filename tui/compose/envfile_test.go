package compose

import (
	"os"
	"strings"
	"testing"
)

func TestGenerateEnv(t *testing.T) {
	tmpFile := t.TempDir() + "/.env"

	config := EnvConfig{
		GHCRRepo: "ghcr.io/teosibileau/streamchop",
		Tag:      "v1.0.0",
		Cameras: []CameraConfig{
			{Index: 1, EnvVar: "CAM1_RTSP_URL", RTSPURL: "rtsp://admin:pass@192.168.1.10:554/stream1"},
			{Index: 2, EnvVar: "CAM2_RTSP_URL", RTSPURL: "rtsp://admin:pass@192.168.1.11:554/stream1"},
		},
	}

	if err := GenerateEnv(tmpFile, config); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	content := string(data)

	mustContain := []string{
		"CAM1_RTSP_URL=rtsp://admin:pass@192.168.1.10:554/stream1",
		"CAM2_RTSP_URL=rtsp://admin:pass@192.168.1.11:554/stream1",
		"GHCR_REPO=ghcr.io/teosibileau/streamchop",
		"TAG=v1.0.0",
		"MQTT_HOST=mqtt",
		"SERVICE_FILE=streamchop.service",
	}

	for _, s := range mustContain {
		if !strings.Contains(content, s) {
			t.Errorf("expected %q in output", s)
		}
	}
}

func TestGenerateEnvPreservesExisting(t *testing.T) {
	tmpFile := t.TempDir() + "/.env"

	existing := `MQTT_HOST=custom-broker
MQTT_PORT=1884
HLS_BASE_URL=http://custom:8080
SERVICE_FILE=custom.service
CAM1_RTSP_URL=rtsp://old:old@192.168.1.99:554/old
`
	if err := os.WriteFile(tmpFile, []byte(existing), 0644); err != nil {
		t.Fatalf("write existing: %v", err)
	}

	config := EnvConfig{
		GHCRRepo: "ghcr.io/teosibileau/streamchop",
		Tag:      "latest",
		Cameras: []CameraConfig{
			{Index: 1, EnvVar: "CAM1_RTSP_URL", RTSPURL: "rtsp://new:new@192.168.1.10:554/new"},
		},
	}

	if err := GenerateEnv(tmpFile, config); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "MQTT_HOST=custom-broker") {
		t.Error("expected preserved MQTT_HOST")
	}
	if !strings.Contains(content, "MQTT_PORT=1884") {
		t.Error("expected preserved MQTT_PORT")
	}
	if !strings.Contains(content, "SERVICE_FILE=custom.service") {
		t.Error("expected preserved SERVICE_FILE")
	}
	if !strings.Contains(content, "CAM1_RTSP_URL=rtsp://new:new@192.168.1.10:554/new") {
		t.Error("expected updated CAM1_RTSP_URL")
	}
	if strings.Contains(content, "192.168.1.99") {
		t.Error("old camera URL should be replaced")
	}
}

func TestParseExistingDist(t *testing.T) {
	dir := t.TempDir()
	envFile := dir + "/.env"

	env := `CAM1_RTSP_URL=rtsp://admin:pass@192.168.1.10:554/stream1
CAM2_RTSP_URL=rtsp://admin:pass@192.168.1.20:554/stream1
MQTT_HOST=mqtt
`
	if err := os.WriteFile(envFile, []byte(env), 0644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	ips := ParseExistingDist(dir+"/docker-compose.dist.yml", envFile)

	if len(ips) != 2 {
		t.Fatalf("expected 2 IPs, got %d", len(ips))
	}

	found := make(map[string]bool)
	for _, ip := range ips {
		found[ip] = true
	}
	if !found["192.168.1.10"] || !found["192.168.1.20"] {
		t.Errorf("unexpected IPs: %v", ips)
	}
}

func TestExtractIPFromRTSPURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"rtsp://admin:pass@192.168.1.10:554/stream1", "192.168.1.10"},
		{"rtsp://192.168.1.20:554/stream1", "192.168.1.20"},
		{"rtsp://user:p@ss@10.0.0.1/s", "10.0.0.1"},
	}

	for _, tt := range tests {
		got := extractIPFromRTSPURL(tt.url)
		if got != tt.want {
			t.Errorf("extractIPFromRTSPURL(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}
