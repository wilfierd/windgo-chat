package ui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
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
	stateMainMenu
	stateChatLobby
)

type lobbyView int

const (
	lobbyViewRooms lobbyView = iota
	lobbyViewPeople
)

var (
	// Clean, minimalist color scheme - like Claude's interface
	// No backgrounds, just simple foreground colors
	
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")) // Bright cyan for headers

	menuStyle = lipgloss.NewStyle().Padding(1, 0)

	// Selected items - just bold and colored, no background
	selectedItem = lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")). // Bright cyan/blue
		Bold(true)

	// Normal items - default terminal color
	normalItem = lipgloss.NewStyle().Foreground(lipgloss.Color("7")) // Default white/gray

	// Dimmed text for secondary info
	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Dim gray

	// Errors - just red, no bold
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // Bright red

	// Success messages
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Bright green

	// Online status - green dot
	onlineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green

	// Offline status - dim
	offlineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Dim gray

	// Help text - dimmed
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Dim gray

	// Borders - simple, no fancy styles
	borderStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("8"))

	// Separators
	separatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

var loginOptions = []string{
	"Login with email/password",
	"Login with GitHub device flow",
}

var mainMenuOptions = []string{
	"Chat Lobby",
	"My Profile",
	"Settings",
	"Logout",
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

	// Chat lobby data
	rooms         []api.Room
	filteredRooms []api.Room
	roomIndex     int

	users         []api.User
	filteredUsers []api.User
	userIndex     int

	currentView  lobbyView
	searchInput  textinput.Model
	searchActive bool

	viewport      viewport.Model
	viewportReady bool
}

// openBrowser opens the specified URL in the user's default browser
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
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

	search := textinput.New()
	search.Placeholder = "Search..."
	search.CharLimit = 50
	search.Width = 30

	return Model{
		client:        client,
		state:         stateLoading,
		emailInput:    email,
		passwordInput: password,
		searchInput:   search,
		currentView:   lobbyViewRooms,
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

type roomsLoadedMsg struct {
	rooms []api.Room
	err   error
}

type usersLoadedMsg struct {
	users []api.User
	err   error
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

func loadRoomsCmd(client *api.Client, token string) tea.Cmd {
	return func() tea.Msg {
		rooms, err := client.GetRooms(token)
		if err != nil {
			return roomsLoadedMsg{err: err}
		}
		return roomsLoadedMsg{rooms: rooms}
	}
}

func loadUsersCmd(client *api.Client, token string) tea.Cmd {
	return func() tea.Msg {
		users, err := client.GetUsers(token, "")
		if err != nil {
			return usersLoadedMsg{err: err}
		}
		return usersLoadedMsg{users: users}
	}
}

// applyFilters filters rooms and users based on search input
func (m *Model) applyFilters() {
	query := strings.ToLower(m.searchInput.Value())

	// Filter rooms
	if query == "" {
		m.filteredRooms = m.rooms
	} else {
		filtered := []api.Room{}
		for _, room := range m.rooms {
			if strings.Contains(strings.ToLower(room.Name), query) {
				filtered = append(filtered, room)
			}
		}
		m.filteredRooms = filtered
	}

	// Filter users
	if query == "" {
		m.filteredUsers = m.users
	} else {
		filtered := []api.User{}
		for _, user := range m.users {
			if strings.Contains(strings.ToLower(user.Username), query) ||
				strings.Contains(strings.ToLower(user.Email), query) {
				filtered = append(filtered, user)
			}
		}
		m.filteredUsers = filtered
	}

	// Reset indices if out of bounds
	if m.roomIndex >= len(m.filteredRooms) {
		m.roomIndex = 0
	}
	if m.userIndex >= len(m.filteredUsers) {
		m.userIndex = 0
	}
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var (
		keyMsg tea.KeyMsg
		isKey  bool
	)
	if km, ok := message.(tea.KeyMsg); ok {
		keyMsg = km
		isKey = true
	}
	switch msg := message.(type) {
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
		m.state = stateMainMenu
		m.menuIndex = 0
		m.status = "" // Clear status, the menu shows who's logged in
		return m, nil

	case deviceStartMsg:
		m.submitting = false
		m.err = nil
		m.deviceInfo = msg.resp
		m.state = stateDeviceSetup
		// Try to automatically open the browser
		if m.deviceInfo.VerificationURIComplete != "" {
			if err := openBrowser(m.deviceInfo.VerificationURIComplete); err == nil {
				m.status = "Browser opened! Authorize the app, then press Enter to continue."
			} else {
				m.status = "Enter the code in your browser, then press Enter to continue."
			}
		} else {
			if err := openBrowser(m.deviceInfo.VerificationURI); err == nil {
				m.status = "Browser opened! Enter the code, then press Enter to continue."
			} else {
				m.status = "Enter the code in your browser, then press Enter to continue."
			}
		}
		return m, nil

	case authSuccessMsg:
		m.submitting = false
		m.err = nil
		m.token = msg.resp.Token
		m.user = &msg.resp.User
		m.state = stateMainMenu
		m.menuIndex = 0
		m.status = "" // Clear status, the menu shows who's logged in
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

	case roomsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.status = "Failed to load chat rooms"
			m.state = stateMainMenu
			return m, nil
		}
		m.rooms = msg.rooms
		m.filteredRooms = msg.rooms
		m.roomIndex = 0
		m.state = stateChatLobby
		m.status = "Loading users..."
		return m, loadUsersCmd(m.client, m.token)

	case usersLoadedMsg:
		if msg.err != nil {
			// Non-critical error, users list is optional
			m.status = fmt.Sprintf("Found %d rooms. Press / to search, Tab to switch views.", len(m.rooms))
			return m, nil
		}
		m.users = msg.users
		m.filteredUsers = msg.users
		m.userIndex = 0
		m.status = fmt.Sprintf("Found %d rooms, %d users. Press / to search, Tab to switch views.", len(m.rooms), len(m.users))
		return m, nil

	}

	if isKey {
		var cmds []tea.Cmd
		if m.state == stateEmailLogin {
			skipInputs := false
			switch keyMsg.String() {
			case "enter", "tab", "shift+tab", "esc":
				skipInputs = true
			}
			if !skipInputs {
				var cmd tea.Cmd
				m.emailInput, cmd = m.emailInput.Update(message)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				m.passwordInput, cmd = m.passwordInput.Update(message)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
		// Handle search input in chat lobby
		if m.state == stateChatLobby && m.searchActive {
			skipSearch := false
			switch keyMsg.String() {
			case "esc", "enter":
				skipSearch = true
			}
			if !skipSearch {
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(message)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				m.applyFilters()
			}
		}
		var keyCmd tea.Cmd
		m, keyCmd = m.handleKey(keyMsg)
		if keyCmd != nil {
			cmds = append(cmds, keyCmd)
		}
		return m, tea.Batch(cmds...)
	}

	switch m.state {
	case stateEmailLogin:
		var cmds []tea.Cmd
		var cmd tea.Cmd
		m.emailInput, cmd = m.emailInput.Update(message)
		cmds = append(cmds, cmd)
		m.passwordInput, cmd = m.passwordInput.Update(message)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	// Handle Ctrl+C globally for all states
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

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

	case stateMainMenu:
		switch msg.String() {
		case "up", "k":
			if m.menuIndex > 0 {
				m.menuIndex--
			}
		case "down", "j":
			if m.menuIndex < len(mainMenuOptions)-1 {
				m.menuIndex++
			}
		case "enter":
			switch m.menuIndex {
			case 0: // Chat Lobby
				m.status = "Loading chat rooms..."
				m.state = stateChatLobby
				return m, tea.Batch(
					loadRoomsCmd(m.client, m.token),
					loadUsersCmd(m.client, m.token),
				)
			case 1: // My Profile
				m.status = "Profile view coming soon..."
			case 2: // Settings
				m.status = "Settings coming soon..."
			case 3: // Logout
				m.token = ""
				m.user = nil
				m.rooms = nil
				m.users = nil
				m.state = stateLoginMenu
				m.menuIndex = 0
				m.status = "Logged out successfully. Choose how you want to sign in."
				// Clear stored credentials
				_ = storage.Save(storage.Credentials{})
			}
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case stateChatLobby:
		if m.searchActive {
			switch msg.String() {
			case "esc":
				m.searchActive = false
				m.searchInput.Blur()
				m.searchInput.SetValue("")
				m.applyFilters()
			case "enter":
				m.searchActive = false
				m.searchInput.Blur()
			}
		} else {
			switch msg.String() {
			case "/":
				m.searchActive = true
				m.searchInput.Focus()
			case "tab":
				if m.currentView == lobbyViewRooms {
					m.currentView = lobbyViewPeople
				} else {
					m.currentView = lobbyViewRooms
				}
			case "up", "k":
				if m.currentView == lobbyViewRooms {
					if m.roomIndex > 0 {
						m.roomIndex--
					}
				} else {
					if m.userIndex > 0 {
						m.userIndex--
					}
				}
			case "down", "j":
				if m.currentView == lobbyViewRooms {
					if m.roomIndex < len(m.filteredRooms)-1 {
						m.roomIndex++
					}
				} else {
					if m.userIndex < len(m.filteredUsers)-1 {
						m.userIndex++
					}
				}
			case "enter":
				if m.currentView == lobbyViewRooms && len(m.filteredRooms) > 0 {
					selectedRoom := m.filteredRooms[m.roomIndex]
					m.status = fmt.Sprintf("Joining room: %s", selectedRoom.Name)
					// TODO: Transition to chat room view (future implementation)
				} else if m.currentView == lobbyViewPeople && len(m.filteredUsers) > 0 {
					selectedUser := m.filteredUsers[m.userIndex]
					m.status = fmt.Sprintf("Starting DM with: %s", selectedUser.Username)
					// TODO: Implement DM functionality
				}
			case "q":
				return m, tea.Quit
			case "m", "esc":
				m.state = stateMainMenu
				m.menuIndex = 0
				m.status = ""
				// Clear search when leaving lobby
				m.searchActive = false
				m.searchInput.SetValue("")
				m.searchInput.Blur()
			}
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

	case stateMainMenu:
		b.WriteString(titleStyle.Render("WindGo Chat"))
		b.WriteString("\n\n")
		b.WriteString(statusStyle.Render(fmt.Sprintf("Logged in as %s\n", m.user.Username)))
		b.WriteString("\n")

		for i, opt := range mainMenuOptions {
			if i == m.menuIndex {
				b.WriteString(selectedItem.Render("> " + opt))
			} else {
				b.WriteString(normalItem.Render("  " + opt))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/↓: navigate | Enter: select | q: quit"))

	case stateChatLobby:
		b.WriteString(titleStyle.Render("Chat Lobby"))
		b.WriteString(" ")
		b.WriteString(statusStyle.Render("- " + m.user.Username))
		b.WriteString("\n\n")

		// Tab selector - clean style
		if m.currentView == lobbyViewRooms {
			b.WriteString(selectedItem.Render("Rooms"))
			b.WriteString("  ")
			b.WriteString(statusStyle.Render("People"))
			b.WriteString("\n\n")
		} else {
			b.WriteString(statusStyle.Render("Rooms"))
			b.WriteString("  ")
			b.WriteString(selectedItem.Render("People"))
			b.WriteString("\n\n")
		}

		// Search bar
		if m.searchActive {
			b.WriteString("Search: " + m.searchInput.View() + "\n\n")
		} else {
			b.WriteString("Press / to search\n\n")
		}

		// Display current view
		if m.currentView == lobbyViewRooms {
			if len(m.filteredRooms) == 0 {
				if m.searchInput.Value() != "" {
					b.WriteString("No rooms match your search.")
				} else {
					b.WriteString("No chat rooms available.")
				}
			} else {
				b.WriteString(fmt.Sprintf("Available rooms (%d):\n\n", len(m.filteredRooms)))
				// Show max 15 items for scrolling simulation
				startIdx := 0
				endIdx := len(m.filteredRooms)
				if endIdx > 15 {
					// Simple viewport: show items around selection
					if m.roomIndex > 7 {
						startIdx = m.roomIndex - 7
					}
					endIdx = startIdx + 15
					if endIdx > len(m.filteredRooms) {
						endIdx = len(m.filteredRooms)
						startIdx = endIdx - 15
						if startIdx < 0 {
							startIdx = 0
						}
					}
				}
				if startIdx > 0 {
					b.WriteString("  ↑ More items above\n")
				}
				for i := startIdx; i < endIdx; i++ {
					room := m.filteredRooms[i]
					if i == m.roomIndex {
						b.WriteString(selectedItem.Render(fmt.Sprintf("> %s", room.Name)))
					} else {
						b.WriteString(fmt.Sprintf("  %s", room.Name))
					}
					b.WriteString("\n")
				}
				if endIdx < len(m.filteredRooms) {
					b.WriteString("  ↓ More items below\n")
				}
			}
		} else {
			// People view
			if len(m.filteredUsers) == 0 {
				if m.searchInput.Value() != "" {
					b.WriteString("No users match your search.")
				} else {
					b.WriteString("No users available.")
				}
			} else {
				// Count online users
				onlineCount := 0
				for _, user := range m.filteredUsers {
					if user.IsOnline {
						onlineCount++
					}
				}
				b.WriteString(fmt.Sprintf("Available users (%d, %s online):\n\n", 
					len(m.filteredUsers),
					onlineStyle.Render(fmt.Sprintf("%d", onlineCount))))
				
				// Show max 15 items for scrolling simulation
				startIdx := 0
				endIdx := len(m.filteredUsers)
				if endIdx > 15 {
					// Simple viewport: show items around selection
					if m.userIndex > 7 {
						startIdx = m.userIndex - 7
					}
					endIdx = startIdx + 15
					if endIdx > len(m.filteredUsers) {
						endIdx = len(m.filteredUsers)
						startIdx = endIdx - 15
						if startIdx < 0 {
							startIdx = 0
						}
					}
				}
				if startIdx > 0 {
					b.WriteString("  ↑ More items above\n")
				}
				for i := startIdx; i < endIdx; i++ {
					user := m.filteredUsers[i]
					
					// Status indicator
					var statusIcon string
					if user.IsOnline {
						statusIcon = onlineStyle.Render("●") // Filled dot
					} else {
						statusIcon = offlineStyle.Render("○") // Empty circle
					}
					
					// Last seen time
					var lastSeen string
					if user.LastActiveAt != nil {
						duration := time.Since(*user.LastActiveAt)
						if duration < time.Minute {
							lastSeen = "just now"
						} else if duration < time.Hour {
							lastSeen = fmt.Sprintf("%dm ago", int(duration.Minutes()))
						} else if duration < 24*time.Hour {
							lastSeen = fmt.Sprintf("%dh ago", int(duration.Hours()))
						} else {
							lastSeen = fmt.Sprintf("%dd ago", int(duration.Hours()/24))
						}
					}
					
					userLine := fmt.Sprintf("%s %s", statusIcon, user.Username)
					if lastSeen != "" && !user.IsOnline {
						userLine += " " + helpStyle.Render("("+lastSeen+")")
					} else if user.IsOnline {
						userLine += " " + onlineStyle.Render("(online)")
					}
					
					if i == m.userIndex {
						b.WriteString(selectedItem.Render("> " + userLine))
					} else {
						b.WriteString("  " + userLine)
					}
					b.WriteString("\n")
				}
				if endIdx < len(m.filteredUsers) {
					b.WriteString("  ↓ More items below\n")
				}
			}
		}

		b.WriteString("\n")
		b.WriteString(helpStyle.Render("Tab: switch view | ↑/↓: navigate | Enter: select | /: search | m/Esc: menu | q: quit"))
	}

	return menuStyle.Render(b.String())
}
