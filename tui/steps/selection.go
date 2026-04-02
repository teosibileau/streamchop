package steps

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/teosibileau/streamchop/tui/onvif"
)

// SelectionModel handles the camera multi-select step.
type SelectionModel struct {
	cameras     []onvif.Camera
	selected    map[int]bool
	cursor      int
	done        bool
	existingIPs []string
}

func NewSelectionModel(existingIPs []string) SelectionModel {
	return SelectionModel{
		selected:    make(map[int]bool),
		existingIPs: existingIPs,
	}
}

// SetCameras populates the list and pre-selects cameras matching existing IPs.
func (m *SelectionModel) SetCameras(cameras []onvif.Camera) {
	m.cameras = cameras
	m.selected = make(map[int]bool)

	ipSet := make(map[string]bool)
	for _, ip := range m.existingIPs {
		ipSet[ip] = true
	}

	for i, cam := range cameras {
		if ipSet[cam.IP] {
			m.selected[i] = true
		}
	}
}

func (m SelectionModel) Init() tea.Cmd {
	return nil
}

func (m SelectionModel) Update(msg tea.Msg) (SelectionModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.cameras)-1 {
				m.cursor++
			}
		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]
			if !m.selected[m.cursor] {
				delete(m.selected, m.cursor)
			}
		case "enter":
			if len(m.selected) > 0 {
				m.done = true
			}
		}
	}

	return m, nil
}

func (m SelectionModel) View() string {
	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Render("Select cameras to configure")
	fmt.Fprintf(&b, "\n  %s\n\n", title)

	for i, cam := range m.cameras {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		checked := "[ ]"
		if m.selected[i] {
			checked = "[x]"
		}

		label := fmt.Sprintf("%s (%s:%s)", cam.Name, cam.IP, cam.Port)
		fmt.Fprintf(&b, "  %s%s %s\n", cursor, checked, label)
	}

	b.WriteString("\n  (space) Toggle  (enter) Confirm  (j/k) Navigate\n")
	return b.String()
}

func (m SelectionModel) Done() bool { return m.done }

func (m SelectionModel) Selected() []onvif.Camera {
	var result []onvif.Camera
	for i, cam := range m.cameras {
		if m.selected[i] {
			result = append(result, cam)
		}
	}
	return result
}
