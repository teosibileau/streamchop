package main

import (
	"github.com/teosibileau/streamchop/tui/compose"
	"github.com/teosibileau/streamchop/tui/onvif"
	"github.com/teosibileau/streamchop/tui/steps"

	tea "github.com/charmbracelet/bubbletea"
)

type step int

const (
	stepDiscovery step = iota
	stepSelection
	stepCredentials
	stepProbe
	stepGenerate
	stepDone
)

type model struct {
	step        step
	discovery   steps.DiscoveryModel
	selection   steps.SelectionModel
	credentials steps.CredentialsModel
	probe       steps.ProbeModel
	generate    steps.GenerateModel
	cameras     []onvif.Camera
	selected    []onvif.Camera
	configured  []steps.ConfiguredCamera
	err         error
}

func newModel() model {
	existing := compose.ParseExistingDist("docker-compose.dist.yml", ".env")
	return model{
		step:      stepDiscovery,
		discovery: steps.NewDiscoveryModel(),
		selection: steps.NewSelectionModel(existing),
	}
}

func (m model) Init() tea.Cmd {
	return m.discovery.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	switch m.step {
	case stepDiscovery:
		updated, cmd := m.discovery.Update(msg)
		m.discovery = updated
		if m.discovery.Done() {
			m.cameras = m.discovery.Cameras()
			if len(m.cameras) == 0 {
				m.err = m.discovery.Err()
				return m, tea.Quit
			}
			m.selection.SetCameras(m.cameras)
			m.step = stepSelection
			return m, m.selection.Init()
		}
		return m, cmd

	case stepSelection:
		updated, cmd := m.selection.Update(msg)
		m.selection = updated
		if m.selection.Done() {
			m.selected = m.selection.Selected()
			m.credentials = steps.NewCredentialsModel(m.selected)
			m.step = stepCredentials
			return m, m.credentials.Init()
		}
		return m, cmd

	case stepCredentials:
		updated, cmd := m.credentials.Update(msg)
		m.credentials = updated
		if m.credentials.Done() {
			creds := m.credentials.Credentials()
			m.probe = steps.NewProbeModel(m.selected, creds)
			m.step = stepProbe
			return m, m.probe.Init()
		}
		return m, cmd

	case stepProbe:
		updated, cmd := m.probe.Update(msg)
		m.probe = updated
		if m.probe.Done() {
			m.configured = m.probe.Configured()
			m.generate = steps.NewGenerateModel(m.configured)
			m.step = stepGenerate
			return m, m.generate.Init()
		}
		return m, cmd

	case stepGenerate:
		updated, cmd := m.generate.Update(msg)
		m.generate = updated
		if m.generate.Done() {
			m.step = stepDone
			return m, tea.Quit
		}
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	switch m.step {
	case stepDiscovery:
		return m.discovery.View()
	case stepSelection:
		return m.selection.View()
	case stepCredentials:
		return m.credentials.View()
	case stepProbe:
		return m.probe.View()
	case stepGenerate:
		return m.generate.View()
	case stepDone:
		return "Setup complete!\n"
	}
	return ""
}
