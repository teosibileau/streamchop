package steps

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type mqttState int

const (
	mqttMenu mqttState = iota
	mqttScanning
	mqttSelectBroker
	mqttManualEntry
	mqttDone
	mqttSkipped
)

type mqttScanResultMsg struct {
	hosts []string
	err   error
}

// MQTTModel handles the MQTT broker configuration step.
type MQTTModel struct {
	state      mqttState
	spinner    spinner.Model
	hostInput  textinput.Model
	portInput  textinput.Model
	field      int // 0=host, 1=port
	config     MQTTConfig
	hostIP     string
	foundHosts []string
	cursor     int
	err        error
	done       bool
}

func NewMQTTModel(hostIP string) MQTTModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	hi := textinput.New()
	hi.Placeholder = "192.168.1.50"
	hi.CharLimit = 45

	pi := textinput.New()
	pi.Placeholder = "1883"
	pi.CharLimit = 5

	return MQTTModel{
		state:     mqttMenu,
		spinner:   s,
		hostInput: hi,
		portInput: pi,
		hostIP:    hostIP,
	}
}

func (m MQTTModel) Init() tea.Cmd {
	return nil
}

func (m MQTTModel) Update(msg tea.Msg) (MQTTModel, tea.Cmd) {
	switch msg := msg.(type) {
	case mqttScanResultMsg:
		if msg.err != nil {
			m.state = mqttMenu
			m.err = msg.err
			return m, nil
		}
		m.foundHosts = msg.hosts
		m.cursor = 0
		m.state = mqttSelectBroker
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case mqttSelectBroker:
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.foundHosts)-1 {
					m.cursor++
				}
			case "enter":
				m.config = MQTTConfig{Enabled: true, Host: m.foundHosts[m.cursor], Port: "1883"}
				m.state = mqttDone
				m.done = true
				return m, nil
			case "esc":
				m.state = mqttMenu
				return m, nil
			}
			return m, nil

		case mqttMenu:
			switch msg.String() {
			case "s":
				m.state = mqttScanning
				m.err = nil
				return m, tea.Batch(m.spinner.Tick, scanMQTTCmd(m.hostIP))
			case "m":
				m.state = mqttManualEntry
				m.hostInput.Focus()
				return m, m.hostInput.Focus()
			case "n":
				m.config = MQTTConfig{Enabled: false}
				m.state = mqttSkipped
				m.done = true
				return m, nil
			}

		case mqttManualEntry:
			switch msg.String() {
			case "tab":
				if m.field == 0 {
					m.field = 1
					m.hostInput.Blur()
					return m, m.portInput.Focus()
				}
				m.field = 0
				m.portInput.Blur()
				return m, m.hostInput.Focus()
			case "enter":
				host := m.hostInput.Value()
				if host == "" {
					return m, nil
				}
				port := m.portInput.Value()
				if port == "" {
					port = "1883"
				}
				m.config = MQTTConfig{Enabled: true, Host: host, Port: port}
				m.state = mqttDone
				m.done = true
				return m, nil
			case "esc":
				m.state = mqttMenu
				m.hostInput.SetValue("")
				m.portInput.SetValue("")
				m.field = 0
				return m, nil
			}

			var cmd tea.Cmd
			if m.field == 0 {
				m.hostInput, cmd = m.hostInput.Update(msg)
			} else {
				m.portInput, cmd = m.portInput.Update(msg)
			}
			return m, cmd
		}

	case spinner.TickMsg:
		if m.state == mqttScanning {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m MQTTModel) View() string {
	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Render("MQTT Broker Configuration")

	switch m.state {
	case mqttMenu:
		fmt.Fprintf(&b, "\n  %s\n\n", title)
		if m.err != nil {
			errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
			fmt.Fprintf(&b, "  %s\n\n", errStyle.Render(m.err.Error()))
		}
		b.WriteString("  (s) Scan network for MQTT broker\n")
		b.WriteString("  (m) Manual entry\n")
		b.WriteString("  (n) No MQTT (skip emitter)\n")

	case mqttScanning:
		fmt.Fprintf(&b, "\n  %s Scanning for MQTT broker on local network...\n", m.spinner.View())

	case mqttSelectBroker:
		fmt.Fprintf(&b, "\n  %s\n\n", title)
		fmt.Fprintf(&b, "  Found %d broker(s):\n\n", len(m.foundHosts))
		for i, host := range m.foundHosts {
			cursor := "  "
			if i == m.cursor {
				cursor = "> "
			}
			fmt.Fprintf(&b, "  %s%s\n", cursor, host)
		}
		b.WriteString("\n  (enter) Select  (j/k) Navigate  (esc) Back\n")

	case mqttManualEntry:
		fmt.Fprintf(&b, "\n  %s\n\n", title)
		fmt.Fprintf(&b, "  Host: %s\n", m.hostInput.View())
		fmt.Fprintf(&b, "  Port: %s\n\n", m.portInput.View())
		b.WriteString("  (tab) Switch field  (enter) Confirm  (esc) Back\n")

	case mqttDone:
		ok := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
		fmt.Fprintf(&b, "\n  %s MQTT broker: %s:%s\n",
			ok.Render("✓"), m.config.Host, m.config.Port)

	case mqttSkipped:
		b.WriteString("\n  MQTT integration skipped — emitter will not be included.\n")
	}

	return b.String()
}

func (m MQTTModel) Done() bool        { return m.done }
func (m MQTTModel) Config() MQTTConfig { return m.config }

func scanMQTTCmd(hostIP string) tea.Cmd {
	return func() tea.Msg {
		hosts, err := ScanMQTTBrokers(hostIP)
		return mqttScanResultMsg{hosts: hosts, err: err}
	}
}
