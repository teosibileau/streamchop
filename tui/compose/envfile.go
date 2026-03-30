package compose

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// EnvConfig holds the values to write to .env.
type EnvConfig struct {
	GHCRRepo    string
	Tag         string
	Cameras     []CameraConfig
	IncludeMQTT bool
	MQTTHost    string
	MQTTPort    string
	HLSBaseURL  string
}

// GenerateEnv writes or updates the .env file, preserving existing values
// that are not managed by the TUI (MQTT, HLS, SERVICE_FILE).
func GenerateEnv(path string, config EnvConfig) error {
	existing := readExistingEnv(path)

	managed := make(map[string]string)
	managed["GHCR_REPO"] = config.GHCRRepo
	managed["TAG"] = config.Tag
	if config.HLSBaseURL != "" {
		managed["HLS_BASE_URL"] = config.HLSBaseURL
	}
	if config.IncludeMQTT {
		managed["MQTT_HOST"] = config.MQTTHost
		managed["MQTT_PORT"] = config.MQTTPort
	}
	for _, cam := range config.Cameras {
		managed[cam.EnvVar] = cam.RTSPURL
	}

	// Merge: managed values override existing, existing non-managed values are preserved
	for k, v := range managed {
		existing[k] = v
	}

	// Remove old camera entries that are no longer configured
	for k := range existing {
		if strings.HasPrefix(k, "CAM") && strings.HasSuffix(k, "_RTSP_URL") {
			if _, ok := managed[k]; !ok {
				delete(existing, k)
			}
		}
	}

	return writeEnv(path, existing, config.Cameras, config.IncludeMQTT)
}

func readExistingEnv(path string) map[string]string {
	env := make(map[string]string)

	f, err := os.Open(path)
	if err != nil {
		return env
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	return env
}

func writeEnv(path string, env map[string]string, cameras []CameraConfig, includeMQTT bool) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create env file: %w", err)
	}
	defer func() { _ = f.Close() }()

	w := bufio.NewWriter(f)
	p := func(format string, args ...interface{}) {
		_, _ = fmt.Fprintf(w, format, args...)
	}
	ln := func(args ...interface{}) {
		_, _ = fmt.Fprintln(w, args...)
	}

	// Camera RTSP URLs
	ln("# Camera RTSP URLs")
	for _, cam := range cameras {
		p("%s=%s\n", cam.EnvVar, env[cam.EnvVar])
	}

	// MQTT (only if enabled)
	if includeMQTT {
		ln()
		ln("# MQTT broker")
		writeEnvVar(w, env, "MQTT_HOST", "mqtt")
		writeEnvVar(w, env, "MQTT_PORT", "1883")
		writeEnvVar(w, env, "MQTT_TOPIC_PREFIX", "streamchop")
	}

	// HLS
	ln()
	ln("# Base URL for HLS file access")
	writeEnvVar(w, env, "HLS_BASE_URL", "http://nginx:80")

	// GHCR
	ln()
	ln("# GHCR image repository")
	p("GHCR_REPO=%s\n", env["GHCR_REPO"])
	p("TAG=%s\n", env["TAG"])

	// Systemd
	ln()
	ln("# Systemd service file name")
	writeEnvVar(w, env, "SERVICE_FILE", "streamchop.service")

	return w.Flush()
}

func writeEnvVar(w *bufio.Writer, env map[string]string, key, defaultVal string) {
	val, ok := env[key]
	if !ok || val == "" {
		val = defaultVal
	}
	_, _ = fmt.Fprintf(w, "%s=%s\n", key, val)
}

// ParseExistingDist reads an existing docker-compose.dist.yml and .env to
// extract previously configured camera IPs. Used to prepopulate the selection.
func ParseExistingDist(composePath, envPath string) []string {
	env := readExistingEnv(envPath)

	var ips []string
	for k, v := range env {
		if strings.HasPrefix(k, "CAM") && strings.HasSuffix(k, "_RTSP_URL") {
			ip := extractIPFromRTSPURL(v)
			if ip != "" {
				ips = append(ips, ip)
			}
		}
	}

	return ips
}

func extractIPFromRTSPURL(rtspURL string) string {
	// rtsp://user:pass@192.168.1.100:554/stream
	rtspURL = strings.TrimPrefix(rtspURL, "rtsp://")
	if idx := strings.Index(rtspURL, "@"); idx >= 0 {
		rtspURL = rtspURL[idx+1:]
	}
	if idx := strings.Index(rtspURL, ":"); idx >= 0 {
		rtspURL = rtspURL[:idx]
	}
	if idx := strings.Index(rtspURL, "/"); idx >= 0 {
		rtspURL = rtspURL[:idx]
	}
	return rtspURL
}
