package steps

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/teosibileau/streamchop/tui/onvif"
)

type discoveryState int

const (
	discoveryScanning discoveryState = iota
	discoveryDone
	discoveryFailed
	discoveryManualEntry
)

type camerasFoundMsg struct {
	cameras []onvif.Camera
	err     error
}

// DiscoveryModel handles the camera discovery step.
type DiscoveryModel struct {
	state       discoveryState
	spinner     spinner.Model
	cameras     []onvif.Camera
	err         error
	manualInput textinput.Model
	done        bool
}

func NewDiscoveryModel() DiscoveryModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.Placeholder = "192.168.1.100"
	ti.CharLimit = 45

	return DiscoveryModel{
		state:       discoveryScanning,
		spinner:     s,
		manualInput: ti,
	}
}

func (m DiscoveryModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, discoverCameras())
}

func (m DiscoveryModel) Update(msg tea.Msg) (DiscoveryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case camerasFoundMsg:
		if msg.err != nil || len(msg.cameras) == 0 {
			m.state = discoveryFailed
			m.err = msg.err
			if m.err == nil {
				m.err = fmt.Errorf("no cameras found on the network")
			}
			return m, nil
		}
		m.cameras = msg.cameras
		m.state = discoveryDone
		m.done = true
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case discoveryFailed:
			switch msg.String() {
			case "r":
				m.state = discoveryScanning
				m.err = nil
				return m, tea.Batch(m.spinner.Tick, discoverCameras())
			case "m":
				m.state = discoveryManualEntry
				m.manualInput.Focus()
				return m, m.manualInput.Focus()
			case "q":
				m.done = true
				return m, nil
			}

		case discoveryManualEntry:
			switch msg.String() {
			case "enter":
				ip := m.manualInput.Value()
				if ip != "" {
					m.state = discoveryScanning
					return m, tea.Batch(m.spinner.Tick, probeAddress(ip))
				}
			case "esc":
				m.state = discoveryFailed
				return m, nil
			}

			var cmd tea.Cmd
			m.manualInput, cmd = m.manualInput.Update(msg)
			return m, cmd
		}

	case spinner.TickMsg:
		if m.state == discoveryScanning {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m DiscoveryModel) View() string {
	switch m.state {
	case discoveryScanning:
		return fmt.Sprintf("\n  %s Scanning LAN for ONVIF cameras...\n", m.spinner.View())

	case discoveryDone:
		return fmt.Sprintf("\n  Found %d camera(s)!\n", len(m.cameras))

	case discoveryFailed:
		return fmt.Sprintf("\n  %s\n\n  (r) Retry  (m) Manual IP entry  (q) Quit\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.err.Error()))

	case discoveryManualEntry:
		return fmt.Sprintf("\n  Enter camera IP address:\n\n  %s\n\n  (enter) Probe  (esc) Back\n",
			m.manualInput.View())
	}

	return ""
}

func (m DiscoveryModel) Done() bool         { return m.done }
func (m DiscoveryModel) Cameras() []onvif.Camera { return m.cameras }
func (m DiscoveryModel) Err() error          { return m.err }

func discoverCameras() tea.Cmd {
	return func() tea.Msg {
		cameras, err := onvif.Discover()
		return camerasFoundMsg{cameras: cameras, err: err}
	}
}

func probeAddress(ip string) tea.Cmd {
	return func() tea.Msg {
		cameras, err := onvif.ProbeAddress(ip)
		return camerasFoundMsg{cameras: cameras, err: err}
	}
}
