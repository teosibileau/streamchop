package steps

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/teosibileau/streamchop/tui/onvif"
)

type probeState int

const (
	probeRunning probeState = iota
	probeResults
)

type probeResultMsg struct {
	index   int
	streams []onvif.StreamInfo
	err     error
}

type cameraProbeResult struct {
	camera  onvif.Camera
	creds   onvif.Credentials
	streams []onvif.StreamInfo
	err     error
	chosen  int // index into streams
}

// ProbeModel handles RTSP stream probing for selected cameras.
type ProbeModel struct {
	state    probeState
	spinner  spinner.Model
	cameras  []onvif.Camera
	creds    []onvif.Credentials
	results  []cameraProbeResult
	pending  int
	cursor   int
	done     bool
}

func NewProbeModel(cameras []onvif.Camera, creds []onvif.Credentials) ProbeModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	results := make([]cameraProbeResult, len(cameras))
	for i, cam := range cameras {
		results[i] = cameraProbeResult{camera: cam, creds: creds[i]}
	}

	return ProbeModel{
		state:   probeRunning,
		spinner: s,
		cameras: cameras,
		creds:   creds,
		results: results,
		pending: len(cameras),
	}
}

func (m ProbeModel) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick}
	for i, cam := range m.cameras {
		cmds = append(cmds, probeCameraCmd(i, cam.XAddr, m.creds[i]))
	}
	return tea.Batch(cmds...)
}

func (m ProbeModel) Update(msg tea.Msg) (ProbeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case probeResultMsg:
		m.results[msg.index].streams = msg.streams
		m.results[msg.index].err = msg.err
		m.pending--
		if m.pending <= 0 {
			m.state = probeResults
		}
		return m, nil

	case tea.KeyMsg:
		if m.state == probeResults {
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.results)-1 {
					m.cursor++
				}
			case "left", "h":
				r := &m.results[m.cursor]
				if len(r.streams) > 1 && r.chosen > 0 {
					r.chosen--
				}
			case "right", "l":
				r := &m.results[m.cursor]
				if len(r.streams) > 1 && r.chosen < len(r.streams)-1 {
					r.chosen++
				}
			case "r":
				r := m.results[m.cursor]
				if r.err != nil {
					m.results[m.cursor].err = nil
					m.pending++
					m.state = probeRunning
					return m, tea.Batch(m.spinner.Tick,
						probeCameraCmd(m.cursor, r.camera.XAddr, r.creds))
				}
			case "enter":
				m.done = true
				return m, nil
			}
		}

	case spinner.TickMsg:
		if m.state == probeRunning {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m ProbeModel) View() string {
	var b strings.Builder

	if m.state == probeRunning {
		fmt.Fprintf(&b, "\n  %s Probing cameras for RTSP streams... (%d remaining)\n",
			m.spinner.View(), m.pending)
		return b.String()
	}

	title := lipgloss.NewStyle().Bold(true).Render("Probe Results")
	fmt.Fprintf(&b, "\n  %s\n\n", title)

	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	for i, r := range m.results {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		if r.err != nil {
			fmt.Fprintf(&b, "  %s%s %s\n",
				cursor, errStyle.Render("✗"), r.camera.Name)
			fmt.Fprintf(&b, "      %s\n", errStyle.Render(r.err.Error()))
		} else {
			stream := r.streams[r.chosen]
			profileInfo := fmt.Sprintf("[%d/%d] %s", r.chosen+1, len(r.streams), stream.ProfileName)
			fmt.Fprintf(&b, "  %s%s %s %s\n",
				cursor, okStyle.Render("✓"), r.camera.Name, profileInfo)
			fmt.Fprintf(&b, "      %s\n", stream.URI)
		}
	}

	b.WriteString("\n  (h/l) Switch profile  (r) Retry failed  (enter) Confirm\n")
	return b.String()
}

func (m ProbeModel) Done() bool { return m.done }

func (m ProbeModel) Configured() []ConfiguredCamera {
	var result []ConfiguredCamera
	for _, r := range m.results {
		if r.err != nil || len(r.streams) == 0 {
			continue
		}
		result = append(result, ConfiguredCamera{
			Camera: r.camera,
			Creds:  r.creds,
			Stream: r.streams[r.chosen],
		})
	}
	return result
}

func probeCameraCmd(index int, xaddr string, creds onvif.Credentials) tea.Cmd {
	return func() tea.Msg {
		streams, err := onvif.GetStreamURIs(xaddr, creds)
		return probeResultMsg{index: index, streams: streams, err: err}
	}
}
