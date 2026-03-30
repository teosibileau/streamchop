package compose

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// CameraConfig holds the info needed to generate a chopper service.
type CameraConfig struct {
	Index   int
	EnvVar  string // e.g. CAM1_RTSP_URL
	RTSPURL string
}

// ComposeFile represents the docker-compose.dist.yml structure.
type ComposeFile struct {
	Services map[string]Service `yaml:"services"`
}

// Service represents a docker-compose service.
type Service struct {
	Image       string   `yaml:"image"`
	Container   string   `yaml:"container_name"`
	Environment []string `yaml:"environment,omitempty"`
	Volumes     []string `yaml:"volumes,omitempty"`
	Ports       []string `yaml:"ports,omitempty"`
	DependsOn   []string `yaml:"depends_on,omitempty"`
}

// GenerateDistCompose writes docker-compose.dist.yml with one chopper per camera
// plus shared emitter and nginx services.
func GenerateDistCompose(path string, cameras []CameraConfig) error {
	services := make(map[string]Service)

	for _, cam := range cameras {
		name := fmt.Sprintf("chopper_cam_%d", cam.Index)
		services[name] = Service{
			Image:     "${GHCR_REPO}/chopper:${TAG:-latest}",
			Container: fmt.Sprintf("streamchop-chopper-cam%d", cam.Index),
			Environment: []string{
				fmt.Sprintf("RTSP_URL=${%s}", cam.EnvVar),
			},
			Volumes: []string{
				fmt.Sprintf("./output/cam%d:/output", cam.Index),
			},
		}
	}

	services["emitter"] = Service{
		Image:     "${GHCR_REPO}/emitter:${TAG:-latest}",
		Container: "streamchop-emitter",
		Environment: []string{
			"MQTT_HOST=${MQTT_HOST}",
			"MQTT_PORT=${MQTT_PORT:-1883}",
			"MQTT_TOPIC_PREFIX=${MQTT_TOPIC_PREFIX:-streamchop}",
			"HLS_BASE_URL=${HLS_BASE_URL}",
			"WATCH_DIR=/output",
			"RUST_LOG=info",
		},
		Volumes: []string{
			"./output:/output:ro",
		},
	}

	services["nginx"] = Service{
		Image:     "${GHCR_REPO}/nginx:${TAG:-latest}",
		Container: "streamchop-nginx",
		Ports: []string{
			"8080:80",
		},
		Volumes: []string{
			"./output:/usr/share/nginx/html:ro",
		},
	}

	compose := ComposeFile{Services: services}

	data, err := yaml.Marshal(&compose)
	if err != nil {
		return fmt.Errorf("marshal compose: %w", err)
	}

	header := "---\n"
	return os.WriteFile(path, append([]byte(header), data...), 0644)
}
