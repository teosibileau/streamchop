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
	cameras       []onvif.Camera
	creds         []onvif.Credentials
	camIndex      int
	field         credentialField
	username      textinput.Model
	password      textinput.Model
	sameForAll    bool
	done          bool
	existingCreds map[string][2]string
}

// NewCredentialsModel creates a credential input step. existingCreds maps
// camera IP to [username, password] for pre-filling from a previous .env.
func NewCredentialsModel(cameras []onvif.Camera, existingCreds map[string][2]string) CredentialsModel {
	u := textinput.New()
	u.Placeholder = "admin"
	u.CharLimit = 64
	u.Focus()

	p := textinput.New()
	p.Placeholder = "password"
	p.CharLimit = 128
	p.EchoMode = textinput.EchoPassword

	// Pre-fill from first camera's existing creds if available
	creds := make([]onvif.Credentials, len(cameras))
	if len(cameras) > 0 {
		if existing, ok := existingCreds[cameras[0].IP]; ok {
			u.SetValue(existing[0])
			p.SetValue(existing[1])
			creds[0] = onvif.Credentials{Username: existing[0], Password: existing[1]}
		}
	}

	return CredentialsModel{
		cameras:       cameras,
		creds:         creds,
		username:      u,
		password:      p,
		existingCreds: existingCreds,
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

			// Pre-fill next camera's creds if available
			nextCam := m.cameras[m.camIndex]
			if existing, ok := m.existingCreds[nextCam.IP]; ok {
				m.username.SetValue(existing[0])
				m.password.SetValue(existing[1])
			} else {
				m.username.SetValue("")
				m.password.SetValue("")
			}
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

	fmt.Fprintf(&b, "\n  %s\n", title)
	fmt.Fprintf(&b, "  Camera %d of %d\n\n", m.camIndex+1, len(m.cameras))
	fmt.Fprintf(&b, "  Username: %s\n", m.username.View())
	fmt.Fprintf(&b, "  Password: %s\n\n", m.password.View())

	sameLabel := "[ ] Same for all cameras"
	if m.sameForAll {
		sameLabel = "[x] Same for all cameras"
	}
	fmt.Fprintf(&b, "  %s\n\n", sameLabel)
	b.WriteString("  (tab) Switch field  (ctrl+a) Toggle same-for-all  (enter) Confirm\n")

	return b.String()
}

func (m CredentialsModel) Done() bool                   { return m.done }
func (m CredentialsModel) Credentials() []onvif.Credentials { return m.creds }
