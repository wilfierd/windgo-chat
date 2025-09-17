package ui

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/wilfierd/windgo-chat-app/cli/internal/api"
	"github.com/wilfierd/windgo-chat-app/cli/internal/storage"
)

type viewState int

const (
	stateLoading viewState = iota
	stateLoginMenu
	stateEmailLogin
	stateDeviceSetup
	stateDeviceWaiting
	stateLoggedIn
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	menuStyle    = lipgloss.NewStyle().Padding(1, 0)
	selectedItem = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	statusStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
)

var loginOptions = []string{
	"Login with email/password",
	"Login with GitHub device flow",
}

// Model holds application state for the login experience.
type Model struct {
	client *api.Client

	state      viewState
	menuIndex  int
	focusIndex int
	err        error
	status     string
	token      string
	user       *api.User
	creds      *storage.Credentials
	submitting bool

	emailInput    textinput.Model
	passwordInput textinput.Model

	deviceInfo *api.DeviceStartResponse
}

func NewModel(client *api.Client) Model {
	email := textinput.New()
	email.Placeholder = "you@example.com"
	email.Prompt = "Email> "
	email.CharLimit = 256

	password := textinput.New()
	password.Placeholder = "password"
	password.Prompt = "Password> "
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '•'
	password.CharLimit = 256

	return Model{
		client:        client,
		state:         stateLoading,
		emailInput:    email,
		passwordInput: password,
	}
}

// Messages emitted from commands

type storedCredsMsg struct {
	creds *storage.Credentials
	err   error
}

type profileLoadedMsg struct {
	user *api.User
	err  error
}

type authSuccessMsg struct {
	resp *api.AuthResponse
}

type credsSavedMsg struct {
	err error
}

type errMsg struct {
	err error
}

type deviceStartMsg struct {
	resp *api.DeviceStartResponse
}

func (m Model) Init() tea.Cmd {
	return loadStoredCredentials()
}

func loadStoredCredentials() tea.Cmd {
	return func() tea.Msg {
		creds, err := storage.Load()
		if err != nil {
			if errors.Is(err, storage.ErrNoCredentials) || errors.Is(err, os.ErrNotExist) {
				return storedCredsMsg{}
			}
			return storedCredsMsg{err: err}
		}
		return storedCredsMsg{creds: creds}
	}
}

func verifyTokenCmd(client *api.Client, token string) tea.Cmd {
	return func() tea.Msg {
		user, err := client.Profile(token)
		if err != nil {
			return profileLoadedMsg{err: err}
		}
		return profileLoadedMsg{user: user}
	}
}

func loginCmd(client *api.Client, email, password string) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.Login(email, password)
		if err != nil {
			return errMsg{err: err}
		}
		return authSuccessMsg{resp: resp}
	}
}

func startDeviceFlowCmd(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.StartDeviceFlow()
		if err != nil {
			return errMsg{err: err}
		}
		return deviceStartMsg{resp: resp}
	}
}

func pollDeviceCmd(client *api.Client, deviceCode string) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.PollDevice(deviceCode, 120)
		if err != nil {
			return errMsg{err: err}
		}
		return authSuccessMsg{resp: resp}
	}
}

func saveCredentialsCmd(resp *api.AuthResponse) tea.Cmd {
	creds := storage.Credentials{
		Token:    resp.Token,
		Username: resp.User.Username,
		Email:    resp.User.Email,
		Provider: resp.User.Provider,
	}
	return func() tea.Msg {
		return credsSavedMsg{err: storage.Save(creds)}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case storedCredsMsg:
		if msg.err != nil {
			m.err = msg.err
			m.status = "Failed to load stored credentials"
			m.state = stateLoginMenu
			return m, nil
		}
		if msg.creds != nil {
			m.creds = msg.creds
			m.token = msg.creds.Token
			m.status = "Found stored session, validating..."
			return m, verifyTokenCmd(m.client, m.token)
		}
		m.state = stateLoginMenu
		m.status = "Choose how you want to sign in."
		return m, nil

	case profileLoadedMsg:
		if msg.err != nil {
			m.status = "Stored credentials expired. Please sign in again."
			m.token = ""
			m.state = stateLoginMenu
			return m, nil
		}
		m.user = msg.user
		m.state = stateLoggedIn
		m.status = fmt.Sprintf("Welcome back, %s!", m.user.Username)
		return m, nil

	case deviceStartMsg:
		m.submitting = false
		m.err = nil
		m.deviceInfo = msg.resp
		m.state = stateDeviceSetup
		m.status = "Enter the code in your browser, then press Enter to continue."
		return m, nil

	case authSuccessMsg:
		m.submitting = false
		m.err = nil
		m.token = msg.resp.Token
		m.user = &msg.resp.User
		m.state = stateLoggedIn
		m.status = fmt.Sprintf("Welcome, %s!", m.user.Username)
		return m, saveCredentialsCmd(msg.resp)

	case credsSavedMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		return m, nil

	case errMsg:
		m.submitting = false
		if m.deviceInfo != nil {
			m.state = stateDeviceSetup
		} else if m.state == stateDeviceWaiting {
			m.state = stateLoginMenu
		}
		m.err = msg.err
		if m.err != nil {
			m.status = ""
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	switch m.state {
	case stateEmailLogin:
		var cmds []tea.Cmd
		var cmd tea.Cmd
		m.emailInput, cmd = m.emailInput.Update(msg)
		cmds = append(cmds, cmd)
		m.passwordInput, cmd = m.passwordInput.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateLoginMenu:
		switch msg.String() {
		case "up", "k":
			if m.menuIndex > 0 {
				m.menuIndex--
			}
		case "down", "j":
			if m.menuIndex < len(loginOptions)-1 {
				m.menuIndex++
			}
		case "enter":
			if m.menuIndex == 0 {
				m.state = stateEmailLogin
				m.err = nil
				m.status = ""
				m.emailInput.SetValue("")
				m.passwordInput.SetValue("")
				m.focusIndex = 0
				m.emailInput.Focus()
				m.passwordInput.Blur()
			} else {
				if m.submitting {
					return m, nil
				}
				m.submitting = true
				m.err = nil
				m.status = "Starting GitHub device flow..."
				return m, startDeviceFlowCmd(m.client)
			}
		}

	case stateEmailLogin:
		switch msg.String() {
		case "esc":
			m.state = stateLoginMenu
			m.status = "Choose how you want to sign in."
			m.err = nil
			m.submitting = false
			m.emailInput.Blur()
			m.passwordInput.Blur()
			return m, nil
		case "tab", "shift+tab":
			if m.focusIndex == 0 {
				m.focusIndex = 1
				m.emailInput.Blur()
				m.passwordInput.Focus()
			} else {
				m.focusIndex = 0
				m.passwordInput.Blur()
				m.emailInput.Focus()
			}
			return m, nil
		case "enter":
			if m.focusIndex == 0 {
				m.focusIndex = 1
				m.emailInput.Blur()
				m.passwordInput.Focus()
				return m, nil
			}
			if m.submitting {
				return m, nil
			}
			email := strings.TrimSpace(m.emailInput.Value())
			password := m.passwordInput.Value()
			if email == "" || password == "" {
				m.err = errors.New("email and password are required")
				return m, nil
			}
			m.submitting = true
			m.err = nil
			m.status = "Signing in..."
			return m, loginCmd(m.client, email, password)
		}

	case stateDeviceSetup:
		switch msg.String() {
		case "enter", "p":
			if m.deviceInfo == nil || m.submitting {
				return m, nil
			}
			m.submitting = true
			m.status = "Waiting for GitHub authorization..."
			m.state = stateDeviceWaiting
			return m, pollDeviceCmd(m.client, m.deviceInfo.DeviceCode)
		case "esc":
			m.deviceInfo = nil
			m.state = stateLoginMenu
			m.status = "Choose how you want to sign in."
			m.err = nil
		}

	case stateDeviceWaiting:
		switch msg.String() {
		case "esc":
			m.submitting = false
			m.state = stateLoginMenu
			m.status = "Choose how you want to sign in."
			m.err = nil
		}

	case stateLoggedIn:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("WindGo CLI"))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render("Error: " + m.err.Error()))
		b.WriteString("\n\n")
	}
	if m.status != "" {
		b.WriteString(statusStyle.Render(m.status))
		b.WriteString("\n\n")
	}

	switch m.state {
	case stateLoading:
		b.WriteString("Loading...")

	case stateLoginMenu:
		b.WriteString("Use the arrows and Enter to pick a sign-in method.\n\n")
		for i, opt := range loginOptions {
			if i == m.menuIndex {
				b.WriteString(selectedItem.Render("> " + opt))
			} else {
				b.WriteString("  " + opt)
			}
			b.WriteString("\n")
		}
		b.WriteString("\nPress Ctrl+C to quit.")

	case stateEmailLogin:
		b.WriteString("Email/password login\n\n")
		b.WriteString(m.emailInput.View())
		b.WriteString("\n")
		b.WriteString(m.passwordInput.View())
		b.WriteString("\n\n")
		if m.submitting {
			b.WriteString("Submitting credentials...\n")
		}
		b.WriteString("Tab to switch fields, Enter to submit, Esc to go back.")

	case stateDeviceSetup:
		if m.deviceInfo == nil {
			b.WriteString("Preparing GitHub device flow...")
			break
		}
		b.WriteString("GitHub Device Flow\n\n")
		b.WriteString(fmt.Sprintf("1. Visit %s\n", m.deviceInfo.VerificationURI))
		b.WriteString(fmt.Sprintf("2. Enter code: %s\n\n", lipgloss.NewStyle().Bold(true).Render(m.deviceInfo.UserCode)))
		if m.deviceInfo.VerificationURIComplete != "" {
			b.WriteString(fmt.Sprintf("Or open: %s\n\n", m.deviceInfo.VerificationURIComplete))
		}
		b.WriteString("Press Enter after authorizing to continue, or Esc to cancel.")

	case stateDeviceWaiting:
		b.WriteString("Waiting for GitHub authorization...\nPress Esc to cancel.")

	case stateLoggedIn:
		b.WriteString(fmt.Sprintf("Logged in as %s (%s).\n", m.user.Username, m.user.Email))
		b.WriteString("Press q to quit. The chat lobby will be available in the next milestone.")
	}

	return menuStyle.Render(b.String())
}
