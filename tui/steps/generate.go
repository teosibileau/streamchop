package steps

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/teosibileau/streamchop/tui/compose"
)

type generateState int

const (
	generateConfirm generateState = iota
	generateWriting
	generateDone
	generateError
)

type generateDoneMsg struct {
	err error
}

// GenerateModel handles the final confirmation and file generation step.
type GenerateModel struct {
	state    generateState
	cameras  []ConfiguredCamera
	err      error
	done     bool
}

func NewGenerateModel(cameras []ConfiguredCamera) GenerateModel {
	return GenerateModel{
		state:   generateConfirm,
		cameras: cameras,
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
		m.state = generateDone
		m.done = true
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case generateConfirm:
			switch msg.String() {
			case "enter", "y":
				m.state = generateWriting
				return m, generateFiles(m.cameras)
			case "q", "n":
				m.done = true
				return m, nil
			}
		case generateError:
			switch msg.String() {
			case "r":
				m.state = generateWriting
				return m, generateFiles(m.cameras)
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
		b.WriteString(fmt.Sprintf("\n  %s\n\n", title))

		for i, cam := range m.cameras {
			b.WriteString(fmt.Sprintf("  Camera %d: %s (%s)\n", i+1, cam.Camera.Name, cam.Camera.IP))
			b.WriteString(fmt.Sprintf("    Stream: %s\n", cam.Stream.ProfileName))
			b.WriteString(fmt.Sprintf("    URI:    %s\n\n", cam.Stream.URI))
		}

		b.WriteString("  Files to write:\n")
		b.WriteString("    - docker-compose.dist.yml\n")
		b.WriteString("    - .env\n\n")
		b.WriteString("  (enter/y) Generate  (q/n) Cancel\n")

	case generateWriting:
		b.WriteString("\n  Writing files...\n")

	case generateDone:
		ok := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
		b.WriteString(fmt.Sprintf("\n  %s\n\n", ok.Render("Setup complete!")))
		b.WriteString("  Generated:\n")
		b.WriteString("    - docker-compose.dist.yml\n")
		b.WriteString("    - .env\n\n")
		b.WriteString("  Next steps:\n")
		b.WriteString("    - Review the generated files\n")
		b.WriteString("    - Run: ahoy systemd install\n")

	case generateError:
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		b.WriteString(fmt.Sprintf("\n  %s\n\n", errStyle.Render(m.err.Error())))
		b.WriteString("  (r) Retry  (q) Quit\n")
	}

	return b.String()
}

func (m GenerateModel) Done() bool { return m.done }

func generateFiles(cameras []ConfiguredCamera) tea.Cmd {
	return func() tea.Msg {
		cameraConfigs := make([]compose.CameraConfig, len(cameras))
		for i, cam := range cameras {
			cameraConfigs[i] = compose.CameraConfig{
				Index:   i + 1,
				EnvVar:  fmt.Sprintf("CAM%d_RTSP_URL", i+1),
				RTSPURL: cam.Stream.URI,
			}
		}

		if err := compose.GenerateDistCompose("docker-compose.dist.yml", cameraConfigs); err != nil {
			return generateDoneMsg{err: fmt.Errorf("write compose file: %w", err)}
		}

		envConfig := compose.EnvConfig{
			GHCRRepo: "ghcr.io/teosibileau/streamchop",
			Tag:      "latest",
			Cameras:  cameraConfigs,
		}

		if err := compose.GenerateEnv(".env", envConfig); err != nil {
			return generateDoneMsg{err: fmt.Errorf("write env file: %w", err)}
		}

		return generateDoneMsg{}
	}
}
