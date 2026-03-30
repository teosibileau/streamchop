package steps

import "github.com/teosibileau/streamchop/tui/onvif"

// ConfiguredCamera holds a fully configured camera ready for compose generation.
type ConfiguredCamera struct {
	Camera onvif.Camera
	Creds  onvif.Credentials
	Stream onvif.StreamInfo
}

// MQTTConfig holds the MQTT broker configuration from the TUI.
type MQTTConfig struct {
	Enabled bool
	Host    string
	Port    string
}
