package compose

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGenerateDistCompose(t *testing.T) {
	tmpFile := t.TempDir() + "/docker-compose.dist.yml"

	cameras := []CameraConfig{
		{Index: 1, EnvVar: "CAM1_RTSP_URL", RTSPURL: "rtsp://admin:pass@192.168.1.10:554/stream1"},
		{Index: 2, EnvVar: "CAM2_RTSP_URL", RTSPURL: "rtsp://admin:pass@192.168.1.11:554/stream1"},
	}

	if err := GenerateDistCompose(tmpFile, cameras, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	content := string(data)

	if !strings.HasPrefix(content, "---\n") {
		t.Error("expected YAML document start")
	}

	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, ok := compose.Services["chopper_cam_1"]; !ok {
		t.Error("missing chopper_cam_1 service")
	}
	if _, ok := compose.Services["chopper_cam_2"]; !ok {
		t.Error("missing chopper_cam_2 service")
	}
	if _, ok := compose.Services["emitter"]; !ok {
		t.Error("missing emitter service")
	}
	if _, ok := compose.Services["nginx"]; !ok {
		t.Error("missing nginx service")
	}

	chopper1 := compose.Services["chopper_cam_1"]
	if chopper1.Image != "${GHCR_REPO}/chopper:${TAG:-latest}" {
		t.Errorf("unexpected image: %s", chopper1.Image)
	}
	if chopper1.Container != "streamchop-chopper-cam1" {
		t.Errorf("unexpected container name: %s", chopper1.Container)
	}
	if len(chopper1.Environment) != 1 || chopper1.Environment[0] != "RTSP_URL=${CAM1_RTSP_URL}" {
		t.Errorf("unexpected environment: %v", chopper1.Environment)
	}

	if len(compose.Services) != 4 {
		t.Errorf("expected 4 services, got %d", len(compose.Services))
	}
}

func TestGenerateDistComposeSingleCamera(t *testing.T) {
	tmpFile := t.TempDir() + "/docker-compose.dist.yml"

	cameras := []CameraConfig{
		{Index: 1, EnvVar: "CAM1_RTSP_URL", RTSPURL: "rtsp://a:b@10.0.0.1:554/s"},
	}

	if err := GenerateDistCompose(tmpFile, cameras, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(compose.Services) != 3 {
		t.Errorf("expected 3 services (1 chopper + emitter + nginx), got %d", len(compose.Services))
	}
}

func TestGenerateDistComposeWithoutEmitter(t *testing.T) {
	tmpFile := t.TempDir() + "/docker-compose.dist.yml"

	cameras := []CameraConfig{
		{Index: 1, EnvVar: "CAM1_RTSP_URL", RTSPURL: "rtsp://a:b@10.0.0.1:554/s"},
	}

	if err := GenerateDistCompose(tmpFile, cameras, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, ok := compose.Services["emitter"]; ok {
		t.Error("emitter should not be present when includeEmitter is false")
	}

	if len(compose.Services) != 2 {
		t.Errorf("expected 2 services (1 chopper + nginx), got %d", len(compose.Services))
	}
}
