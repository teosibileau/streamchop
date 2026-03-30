package steps

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/teosibileau/streamchop/tui/onvif"
)

type credentialField int

const (
	fieldUsername credentialField = iota
	fieldPassword
)

// CredentialsModel handles per-camera credential input.
type CredentialsModel struct {
	cameras    []onvif.Camera
	creds      []onvif.Credentials
	camIndex   int
	field      credentialField
	username   textinput.Model
	password   textinput.Model
	sameForAll bool
	done       bool
}

func NewCredentialsModel(cameras []onvif.Camera) CredentialsModel {
	u := textinput.New()
	u.Placeholder = "admin"
	u.CharLimit = 64
	u.Focus()

	p := textinput.New()
	p.Placeholder = "password"
	p.CharLimit = 128
	p.EchoMode = textinput.EchoPassword

	return CredentialsModel{
		cameras:  cameras,
		creds:    make([]onvif.Credentials, len(cameras)),
		username: u,
		password: p,
	}
}

func (m CredentialsModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m CredentialsModel) Update(msg tea.Msg) (CredentialsModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "tab":
			if m.field == fieldUsername {
				m.field = fieldPassword
				m.username.Blur()
				return m, m.password.Focus()
			}
			m.field = fieldUsername
			m.password.Blur()
			return m, m.username.Focus()

		case "ctrl+a":
			m.sameForAll = !m.sameForAll

		case "enter":
			user := m.username.Value()
			pass := m.password.Value()
			if user == "" {
				return m, nil
			}

			cred := onvif.Credentials{Username: user, Password: pass}

			if m.sameForAll {
				for i := range m.creds {
					m.creds[i] = cred
				}
				m.done = true
				return m, nil
			}

			m.creds[m.camIndex] = cred
			m.camIndex++

			if m.camIndex >= len(m.cameras) {
				m.done = true
				return m, nil
			}

			m.username.SetValue("")
			m.password.SetValue("")
			m.field = fieldUsername
			m.password.Blur()
			return m, m.username.Focus()
		}
	}

	var cmd tea.Cmd
	if m.field == fieldUsername {
		m.username, cmd = m.username.Update(msg)
	} else {
		m.password, cmd = m.password.Update(msg)
	}

	return m, cmd
}

func (m CredentialsModel) View() string {
	var b strings.Builder

	cam := m.cameras[m.camIndex]
	title := lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf("Credentials for %s (%s)", cam.Name, cam.IP))

	b.WriteString(fmt.Sprintf("\n  %s\n", title))
	b.WriteString(fmt.Sprintf("  Camera %d of %d\n\n", m.camIndex+1, len(m.cameras)))
	b.WriteString(fmt.Sprintf("  Username: %s\n", m.username.View()))
	b.WriteString(fmt.Sprintf("  Password: %s\n\n", m.password.View()))

	sameLabel := "[ ] Same for all cameras"
	if m.sameForAll {
		sameLabel = "[x] Same for all cameras"
	}
	b.WriteString(fmt.Sprintf("  %s\n\n", sameLabel))
	b.WriteString("  (tab) Switch field  (ctrl+a) Toggle same-for-all  (enter) Confirm\n")

	return b.String()
}

func (m CredentialsModel) Done() bool                   { return m.done }
func (m CredentialsModel) Credentials() []onvif.Credentials { return m.creds }
