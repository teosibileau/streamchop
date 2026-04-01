package steps

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/teosibileau/streamchop/tui/compose"
	"github.com/teosibileau/streamchop/tui/systemd"
)

type generateState int

const (
	generateConfirm generateState = iota
	generateWriting
	generateAskInstall
	generateInstalling
	generateInstallDone
	generateDone
	generateError
)

type generateDoneMsg struct {
	err error
}

type installDoneMsg struct {
	err error
}

// GenerateModel handles the final confirmation and file generation step.
type GenerateModel struct {
	state      generateState
	cameras    []ConfiguredCamera
	mqttConfig MQTTConfig
	hostIP     string
	err        error
	done       bool
}

func NewGenerateModel(cameras []ConfiguredCamera, mqttConfig MQTTConfig, hostIP string) GenerateModel {
	return GenerateModel{
		state:      generateConfirm,
		cameras:    cameras,
		mqttConfig: mqttConfig,
		hostIP:     hostIP,
	}
}

func (m GenerateModel) Init() tea.Cmd {
	return nil
}

func (m GenerateModel) Update(msg tea.Msg) (GenerateModel, tea.Cmd) {
	switch msg := msg.(type) {
	case generateDoneMsg:
		if msg.err != nil {
			m.state = generateError
			m.err = msg.err
			return m, nil
		}
		m.state = generateAskInstall
		return m, nil

	case installDoneMsg:
		if msg.err != nil {
			m.state = generateError
			m.err = msg.err
			return m, nil
		}
		m.state = generateInstallDone
		m.done = true
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case generateConfirm:
			switch msg.String() {
			case "enter", "y":
				m.state = generateWriting
				return m, generateFiles(m.cameras, m.mqttConfig, m.hostIP)
			case "q", "n":
				m.done = true
				return m, nil
			}
		case generateAskInstall:
			switch msg.String() {
			case "y":
				m.state = generateInstalling
				return m, installServiceCmd()
			case "n":
				m.state = generateDone
				m.done = true
				return m, nil
			}
		case generateError:
			switch msg.String() {
			case "r":
				m.state = generateWriting
				return m, generateFiles(m.cameras, m.mqttConfig, m.hostIP)
			case "q":
				m.done = true
				return m, nil
			}
		}
	}

	return m, nil
}

func (m GenerateModel) View() string {
	var b strings.Builder

	switch m.state {
	case generateConfirm:
		title := lipgloss.NewStyle().Bold(true).Render("Summary")
		fmt.Fprintf(&b, "\n  %s\n\n", title)

		for i, cam := range m.cameras {
			fmt.Fprintf(&b, "  Camera %d: %s (%s)\n", i+1, cam.Camera.Name, cam.Camera.IP)
			fmt.Fprintf(&b, "    Stream: %s\n", cam.Stream.ProfileName)
			fmt.Fprintf(&b, "    URI:    %s\n\n", cam.Stream.URI)
		}

		fmt.Fprintf(&b, "  HLS Base URL: http://%s:8080\n\n", m.hostIP)

		if m.mqttConfig.Enabled {
			fmt.Fprintf(&b, "  MQTT Broker: %s:%s\n\n", m.mqttConfig.Host, m.mqttConfig.Port)
		} else {
			b.WriteString("  MQTT: disabled (emitter will not be included)\n\n")
		}

		b.WriteString("  Files to write:\n")
		b.WriteString("    - docker-compose.dist.yml\n")
		b.WriteString("    - .env\n\n")
		b.WriteString("  (enter/y) Generate  (q/n) Cancel\n")

	case generateWriting:
		b.WriteString("\n  Writing files...\n")

	case generateAskInstall:
		ok := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
		fmt.Fprintf(&b, "\n  %s\n\n", ok.Render("Files generated!"))
		b.WriteString("  Generated:\n")
		b.WriteString("    - docker-compose.dist.yml\n")
		b.WriteString("    - .env\n\n")
		b.WriteString("  Install as systemd service? (y/n)\n")

	case generateInstalling:
		b.WriteString("\n  Installing systemd service...\n")

	case generateInstallDone:
		ok := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
		fmt.Fprintf(&b, "\n  %s\n\n", ok.Render("Setup complete!"))
		b.WriteString("  Generated:\n")
		b.WriteString("    - docker-compose.dist.yml\n")
		b.WriteString("    - .env\n\n")
		b.WriteString("  Systemd service installed and running.\n")
		b.WriteString("  Run 'streamchop status' to check health.\n")

	case generateDone:
		ok := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
		fmt.Fprintf(&b, "\n  %s\n\n", ok.Render("Setup complete!"))
		b.WriteString("  Generated:\n")
		b.WriteString("    - docker-compose.dist.yml\n")
		b.WriteString("    - .env\n\n")
		b.WriteString("  Run 'streamchop install' to install as a systemd service.\n")

	case generateError:
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		fmt.Fprintf(&b, "\n  %s\n\n", errStyle.Render(m.err.Error()))
		b.WriteString("  (r) Retry  (q) Quit\n")
	}

	return b.String()
}

func (m GenerateModel) Done() bool { return m.done }

func generateFiles(cameras []ConfiguredCamera, mqttConfig MQTTConfig, hostIP string) tea.Cmd {
	return func() tea.Msg {
		cameraConfigs := make([]compose.CameraConfig, len(cameras))
		for i, cam := range cameras {
			cameraConfigs[i] = compose.CameraConfig{
				Index:   i + 1,
				EnvVar:  fmt.Sprintf("CAM%d_RTSP_URL", i+1),
				RTSPURL: cam.Stream.URI,
			}
		}

		if err := compose.GenerateDistCompose("docker-compose.dist.yml", cameraConfigs, mqttConfig.Enabled); err != nil {
			return generateDoneMsg{err: fmt.Errorf("write compose file: %w", err)}
		}

		envConfig := compose.EnvConfig{
			GHCRRepo:    "ghcr.io/teosibileau/streamchop",
			Tag:         "latest",
			Cameras:     cameraConfigs,
			IncludeMQTT: mqttConfig.Enabled,
			MQTTHost:    mqttConfig.Host,
			MQTTPort:    mqttConfig.Port,
			HLSBaseURL:  fmt.Sprintf("http://%s:8080", hostIP),
		}

		if err := compose.GenerateEnv(".env", envConfig); err != nil {
			return generateDoneMsg{err: fmt.Errorf("write env file: %w", err)}
		}

		return generateDoneMsg{}
	}
}

func installServiceCmd() tea.Cmd {
	return func() tea.Msg {
		wd, err := os.Getwd()
		if err != nil {
			return installDoneMsg{err: err}
		}
		if err := systemd.Install(wd); err != nil {
			return installDoneMsg{err: err}
		}
		return installDoneMsg{}
	}
}
