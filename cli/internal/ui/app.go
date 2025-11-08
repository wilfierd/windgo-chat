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
	stateConversation
	stateThreadView
	stateEditingMessage
	stateConfirmDelete
	stateAdminRooms
	stateCreateRoom
	stateEditRoom
	stateConfirmDeleteRoom
	stateNewDMDialog
)

type lobbyView int

const (
	lobbyViewRooms lobbyView = iota
	lobbyViewPeople
	lobbyViewDirectMessages
)

// Tab order: Group Rooms -> People -> Direct Messages -> Group Rooms (cycles)

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

var adminMenuOptions = []string{
	"Chat Lobby",
	"My Profile",
	"Settings",
	"Admin: Manage Rooms",
	"Logout",
}

// Model holds application state for the login experience.
type Model struct {
	client   *api.Client
	wsClient *api.WSClient

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

	directRooms         []api.DirectRoom
	filteredDirectRooms []api.DirectRoom
	dmIndex             int

	currentView  lobbyView
	searchInput  textinput.Model
	searchActive bool

	viewport      viewport.Model
	viewportReady bool

	// Conversation state
	currentRoom      *api.Room
	currentDMUser    *api.User
	messages         []api.Message
	messageInput     textinput.Model
	messageViewport  viewport.Model
	lastMessageID    uint
	currentPage      int     // Current page of messages loaded
	loadingMore      bool    // Whether we're loading more messages
	hasMoreMessages  bool    // Whether there are more messages to load
	lastScrollOffset float64 // Store scroll position before loading more

	// User status polling (keep for now, can be replaced with WebSocket later)
	userPollingActive bool      // Whether user status polling is active
	lastUserPollTime  time.Time // Last time we polled for user status

	// Message editing and deletion
	selectedMessageIndex int             // Currently selected message for edit/delete
	editingMessage       *api.Message    // Message being edited
	editInput            textinput.Model // Text input for editing message
	replyingTo           *api.Message    // Message being replied to

	// Thread view state
	threadParentMessage *api.Message    // Parent message of the thread being viewed
	threadMessages      []api.Message   // Messages in the current thread
	threadMessageIndex  int             // Selected message index in thread view
	threadInput         textinput.Model // Text input for thread replies

	// Admin room management
	adminRoomIndex   int             // Currently selected room in admin view
	roomNameInput    textinput.Model // Text input for room name (create/edit)
	selectedRoomID   uint            // Room ID being edited/deleted
	selectedRoomName string          // Room name being edited/deleted

	// New DM dialog
	availableUsers         []api.AvailableUser // Users available for starting DMs
	filteredAvailableUsers []api.AvailableUser // Filtered users based on search
	availableUserIndex     int                 // Currently selected user in new DM dialog
	dmSearchInput          textinput.Model     // Search input for new DM dialog
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

	messageInput := textinput.New()
	messageInput.Placeholder = "Type a message... (ESC to go back)"
	messageInput.CharLimit = 1000
	messageInput.Width = 80

	editInput := textinput.New()
	editInput.Placeholder = "Edit message..."
	editInput.CharLimit = 1000
	editInput.Width = 80

	roomNameInput := textinput.New()
	roomNameInput.Placeholder = "Room name..."
	roomNameInput.CharLimit = 50
	roomNameInput.Width = 40

	threadInput := textinput.New()
	threadInput.Placeholder = "Reply to thread... (ESC to go back)"
	threadInput.CharLimit = 1000
	threadInput.Width = 80

	dmSearchInput := textinput.New()
	dmSearchInput.Placeholder = "Search users..."
	dmSearchInput.CharLimit = 50
	dmSearchInput.Width = 30

	return Model{
		client:               client,
		state:                stateLoading,
		emailInput:           email,
		passwordInput:        password,
		searchInput:          search,
		messageInput:         messageInput,
		editInput:            editInput,
		roomNameInput:        roomNameInput,
		threadInput:          threadInput,
		dmSearchInput:        dmSearchInput,
		currentView:          lobbyViewRooms,
		selectedMessageIndex: -1,
		threadMessageIndex:   -1,
		adminRoomIndex:       -1,
		availableUserIndex:   0,
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

type directRoomsLoadedMsg struct {
	directRooms []api.DirectRoom
	err         error
}

type messagesLoadedMsg struct {
	messages []api.Message
	err      error
}

type wsConnectedMsg struct {
	err      error
	wsClient *api.WSClient
}

type wsMessageReceivedMsg struct {
	msg api.WSMessage
}

type wsRoomJoinedMsg struct {
	roomID uint
	err    error
}

type messageSentMsg struct {
	message *api.Message
	err     error
}

type moreMessagesLoadedMsg struct {
	messages []api.Message
	page     int
	err      error
}

type messageUpdatedMsg struct {
	message *api.Message
	err     error
}

type messageDeletedMsg struct {
	messageID uint
	err       error
}

type roomCreatedMsg struct {
	room *api.Room
	err  error
}

type roomUpdatedMsg struct {
	room *api.Room
	err  error
}

type roomDeletedMsg struct {
	err error
}

type userPollTickMsg time.Time

type availableUsersLoadedMsg struct {
	users []api.AvailableUser
	err   error
}

type directRoomCreatedMsg struct {
	directRoom *api.DirectRoom
	err        error
}

type roomMarkedAsReadMsg struct {
	roomID uint
	err    error
}

func (m Model) Init() tea.Cmd {
	return loadStoredCredentials()
}

// waitForWSMessage waits for the next WebSocket message
func waitForWSMessage(sub <-chan api.WSMessage) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-sub
		if !ok {
			return nil // Channel closed
		}
		return wsMessageReceivedMsg{msg: msg}
	}
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

func loadDirectRoomsCmd(client *api.Client, token string) tea.Cmd {
	return func() tea.Msg {
		directRooms, err := client.GetDirectRooms(token)
		if err != nil {
			return directRoomsLoadedMsg{err: err}
		}
		return directRoomsLoadedMsg{directRooms: directRooms}
	}
}

func loadMessagesCmd(client *api.Client, token string, roomID uint) tea.Cmd {
	return func() tea.Msg {
		messages, err := client.GetMessages(token, roomID, 1, 50)
		if err != nil {
			return messagesLoadedMsg{err: err}
		}
		return messagesLoadedMsg{messages: messages}
	}
}

func loadMoreMessagesCmd(client *api.Client, token string, roomID uint, page int) tea.Cmd {
	return func() tea.Msg {
		messages, err := client.GetMessages(token, roomID, page, 50)
		if err != nil {
			return moreMessagesLoadedMsg{page: page, err: err}
		}
		return moreMessagesLoadedMsg{messages: messages, page: page}
	}
}

func sendMessageCmd(client *api.Client, token string, roomID uint, content string, parentID *uint) tea.Cmd {
	return func() tea.Msg {
		message, err := client.SendMessage(token, roomID, content, parentID)
		if err != nil {
			return messageSentMsg{err: err}
		}
		return messageSentMsg{message: message}
	}
}

func updateMessageCmd(client *api.Client, token string, messageID uint, content string) tea.Cmd {
	return func() tea.Msg {
		message, err := client.UpdateMessage(token, messageID, content)
		if err != nil {
			return messageUpdatedMsg{err: err}
		}
		return messageUpdatedMsg{message: message}
	}
}

func deleteMessageCmd(client *api.Client, token string, messageID uint) tea.Cmd {
	return func() tea.Msg {
		err := client.DeleteMessage(token, messageID)
		if err != nil {
			return messageDeletedMsg{err: err}
		}
		return messageDeletedMsg{messageID: messageID}
	}
}

func markRoomAsReadCmd(client *api.Client, token string, roomID uint) tea.Cmd {
	return func() tea.Msg {
		err := client.MarkRoomAsRead(token, roomID)
		if err != nil {
			return roomMarkedAsReadMsg{roomID: roomID, err: err}
		}
		return roomMarkedAsReadMsg{roomID: roomID}
	}
}

func createRoomCmd(client *api.Client, token string, name string) tea.Cmd {
	return func() tea.Msg {
		room, err := client.CreateRoom(token, name)
		if err != nil {
			return roomCreatedMsg{err: err}
		}
		return roomCreatedMsg{room: room}
	}
}

func updateRoomCmd(client *api.Client, token string, roomID uint, name string) tea.Cmd {
	return func() tea.Msg {
		room, err := client.UpdateRoom(token, roomID, name)
		if err != nil {
			return roomUpdatedMsg{err: err}
		}
		return roomUpdatedMsg{room: room}
	}
}

func deleteRoomCmd(client *api.Client, token string, roomID uint) tea.Cmd {
	return func() tea.Msg {
		err := client.DeleteRoom(token, roomID)
		if err != nil {
			return roomDeletedMsg{err: err}
		}
		return roomDeletedMsg{}
	}
}

func connectWebSocketCmd(baseURL, token string) tea.Cmd {
	return func() tea.Msg {
		wsClient := api.NewWSClient(baseURL, token)

		if err := wsClient.Connect(); err != nil {
			return wsConnectedMsg{err: err, wsClient: nil}
		}

		return wsConnectedMsg{err: nil, wsClient: wsClient}
	}
}

func joinRoomCmd(wsClient *api.WSClient, roomID uint) tea.Cmd {
	return func() tea.Msg {
		if wsClient == nil || !wsClient.IsConnected() {
			return wsRoomJoinedMsg{roomID: roomID, err: errors.New("websocket not connected")}
		}
		err := wsClient.JoinRoom(roomID)
		return wsRoomJoinedMsg{roomID: roomID, err: err}
	}
}

func leaveRoomCmd(wsClient *api.WSClient, roomID uint) tea.Cmd {
	return func() tea.Msg {
		if wsClient == nil || !wsClient.IsConnected() {
			return nil
		}
		wsClient.LeaveRoom(roomID)
		return nil
	}
}

func pollUsersCmd() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return userPollTickMsg(t)
	})
}

func loadAvailableUsersCmd(client *api.Client, token string) tea.Cmd {
	return func() tea.Msg {
		users, err := client.GetAvailableUsers(token)
		if err != nil {
			return availableUsersLoadedMsg{err: err}
		}
		return availableUsersLoadedMsg{users: users}
	}
}

func createDirectRoomCmd(client *api.Client, token string, targetUserID uint) tea.Cmd {
	return func() tea.Msg {
		directRoom, err := client.CreateDirectRoom(token, targetUserID)
		if err != nil {
			return directRoomCreatedMsg{err: err}
		}
		return directRoomCreatedMsg{directRoom: directRoom}
	}
}

// applyFilters filters rooms, users, and direct rooms based on search input
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

	// Filter direct rooms
	if query == "" {
		m.filteredDirectRooms = m.directRooms
	} else {
		filtered := []api.DirectRoom{}
		for _, dm := range m.directRooms {
			if strings.Contains(strings.ToLower(dm.OtherUser.Username), query) ||
				(dm.LastMessage != nil && strings.Contains(strings.ToLower(dm.LastMessage.Content), query)) {
				filtered = append(filtered, dm)
			}
		}
		m.filteredDirectRooms = filtered
	}

	// Reset indices if out of bounds
	if m.roomIndex >= len(m.filteredRooms) {
		m.roomIndex = 0
	}
	if m.userIndex >= len(m.filteredUsers) {
		m.userIndex = 0
	}
	if m.dmIndex >= len(m.filteredDirectRooms) {
		m.dmIndex = 0
	}
}

// applyDMSearchFilter filters available users based on DM search input
func (m *Model) applyDMSearchFilter() {
	query := strings.ToLower(m.dmSearchInput.Value())

	if query == "" {
		m.filteredAvailableUsers = m.availableUsers
	} else {
		filtered := []api.AvailableUser{}
		for _, user := range m.availableUsers {
			if strings.Contains(strings.ToLower(user.Username), query) {
				filtered = append(filtered, user)
			}
		}
		m.filteredAvailableUsers = filtered
	}

	// Reset index if out of bounds
	if m.availableUserIndex >= len(m.filteredAvailableUsers) {
		m.availableUserIndex = 0
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
		// Start user status polling and connect WebSocket after successful login
		m.userPollingActive = true
		m.lastUserPollTime = time.Now()
		return m, tea.Batch(
			pollUsersCmd(),
			connectWebSocketCmd(m.client.BaseURL, m.token),
		)

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
		// Start user status polling and connect WebSocket after successful login
		m.userPollingActive = true
		m.lastUserPollTime = time.Now()
		return m, tea.Batch(
			saveCredentialsCmd(msg.resp),
			pollUsersCmd(),
			connectWebSocketCmd(m.client.BaseURL, m.token),
		)

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
			if len(m.rooms) > 0 {
				m.status = fmt.Sprintf("Found %d rooms. Press / to search, Tab to switch views.", len(m.rooms))
			}
			return m, nil
		}

		// Store current selected user if any
		var currentSelectedUsername string
		if len(m.filteredUsers) > 0 && m.userIndex < len(m.filteredUsers) {
			currentSelectedUsername = m.filteredUsers[m.userIndex].Username
		}

		m.users = msg.users
		m.applyFilters()

		// Try to restore selection to the same user
		if currentSelectedUsername != "" {
			for i, user := range m.filteredUsers {
				if user.Username == currentSelectedUsername {
					m.userIndex = i
					break
				}
			}
		}

		// Ensure index is in bounds
		if m.userIndex >= len(m.filteredUsers) {
			m.userIndex = 0
		}

		// Update currentDMUser status if we're in a DM conversation
		if m.currentDMUser != nil {
			for _, user := range msg.users {
				if user.ID == m.currentDMUser.ID {
					m.currentDMUser.IsOnline = user.IsOnline
					m.currentDMUser.LastActiveAt = user.LastActiveAt
					break
				}
			}
		}

		// Only update status message if we're entering the lobby for the first time
		if m.state != stateChatLobby {
			m.status = fmt.Sprintf("Found %d rooms, %d users. Press / to search, Tab to switch views.", len(m.rooms), len(m.users))
		}
		return m, nil

	case directRoomsLoadedMsg:
		if msg.err != nil {
			// Non-critical error, DM list is optional
			return m, nil
		}

		// Store current selected DM if any
		var currentSelectedDMID uint
		if len(m.filteredDirectRooms) > 0 && m.dmIndex < len(m.filteredDirectRooms) {
			currentSelectedDMID = m.filteredDirectRooms[m.dmIndex].ID
		}

		m.directRooms = msg.directRooms
		m.applyFilters()

		// Try to restore selection to the same DM
		if currentSelectedDMID > 0 {
			for i, dm := range m.filteredDirectRooms {
				if dm.ID == currentSelectedDMID {
					m.dmIndex = i
					break
				}
			}
		}

		// Ensure index is in bounds
		if m.dmIndex >= len(m.filteredDirectRooms) {
			m.dmIndex = 0
		}

		return m, nil

	case messagesLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.status = "Failed to load messages"
			return m, nil
		}
		m.messages = msg.messages
		m.currentPage = 1                           // Reset to page 1
		m.hasMoreMessages = len(msg.messages) >= 50 // If we got 50, there might be more
		// Track the last message ID for deduplication
		if len(m.messages) > 0 {
			m.lastMessageID = m.messages[0].ID
		}
		m.updateMessageViewport()
		m.status = ""
		return m, nil

	case messageSentMsg:
		if msg.err != nil {
			m.status = errorStyle.Render(fmt.Sprintf("Failed to send message: %v", msg.err))
			return m, nil
		}
		// Clear appropriate input on success
		if m.state == stateThreadView {
			m.threadInput.SetValue("")
		} else {
			m.messageInput.SetValue("")
		}
		m.status = ""
		// Add the new message to the list if not already there
		found := false
		for _, existing := range m.messages {
			if existing.ID == msg.message.ID {
				found = true
				break
			}
		}
		if !found {
			m.messages = append([]api.Message{*msg.message}, m.messages...)
			m.lastMessageID = msg.message.ID
			m.updateMessageViewport()
		}
		// If in thread view and this is a reply to the current thread
		if m.state == stateThreadView && m.threadParentMessage != nil {
			if msg.message.ParentID != nil && *msg.message.ParentID == m.threadParentMessage.ID {
				threadFound := false
				for _, existing := range m.threadMessages {
					if existing.ID == msg.message.ID {
						threadFound = true
						break
					}
				}
				if !threadFound {
					m.threadMessages = append(m.threadMessages, *msg.message)
				}
			}
		}
		return m, nil

	case moreMessagesLoadedMsg:
		m.loadingMore = false
		if msg.err != nil {
			m.status = errorStyle.Render(fmt.Sprintf("Failed to load more messages: %v", msg.err))
			return m, nil
		}
		if len(msg.messages) == 0 {
			m.hasMoreMessages = false
			m.status = helpStyle.Render("No more messages to load")
			return m, nil
		}
		// Store current scroll position
		scrollPercent := m.messageViewport.ScrollPercent()

		// Append older messages to the end of the array
		// (remember: messages[0] is newest, messages[len-1] is oldest)
		m.messages = append(m.messages, msg.messages...)
		m.currentPage = msg.page

		// Check if there might be more messages
		m.hasMoreMessages = len(msg.messages) >= 50

		// Update viewport content
		m.updateMessageViewport()

		// Try to restore approximate scroll position
		// Since we added content above, we need to scroll down a bit
		// to keep the user's view stable
		m.messageViewport.SetYOffset(int(float64(m.messageViewport.TotalLineCount()) * scrollPercent))

		m.status = helpStyle.Render(fmt.Sprintf("Loaded %d older messages (page %d)", len(msg.messages), msg.page))
		return m, nil

	case wsConnectedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("WebSocket connection failed: %v", msg.err)
			return m, nil
		}
		// WebSocket connected successfully
		m.wsClient = msg.wsClient
		m.status = "WebSocket connected"

		// Start listening for WebSocket messages using subscription channel
		return m, waitForWSMessage(m.wsClient.Subscribe())

	case wsMessageReceivedMsg:
		// Handle incoming WebSocket messages
		var nextCmd tea.Cmd

		if msg.msg.Type == "message" {
			// Parse the message content
			if contentMap, ok := msg.msg.Content.(map[string]interface{}); ok {
				// Convert to api.Message
				var apiMsg api.Message
				if id, ok := contentMap["id"].(float64); ok {
					apiMsg.ID = uint(id)
				}
				if userID, ok := contentMap["user_id"].(float64); ok {
					apiMsg.UserID = uint(userID)
				}
				if roomID, ok := contentMap["room_id"].(float64); ok {
					apiMsg.RoomID = uint(roomID)
				}
				if content, ok := contentMap["content"].(string); ok {
					apiMsg.Content = content
				}
				if parentID, ok := contentMap["parent_id"].(float64); ok && parentID > 0 {
					pid := uint(parentID)
					apiMsg.ParentID = &pid
				}
				if user, ok := contentMap["user"].(map[string]interface{}); ok {
					if username, ok := user["username"].(string); ok {
						apiMsg.User.Username = username
					}
					if userID, ok := user["id"].(float64); ok {
						apiMsg.User.ID = uint(userID)
					}
				}
				if createdAt, ok := contentMap["created_at"].(string); ok {
					parsedTime, err := time.Parse(time.RFC3339, createdAt)
					if err == nil {
						apiMsg.CreatedAt = parsedTime
					}
				}

				// Add message to current conversation if it's for the current room
				if m.currentRoom != nil && apiMsg.RoomID == m.currentRoom.ID {
					// Check for duplicates
					isDuplicate := false
					for _, existingMsg := range m.messages {
						if existingMsg.ID == apiMsg.ID {
							isDuplicate = true
							break
						}
					}

					if !isDuplicate {
						m.messages = append([]api.Message{apiMsg}, m.messages...)
						m.updateMessageViewport()
					}

					// If in thread view and message is a reply to current thread parent
					if m.state == stateThreadView && m.threadParentMessage != nil {
						if apiMsg.ParentID != nil && *apiMsg.ParentID == m.threadParentMessage.ID {
							// Check for duplicates in thread
							threadDuplicate := false
							for _, existingMsg := range m.threadMessages {
								if existingMsg.ID == apiMsg.ID {
									threadDuplicate = true
									break
								}
							}
							if !threadDuplicate {
								m.threadMessages = append(m.threadMessages, apiMsg)
							}
						}
					}
				}
			}
		}

		// Continue listening for more WebSocket messages
		if m.wsClient != nil && m.wsClient.IsConnected() {
			nextCmd = waitForWSMessage(m.wsClient.Subscribe())
		}

		return m, nextCmd

	case wsRoomJoinedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Failed to join room via WebSocket: %v", msg.err)
		}
		return m, nil

	case userPollTickMsg:
		// Poll for user status updates if polling is active
		if m.userPollingActive {
			// Avoid polling too frequently
			if time.Since(m.lastUserPollTime) >= 25*time.Second {
				m.lastUserPollTime = time.Now()
				return m, tea.Batch(
					loadUsersCmd(m.client, m.token),
					pollUsersCmd(),
				)
			}
			// Schedule next poll
			return m, pollUsersCmd()
		}
		return m, nil

	case messageUpdatedMsg:
		if msg.err != nil {
			m.status = errorStyle.Render(fmt.Sprintf("Failed to update message: %v", msg.err))
			if m.state == stateEditingMessage {
				m.state = stateConversation
			}
			return m, nil
		}
		// Update the message in the main list
		for i, existingMsg := range m.messages {
			if existingMsg.ID == msg.message.ID {
				m.messages[i] = *msg.message
				break
			}
		}
		// Update in thread list if applicable
		if m.state == stateThreadView {
			for i, existingMsg := range m.threadMessages {
				if existingMsg.ID == msg.message.ID {
					m.threadMessages[i] = *msg.message
					break
				}
			}
		}
		m.updateMessageViewport()
		m.status = successStyle.Render("Message updated successfully")
		if m.state == stateEditingMessage {
			if m.state == stateThreadView {
				m.state = stateThreadView
			} else {
				m.state = stateConversation
			}
		}
		m.editingMessage = nil
		m.selectedMessageIndex = -1
		m.messageInput.Focus()
		return m, nil

	case messageDeletedMsg:
		if msg.err != nil {
			m.status = errorStyle.Render(fmt.Sprintf("Failed to delete message: %v", msg.err))
			if m.state == stateConfirmDelete {
				m.state = stateConversation
			}
			return m, nil
		}
		// Remove the message from the main list
		for i, existingMsg := range m.messages {
			if existingMsg.ID == msg.messageID {
				m.messages = append(m.messages[:i], m.messages[i+1:]...)
				break
			}
		}
		// Remove from thread list if applicable
		if m.state == stateThreadView {
			for i, existingMsg := range m.threadMessages {
				if existingMsg.ID == msg.messageID {
					m.threadMessages = append(m.threadMessages[:i], m.threadMessages[i+1:]...)
					break
				}
			}
		}
		m.updateMessageViewport()
		m.status = successStyle.Render("Message deleted successfully")
		if m.state == stateConfirmDelete {
			if m.threadInput.Focused() || len(m.threadMessages) > 0 {
				m.state = stateThreadView
			} else {
				m.state = stateConversation
			}
		}
		m.selectedMessageIndex = -1
		m.messageInput.Focus()
		return m, nil

	case roomCreatedMsg:
		if msg.err != nil {
			m.status = errorStyle.Render(fmt.Sprintf("Failed to create room: %v", msg.err))
			m.state = stateCreateRoom
			return m, nil
		}
		m.status = successStyle.Render(fmt.Sprintf("Room '%s' created successfully!", msg.room.Name))
		m.state = stateAdminRooms
		m.roomNameInput.SetValue("")
		return m, loadRoomsCmd(m.client, m.token)

	case roomUpdatedMsg:
		if msg.err != nil {
			m.status = errorStyle.Render(fmt.Sprintf("Failed to update room: %v", msg.err))
			m.state = stateEditRoom
			return m, nil
		}
		m.status = successStyle.Render(fmt.Sprintf("Room '%s' updated successfully!", msg.room.Name))
		m.state = stateAdminRooms
		m.roomNameInput.SetValue("")
		return m, loadRoomsCmd(m.client, m.token)

	case roomDeletedMsg:
		if msg.err != nil {
			m.status = errorStyle.Render(fmt.Sprintf("Failed to delete room: %v", msg.err))
			m.state = stateConfirmDeleteRoom
			return m, nil
		}
		m.status = successStyle.Render("Room deleted successfully!")
		m.state = stateAdminRooms
		return m, loadRoomsCmd(m.client, m.token)

	case availableUsersLoadedMsg:
		if msg.err != nil {
			m.status = errorStyle.Render(fmt.Sprintf("Failed to load users: %v", msg.err))
			m.state = stateChatLobby
			return m, nil
		}
		m.availableUsers = msg.users
		m.applyDMSearchFilter()
		m.availableUserIndex = 0
		return m, nil

	case directRoomCreatedMsg:
		if msg.err != nil {
			m.status = errorStyle.Render(fmt.Sprintf("Failed to create DM: %v", msg.err))
			return m, nil
		}
		// Convert DirectRoom to Room for conversation view
		dmRoom := api.Room{
			ID:        msg.directRoom.ID,
			Name:      msg.directRoom.Name,
			Type:      msg.directRoom.Type,
			CreatedAt: msg.directRoom.CreatedAt,
			UpdatedAt: msg.directRoom.UpdatedAt,
		}
		m.currentRoom = &dmRoom
		m.currentDMUser = &msg.directRoom.OtherUser
		m.state = stateConversation
		m.status = fmt.Sprintf("Opening DM with: %s", msg.directRoom.OtherUser.Username)
		m.messages = nil
		m.messageInput.SetValue("")
		m.messageInput.Focus()
		m.currentPage = 1
		m.hasMoreMessages = true
		// Refresh DM list and load messages
		return m, tea.Batch(
			loadMessagesCmd(m.client, m.token, msg.directRoom.ID),
			joinRoomCmd(m.wsClient, msg.directRoom.ID),
			loadDirectRoomsCmd(m.client, m.token),
		)

	case tea.WindowSizeMsg:
		if !m.viewportReady {
			m.viewport = viewport.New(msg.Width, msg.Height-10)
			m.messageViewport = viewport.New(msg.Width-4, msg.Height-8)
			m.viewportReady = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 10
			m.messageViewport.Width = msg.Width - 4
			m.messageViewport.Height = msg.Height - 8
		}
		if m.state == stateConversation {
			m.updateMessageViewport()
		}
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
		// Handle message input in conversation
		if m.state == stateConversation {
			skipMessageInput := false
			// Only skip command keys when input is not focused (i.e., when navigating messages)
			// When input is focused (typing/replying), allow all keys through except tab, esc, enter
			if !m.messageInput.Focused() {
				switch keyMsg.String() {
				case "tab", "esc", "enter", "up", "k", "down", "j", "pgup", "pgdown", "r", "e", "d":
					skipMessageInput = true
				}
			} else {
				// When focused (including reply mode), skip tab, esc, enter (they trigger actions in handleKey)
				switch keyMsg.String() {
				case "tab", "esc", "enter":
					skipMessageInput = true
				}
			}
			if !skipMessageInput {
				var cmd tea.Cmd
				m.messageInput, cmd = m.messageInput.Update(message)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
		// Handle edit input in editing state
		if m.state == stateEditingMessage {
			skipEditInput := false
			switch keyMsg.String() {
			case "esc", "enter":
				skipEditInput = true
			}
			if !skipEditInput {
				var cmd tea.Cmd
				m.editInput, cmd = m.editInput.Update(message)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
		// Handle room name input in create/edit room states
		if m.state == stateCreateRoom || m.state == stateEditRoom {
			skipRoomInput := false
			switch keyMsg.String() {
			case "esc", "enter":
				skipRoomInput = true
			}
			if !skipRoomInput {
				var cmd tea.Cmd
				m.roomNameInput, cmd = m.roomNameInput.Update(message)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
		// Handle thread input in thread view state
		if m.state == stateThreadView {
			skipThreadInput := false
			switch keyMsg.String() {
			case "esc", "enter", "tab", "up", "down", "k", "j", "e", "d":
				skipThreadInput = true
			}
			if !skipThreadInput && m.threadInput.Focused() {
				var cmd tea.Cmd
				m.threadInput, cmd = m.threadInput.Update(message)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
		// Handle DM search input in new DM dialog
		if m.state == stateNewDMDialog {
			skipDMSearch := false
			switch keyMsg.String() {
			case "esc", "enter", "up", "down", "k", "j":
				skipDMSearch = true
			}
			if !skipDMSearch {
				var cmd tea.Cmd
				m.dmSearchInput, cmd = m.dmSearchInput.Update(message)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				m.applyDMSearchFilter()
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
		// Determine which menu options to use based on user role
		menuOpts := mainMenuOptions
		if m.user != nil && m.user.Role == "admin" {
			menuOpts = adminMenuOptions
		}

		switch msg.String() {
		case "up", "k":
			if m.menuIndex > 0 {
				m.menuIndex--
			}
		case "down", "j":
			if m.menuIndex < len(menuOpts)-1 {
				m.menuIndex++
			}
		case "enter":
			// Handle admin menu differently
			if m.user != nil && m.user.Role == "admin" {
				switch m.menuIndex {
				case 0: // Chat Lobby
					m.status = "Loading chat rooms..."
					m.state = stateChatLobby
					return m, tea.Batch(
						loadRoomsCmd(m.client, m.token),
						loadUsersCmd(m.client, m.token),
						loadDirectRoomsCmd(m.client, m.token),
					)
				case 1: // My Profile
					m.status = "Profile view coming soon..."
				case 2: // Settings
					m.status = "Settings coming soon..."
				case 3: // Admin: Manage Rooms
					m.status = "Loading rooms for management..."
					m.state = stateAdminRooms
					m.adminRoomIndex = 0
					return m, loadRoomsCmd(m.client, m.token)
				case 4: // Logout
					m.token = ""
					m.user = nil
					m.rooms = nil
					m.users = nil
					m.state = stateLoginMenu
					m.menuIndex = 0
					m.status = "Logged out successfully. Choose how you want to sign in."
					// Stop polling, disconnect WebSocket, and clear stored credentials
					m.userPollingActive = false
					if m.wsClient != nil {
						m.wsClient.Disconnect()
						m.wsClient = nil
					}
					_ = storage.Save(storage.Credentials{})
					return m, nil
				}
			} else {
				// Regular user menu
				switch m.menuIndex {
				case 0: // Chat Lobby
					m.status = "Loading chat rooms..."
					m.state = stateChatLobby
					return m, tea.Batch(
						loadRoomsCmd(m.client, m.token),
						loadUsersCmd(m.client, m.token),
						loadDirectRoomsCmd(m.client, m.token),
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
					// Stop polling, disconnect WebSocket, and clear stored credentials
					m.userPollingActive = false
					if m.wsClient != nil {
						m.wsClient.Disconnect()
						m.wsClient = nil
					}
					_ = storage.Save(storage.Credentials{})
					return m, nil
				}
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
			case "n":
				// Open new DM dialog only when in Direct Messages tab
				if m.currentView == lobbyViewDirectMessages {
					m.state = stateNewDMDialog
					m.status = "Loading available users..."
					m.dmSearchInput.SetValue("")
					m.dmSearchInput.Focus()
					m.availableUserIndex = 0
					return m, loadAvailableUsersCmd(m.client, m.token)
				}
			case "tab":
				// Cycle through: Rooms -> People -> Direct Messages -> Rooms
				switch m.currentView {
				case lobbyViewRooms:
					m.currentView = lobbyViewPeople
				case lobbyViewPeople:
					m.currentView = lobbyViewDirectMessages
				case lobbyViewDirectMessages:
					m.currentView = lobbyViewRooms
				}
			case "up", "k":
				switch m.currentView {
				case lobbyViewRooms:
					if m.roomIndex > 0 {
						m.roomIndex--
					}
				case lobbyViewPeople:
					if m.userIndex > 0 {
						m.userIndex--
					}
				case lobbyViewDirectMessages:
					if m.dmIndex > 0 {
						m.dmIndex--
					}
				}
			case "down", "j":
				switch m.currentView {
				case lobbyViewRooms:
					if m.roomIndex < len(m.filteredRooms)-1 {
						m.roomIndex++
					}
				case lobbyViewPeople:
					if m.userIndex < len(m.filteredUsers)-1 {
						m.userIndex++
					}
				case lobbyViewDirectMessages:
					if m.dmIndex < len(m.filteredDirectRooms)-1 {
						m.dmIndex++
					}
				}
			case "enter":
				switch m.currentView {
				case lobbyViewRooms:
					if len(m.filteredRooms) > 0 {
						selectedRoom := m.filteredRooms[m.roomIndex]
						m.currentRoom = &selectedRoom
						m.currentDMUser = nil
						m.state = stateConversation
						m.status = fmt.Sprintf("Loading room: %s", selectedRoom.Name)
						m.messages = nil
						m.messageInput.SetValue("")
						m.messageInput.Focus()
						m.currentPage = 1
						m.hasMoreMessages = true
						// Load initial messages via REST, join room via WebSocket, and mark as read
						return m, tea.Batch(
							loadMessagesCmd(m.client, m.token, selectedRoom.ID),
							joinRoomCmd(m.wsClient, selectedRoom.ID),
							markRoomAsReadCmd(m.client, m.token, selectedRoom.ID),
						)
					}
				case lobbyViewPeople:
					if len(m.filteredUsers) > 0 {
						selectedUser := m.filteredUsers[m.userIndex]
						m.status = fmt.Sprintf("Starting DM with: %s (DM not yet implemented)", selectedUser.Username)
						// TODO: Implement DM functionality - needs backend support
					}
				case lobbyViewDirectMessages:
					if len(m.filteredDirectRooms) > 0 {
						selectedDM := m.filteredDirectRooms[m.dmIndex]
						// Convert DirectRoom to Room for conversation view
						dmRoom := api.Room{
							ID:        selectedDM.ID,
							Name:      selectedDM.Name,
							Type:      selectedDM.Type,
							CreatedAt: selectedDM.CreatedAt,
							UpdatedAt: selectedDM.UpdatedAt,
						}
						m.currentRoom = &dmRoom
						m.currentDMUser = &selectedDM.OtherUser
						m.state = stateConversation
						m.status = fmt.Sprintf("Loading DM with: %s", selectedDM.OtherUser.Username)
						m.messages = nil
						m.messageInput.SetValue("")
						m.messageInput.Focus()
						m.currentPage = 1
						m.hasMoreMessages = true
						// Load initial messages via REST, join room via WebSocket, and mark as read
						return m, tea.Batch(
							loadMessagesCmd(m.client, m.token, selectedDM.ID),
							joinRoomCmd(m.wsClient, selectedDM.ID),
							markRoomAsReadCmd(m.client, m.token, selectedDM.ID),
						)
					}
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

	case stateConversation:
		switch msg.String() {
		case "tab":
			// Toggle between typing mode and navigation mode
			if m.messageInput.Focused() {
				// Switch to navigation mode: blur input, select newest message
				m.messageInput.Blur()
				if len(m.messages) > 0 {
					m.selectedMessageIndex = 0
					m.status = helpStyle.Render("Navigation mode: use ↑/↓ to navigate messages")
				}
				m.updateMessageViewport()
			} else {
				// Switch to typing mode: focus input, deselect message
				m.selectedMessageIndex = -1
				m.messageInput.Focus()
				m.status = ""
				m.updateMessageViewport()
			}
		case "esc":
			// If replying to a message, cancel reply mode
			if m.replyingTo != nil {
				m.replyingTo = nil
				m.messageInput.Placeholder = "Type a message... (ESC to go back)"
				m.messageInput.Focus()
				m.status = ""
				return m, nil
			}
			// If a message is selected, deselect it
			if m.selectedMessageIndex >= 0 {
				m.selectedMessageIndex = -1
				m.status = ""
				m.updateMessageViewport()
				return m, nil
			}
			// Otherwise, exit conversation and leave room via WebSocket
			var cmd tea.Cmd
			if m.currentRoom != nil && m.wsClient != nil {
				cmd = leaveRoomCmd(m.wsClient, m.currentRoom.ID)
			}
			m.state = stateChatLobby
			m.currentRoom = nil
			m.currentDMUser = nil
			m.messages = nil
			m.status = ""
			m.messageInput.SetValue("")
			m.messageInput.Placeholder = "Type a message... (ESC to go back)"
			m.messageInput.Blur()
			m.selectedMessageIndex = -1
			m.replyingTo = nil
			return m, cmd
		case "enter":
			// If a message is selected and it has replies, enter thread view
			if m.selectedMessageIndex >= 0 && m.selectedMessageIndex < len(m.messages) {
				selectedMsg := m.messages[m.selectedMessageIndex]
				// Check if this message has replies
				hasReplies := false
				for _, msg := range m.messages {
					if msg.ParentID != nil && *msg.ParentID == selectedMsg.ID {
						hasReplies = true
						break
					}
				}
				if hasReplies {
					// Enter thread view
					m.threadParentMessage = &selectedMsg
					m.threadMessages = []api.Message{selectedMsg}
					// Collect all replies
					for _, msg := range m.messages {
						if msg.ParentID != nil && *msg.ParentID == selectedMsg.ID {
							m.threadMessages = append(m.threadMessages, msg)
						}
					}
					m.state = stateThreadView
					m.threadInput.Focus()
					m.threadMessageIndex = -1
					m.status = fmt.Sprintf("Thread: %d replies", len(m.threadMessages)-1)
					return m, nil
				}
			}
			// Otherwise, send message
			content := strings.TrimSpace(m.messageInput.Value())
			if content != "" && m.currentRoom != nil {
				// Check for commands
				if strings.HasPrefix(content, "/") {
					m.handleCommand(content)
				} else {
					// Include parent ID if replying to a message
					var parentID *uint
					if m.replyingTo != nil {
						parentID = &m.replyingTo.ID
					}
					cmd := sendMessageCmd(m.client, m.token, m.currentRoom.ID, content, parentID)
					// Clear reply context after sending and keep input focused
					m.replyingTo = nil
					m.messageInput.Placeholder = "Type a message... (ESC to go back)"
					m.messageInput.Focus()
					return m, cmd
				}
			}
		case "up", "k":
			// Navigation only works when input is not focused (use Tab to switch modes)
			if m.messageInput.Focused() {
				return m, nil
			}
			// If no message is selected, select the newest message (bottom of screen)
			if m.selectedMessageIndex == -1 && len(m.messages) > 0 {
				m.selectedMessageIndex = 0
				m.messageInput.Blur() // Blur input when selecting messages
				m.replyingTo = nil    // Clear reply context when navigating
				m.messageInput.Placeholder = "Type a message... (ESC to go back)"
				// Show different help based on message ownership
				selectedMsg := m.messages[m.selectedMessageIndex]
				if selectedMsg.UserID == m.user.ID {
					m.status = helpStyle.Render("Press 'r' to reply, 'e' to edit, 'd' to delete, or arrow keys to navigate")
				} else {
					m.status = helpStyle.Render("Press 'r' to reply or arrow keys to navigate")
				}
				m.updateMessageViewport() // Refresh viewport to show selection
			} else if m.selectedMessageIndex < len(m.messages)-1 {
				// Move UP on screen = older message = higher index
				m.selectedMessageIndex++
				// Update status based on newly selected message
				selectedMsg := m.messages[m.selectedMessageIndex]
				if selectedMsg.UserID == m.user.ID {
					m.status = helpStyle.Render("Press 'r' to reply, 'e' to edit, 'd' to delete, or arrow keys to navigate")
				} else {
					m.status = helpStyle.Render("Press 'r' to reply or arrow keys to navigate")
				}
				m.updateMessageViewport() // Refresh viewport to show selection
			} else {
				// Already at oldest message, scroll viewport
				m.messageViewport.LineUp(1)
				// Check if we're at the top and should load more messages
				if m.messageViewport.AtTop() && !m.loadingMore && m.hasMoreMessages && m.currentRoom != nil {
					m.loadingMore = true
					m.status = helpStyle.Render("Loading older messages...")
					return m, loadMoreMessagesCmd(m.client, m.token, m.currentRoom.ID, m.currentPage+1)
				}
			}
		case "down", "j":
			// Navigation only works when input is not focused (use Tab to switch modes)
			if m.messageInput.Focused() {
				return m, nil
			}
			if m.selectedMessageIndex > 0 {
				// Move DOWN on screen = newer message = lower index
				m.selectedMessageIndex--
				// Update status based on newly selected message
				selectedMsg := m.messages[m.selectedMessageIndex]
				if selectedMsg.UserID == m.user.ID {
					m.status = helpStyle.Render("Press 'r' to reply, 'e' to edit, 'd' to delete, or arrow keys to navigate")
				} else {
					m.status = helpStyle.Render("Press 'r' to reply or arrow keys to navigate")
				}
				m.updateMessageViewport() // Refresh viewport to show selection
			} else {
				// At newest message or no selection, just scroll viewport
				m.messageViewport.LineDown(1)
			}
		case "r":
			// Reply to selected message
			if m.selectedMessageIndex >= 0 && m.selectedMessageIndex < len(m.messages) {
				selectedMsg := m.messages[m.selectedMessageIndex]
				m.replyingTo = &selectedMsg
				m.selectedMessageIndex = -1
				// Ensure message input is focused for typing reply
				m.messageInput.Focus()
				// Update placeholder to show reply context
				replyTo := selectedMsg.User.Username
				if selectedMsg.UserID == m.user.ID {
					replyTo = "yourself"
				}
				m.messageInput.Placeholder = fmt.Sprintf("Replying to %s... (ESC to cancel)", replyTo)
				m.status = successStyle.Render(fmt.Sprintf("↳ Replying to: %s", selectedMsg.Content[:min(50, len(selectedMsg.Content))]))
			}
		case "e":
			// Edit selected message if it belongs to the user
			if m.selectedMessageIndex >= 0 && m.selectedMessageIndex < len(m.messages) {
				selectedMsg := m.messages[m.selectedMessageIndex]
				if selectedMsg.UserID == m.user.ID {
					m.editingMessage = &selectedMsg
					m.editInput.SetValue(selectedMsg.Content)
					m.editInput.Focus()
					m.state = stateEditingMessage
					m.status = "Editing message... (ESC to cancel, Enter to save)"
				} else {
					m.status = errorStyle.Render("You can only edit your own messages")
				}
			}
		case "d":
			// Delete selected message if it belongs to the user
			if m.selectedMessageIndex >= 0 && m.selectedMessageIndex < len(m.messages) {
				selectedMsg := m.messages[m.selectedMessageIndex]
				if selectedMsg.UserID == m.user.ID {
					m.editingMessage = &selectedMsg
					m.state = stateConfirmDelete
					m.status = "Delete this message? (y/n)"
				} else {
					m.status = errorStyle.Render("You can only delete your own messages")
				}
			}
		case "pgup":
			m.selectedMessageIndex = -1
			m.replyingTo = nil // Clear reply context
			m.messageInput.Placeholder = "Type a message... (ESC to go back)"
			m.messageInput.Focus() // Refocus input when deselecting
			m.status = ""
			m.updateMessageViewport() // Refresh viewport to clear selection
			m.messageViewport.ViewUp()
			// Check if we're at the top after page up
			if m.messageViewport.AtTop() && !m.loadingMore && m.hasMoreMessages && m.currentRoom != nil {
				m.loadingMore = true
				m.status = helpStyle.Render("Loading older messages...")
				return m, loadMoreMessagesCmd(m.client, m.token, m.currentRoom.ID, m.currentPage+1)
			}
		case "pgdown":
			m.selectedMessageIndex = -1
			m.replyingTo = nil // Clear reply context
			m.messageInput.Placeholder = "Type a message... (ESC to go back)"
			m.messageInput.Focus() // Refocus input when deselecting
			m.status = ""
			m.updateMessageViewport() // Refresh viewport to clear selection
			m.messageViewport.ViewDown()
		}

	case stateEditingMessage:
		switch msg.String() {
		case "esc":
			m.state = stateConversation
			m.editingMessage = nil
			m.editInput.Blur()
			m.messageInput.Focus() // Refocus message input when canceling edit
			m.status = ""
		case "enter":
			content := strings.TrimSpace(m.editInput.Value())
			if content != "" && m.editingMessage != nil {
				return m, updateMessageCmd(m.client, m.token, m.editingMessage.ID, content)
			}
		}

	case stateConfirmDelete:
		switch msg.String() {
		case "y", "Y":
			if m.editingMessage != nil {
				return m, deleteMessageCmd(m.client, m.token, m.editingMessage.ID)
			}
		case "n", "N", "esc":
			m.state = stateConversation
			m.editingMessage = nil
			m.messageInput.Focus() // Refocus message input when canceling delete
			m.status = ""
		}

	case stateAdminRooms:
		switch msg.String() {
		case "up", "k":
			if m.adminRoomIndex > 0 {
				m.adminRoomIndex--
			}
		case "down", "j":
			if m.adminRoomIndex < len(m.rooms)-1 {
				m.adminRoomIndex++
			}
		case "c":
			// Create new room
			m.state = stateCreateRoom
			m.roomNameInput.SetValue("")
			m.roomNameInput.Focus()
			m.status = "Enter a name for the new room"
		case "e":
			// Edit selected room
			if len(m.rooms) > 0 && m.adminRoomIndex < len(m.rooms) {
				selectedRoom := m.rooms[m.adminRoomIndex]
				m.selectedRoomID = selectedRoom.ID
				m.selectedRoomName = selectedRoom.Name
				m.roomNameInput.SetValue(selectedRoom.Name)
				m.roomNameInput.Focus()
				m.state = stateEditRoom
				m.status = fmt.Sprintf("Editing room: %s", selectedRoom.Name)
			}
		case "d":
			// Delete selected room
			if len(m.rooms) > 0 && m.adminRoomIndex < len(m.rooms) {
				selectedRoom := m.rooms[m.adminRoomIndex]
				m.selectedRoomID = selectedRoom.ID
				m.selectedRoomName = selectedRoom.Name
				m.state = stateConfirmDeleteRoom
				m.status = fmt.Sprintf("Delete room '%s'? (y/n)", selectedRoom.Name)
			}
		case "esc":
			m.state = stateMainMenu
			m.status = ""
			m.adminRoomIndex = 0
		}

	case stateCreateRoom:
		switch msg.String() {
		case "enter":
			roomName := strings.TrimSpace(m.roomNameInput.Value())
			if roomName == "" {
				m.status = errorStyle.Render("Room name cannot be empty")
				return m, nil
			}
			m.status = "Creating room..."
			return m, createRoomCmd(m.client, m.token, roomName)
		case "esc":
			m.state = stateAdminRooms
			m.roomNameInput.SetValue("")
			m.roomNameInput.Blur()
			m.status = ""
		}

	case stateEditRoom:
		switch msg.String() {
		case "enter":
			roomName := strings.TrimSpace(m.roomNameInput.Value())
			if roomName == "" {
				m.status = errorStyle.Render("Room name cannot be empty")
				return m, nil
			}
			m.status = "Updating room..."
			return m, updateRoomCmd(m.client, m.token, m.selectedRoomID, roomName)
		case "esc":
			m.state = stateAdminRooms
			m.roomNameInput.SetValue("")
			m.roomNameInput.Blur()
			m.status = ""
		}

	case stateConfirmDeleteRoom:
		switch msg.String() {
		case "y", "Y":
			m.status = "Deleting room..."
			return m, deleteRoomCmd(m.client, m.token, m.selectedRoomID)
		case "n", "N", "esc":
			m.state = stateAdminRooms
			m.status = ""
		}

	case stateThreadView:
		switch msg.String() {
		case "up", "k":
			if m.threadInput.Focused() {
				return m, nil
			}
			if m.threadMessageIndex > 0 {
				m.threadMessageIndex--
				selectedMsg := m.threadMessages[m.threadMessageIndex]
				if selectedMsg.UserID == m.user.ID {
					m.status = helpStyle.Render("Press 'e' to edit, 'd' to delete, or arrow keys to navigate")
				} else {
					m.status = helpStyle.Render("Arrow keys to navigate")
				}
			}
		case "down", "j":
			if m.threadInput.Focused() {
				return m, nil
			}
			if m.threadMessageIndex < len(m.threadMessages)-1 {
				m.threadMessageIndex++
				selectedMsg := m.threadMessages[m.threadMessageIndex]
				if selectedMsg.UserID == m.user.ID {
					m.status = helpStyle.Render("Press 'e' to edit, 'd' to delete, or arrow keys to navigate")
				} else {
					m.status = helpStyle.Render("Arrow keys to navigate")
				}
			}
		case "tab":
			if m.threadInput.Focused() {
				m.threadInput.Blur()
				m.threadMessageIndex = 0
			} else {
				m.threadInput.Focus()
			}
		case "enter":
			if !m.threadInput.Focused() {
				return m, nil
			}
			content := strings.TrimSpace(m.threadInput.Value())
			if content == "" {
				return m, nil
			}
			if m.currentRoom == nil {
				return m, nil
			}
			parentID := m.threadParentMessage.ID
			m.status = "Sending reply..."
			return m, sendMessageCmd(m.client, m.token, m.currentRoom.ID, content, &parentID)
		case "e":
			if m.threadInput.Focused() {
				return m, nil
			}
			if m.threadMessageIndex >= 0 && m.threadMessageIndex < len(m.threadMessages) {
				selectedMsg := m.threadMessages[m.threadMessageIndex]
				if selectedMsg.UserID == m.user.ID {
					m.state = stateEditingMessage
					m.editingMessage = &selectedMsg
					m.editInput.SetValue(selectedMsg.Content)
					m.editInput.Focus()
					m.status = "Editing reply..."
				}
			}
		case "d":
			if m.threadInput.Focused() {
				return m, nil
			}
			if m.threadMessageIndex >= 0 && m.threadMessageIndex < len(m.threadMessages) {
				selectedMsg := m.threadMessages[m.threadMessageIndex]
				if selectedMsg.UserID == m.user.ID {
					m.state = stateConfirmDelete
					m.editingMessage = &selectedMsg
					m.status = "Delete this reply? (y/n)"
				}
			}
		case "esc":
			m.state = stateConversation
			m.threadParentMessage = nil
			m.threadMessages = nil
			m.threadMessageIndex = 0
			m.threadInput.SetValue("")
			m.threadInput.Blur()
			m.messageInput.Focus()
			m.status = ""
		}

	case stateNewDMDialog:
		switch msg.String() {
		case "up", "k":
			if m.availableUserIndex > 0 {
				m.availableUserIndex--
			}
		case "down", "j":
			if m.availableUserIndex < len(m.filteredAvailableUsers)-1 {
				m.availableUserIndex++
			}
		case "enter":
			// Create or open DM with selected user
			if len(m.filteredAvailableUsers) > 0 && m.availableUserIndex < len(m.filteredAvailableUsers) {
				selectedUser := m.filteredAvailableUsers[m.availableUserIndex]
				m.status = fmt.Sprintf("Creating DM with %s...", selectedUser.Username)
				m.dmSearchInput.Blur()
				return m, createDirectRoomCmd(m.client, m.token, selectedUser.ID)
			}
		case "esc":
			// Cancel and return to lobby
			m.state = stateChatLobby
			m.dmSearchInput.SetValue("")
			m.dmSearchInput.Blur()
			m.status = ""
			m.availableUsers = nil
			m.filteredAvailableUsers = nil
			m.availableUserIndex = 0
		}
	}

	return m, nil
}

func (m *Model) handleCommand(cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	command := strings.ToLower(parts[0])
	switch command {
	case "/vault":
		m.status = helpStyle.Render("🔒 Vault feature coming soon...")
	case "/help":
		m.status = helpStyle.Render("Commands: /vault (coming soon), /help, /back, /quit | ESC to go back")
	case "/back":
		if m.currentRoom != nil && m.wsClient != nil {
			m.wsClient.LeaveRoom(m.currentRoom.ID)
		}
		m.state = stateChatLobby
		m.currentRoom = nil
		m.messages = nil
		m.messageInput.SetValue("")
	case "/quit":
		if m.wsClient != nil {
			m.wsClient.Disconnect()
		}
	default:
		m.status = errorStyle.Render(fmt.Sprintf("Unknown command: %s", command))
	}
	m.messageInput.SetValue("")
}

func (m *Model) updateMessageViewport() {
	// Can update viewport for both group rooms and DMs
	if m.currentRoom == nil {
		return
	}

	var b strings.Builder

	// Build a map of message ID to replies for threading
	repliesMap := make(map[uint][]int) // parent_id -> []message_indices
	topLevel := []int{}                // indices of top-level messages (no parent)

	for i := 0; i < len(m.messages); i++ {
		msg := m.messages[i]
		if msg.ParentID != nil {
			repliesMap[*msg.ParentID] = append(repliesMap[*msg.ParentID], i)
		} else {
			topLevel = append(topLevel, i)
		}
	}

	// Render only top-level messages (no threading in main view)
	for i := len(topLevel) - 1; i >= 0; i-- {
		msgIndex := topLevel[i]
		if msgIndex < 0 || msgIndex >= len(m.messages) {
			continue
		}

		msg := m.messages[msgIndex]
		timestamp := msg.CreatedAt.Format("15:04")
		username := msg.User.Username
		if msg.UserID == m.user.ID {
			username = "You"
		}

		// Check if this message is selected
		isSelected := m.selectedMessageIndex == msgIndex

		// Build message line
		var messageLine strings.Builder
		messageLine.WriteString(helpStyle.Render(timestamp))
		messageLine.WriteString(" ")
		messageLine.WriteString(selectedItem.Render(username))
		messageLine.WriteString(": ")
		messageLine.WriteString(msg.Content)

		// Add thread indicator if this message has replies
		replyCount := len(repliesMap[msg.ID])
		if replyCount > 0 {
			messageLine.WriteString(" ")
			messageLine.WriteString(helpStyle.Render(fmt.Sprintf("💬 %d", replyCount)))
		}

		// Highlight if selected
		if isSelected {
			b.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("11")). // Bright yellow
				Bold(true).
				Render("> " + messageLine.String()))
		} else {
			b.WriteString("  " + messageLine.String())
		}
		b.WriteString("\n")
	}

	m.messageViewport.SetContent(b.String())
	// Scroll to bottom to show newest messages
	m.messageViewport.GotoBottom()
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
		b.WriteString(statusStyle.Render(fmt.Sprintf("Logged in as %s", m.user.Username)))
		if m.user != nil && m.user.Role == "admin" {
			b.WriteString(statusStyle.Render(" (Admin)"))
		}
		b.WriteString("\n\n")

		// Use appropriate menu based on role
		menuOpts := mainMenuOptions
		if m.user != nil && m.user.Role == "admin" {
			menuOpts = adminMenuOptions
		}

		for i, opt := range menuOpts {
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

		// Tab selector - clean style with three tabs
		// Group Rooms tab
		if m.currentView == lobbyViewRooms {
			b.WriteString(selectedItem.Render("Group Rooms"))
		} else {
			b.WriteString(statusStyle.Render("Group Rooms"))
		}
		b.WriteString("  ")
		
		// Direct Messages tab
		if m.currentView == lobbyViewDirectMessages {
			b.WriteString(selectedItem.Render("Direct Messages"))
		} else {
			b.WriteString(statusStyle.Render("Direct Messages"))
		}
		b.WriteString("  ")
		
		// People tab
		if m.currentView == lobbyViewPeople {
			b.WriteString(selectedItem.Render("People"))
		} else {
			b.WriteString(statusStyle.Render("People"))
		}
		b.WriteString("\n\n")

		// Search bar
		if m.searchActive {
			b.WriteString("Search: " + m.searchInput.View() + "\n\n")
		} else {
			b.WriteString("Press / to search\n\n")
		}

		// Display current view
		switch m.currentView {
		case lobbyViewRooms:
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
					roomDisplay := room.Name

					// Add unread badge if there are unread messages
					if room.UnreadCount > 0 {
						roomDisplay += " " + errorStyle.Render(fmt.Sprintf("(%d)", room.UnreadCount))
					}

					if i == m.roomIndex {
						b.WriteString(selectedItem.Render(fmt.Sprintf("> %s", roomDisplay)))
					} else {
						b.WriteString(fmt.Sprintf("  %s", roomDisplay))
					}
					b.WriteString("\n")
				}
				if endIdx < len(m.filteredRooms) {
					b.WriteString("  ↓ More items below\n")
				}
			}
		
		case lobbyViewDirectMessages:
			// Direct Messages view
			if len(m.filteredDirectRooms) == 0 {
				if m.searchInput.Value() != "" {
					b.WriteString("No direct messages match your search.")
				} else {
					b.WriteString("No direct messages yet.")
					b.WriteString("\n\n")
					b.WriteString(helpStyle.Render("Start a conversation from the People tab!"))
				}
			} else {
				b.WriteString(fmt.Sprintf("Direct Messages (%d):\n\n", len(m.filteredDirectRooms)))
				
				// Show max 15 items for scrolling simulation
				startIdx := 0
				endIdx := len(m.filteredDirectRooms)
				if endIdx > 15 {
					// Simple viewport: show items around selection
					if m.dmIndex > 7 {
						startIdx = m.dmIndex - 7
					}
					endIdx = startIdx + 15
					if endIdx > len(m.filteredDirectRooms) {
						endIdx = len(m.filteredDirectRooms)
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
					dm := m.filteredDirectRooms[i]
					
					// Status indicator
					var statusIcon string
					if dm.OtherUser.IsOnline {
						statusIcon = onlineStyle.Render("●") // Filled dot
					} else {
						statusIcon = offlineStyle.Render("○") // Empty circle
					}
					
					// Format last message preview
					var lastMsgPreview string
					if dm.LastMessage != nil {
						// Truncate message content to fit
						maxLen := 40
						content := dm.LastMessage.Content
						if len(content) > maxLen {
							content = content[:maxLen] + "..."
						}
						lastMsgPreview = helpStyle.Render(content)
					} else {
						lastMsgPreview = helpStyle.Render("No messages yet")
					}
					
					// Format timestamp
					var timestamp string
					if dm.LastMessage != nil {
						duration := time.Since(dm.LastMessage.CreatedAt)
						if duration < time.Minute {
							timestamp = "just now"
						} else if duration < time.Hour {
							timestamp = fmt.Sprintf("%dm", int(duration.Minutes()))
						} else if duration < 24*time.Hour {
							timestamp = fmt.Sprintf("%dh", int(duration.Hours()))
						} else if duration < 7*24*time.Hour {
							timestamp = fmt.Sprintf("%dd", int(duration.Hours()/24))
						} else {
							timestamp = dm.LastMessage.CreatedAt.Format("Jan 2")
						}
					}
					
					// Build the DM line
					dmLine := fmt.Sprintf("%s %s", statusIcon, dm.OtherUser.Username)
					
					// Add unread badge if there are unread messages
					if dm.UnreadCount > 0 {
						dmLine += " " + errorStyle.Render(fmt.Sprintf("(%d)", dm.UnreadCount))
					}
					
					dmLine += "\n    " + lastMsgPreview
					
					if timestamp != "" {
						dmLine += " " + helpStyle.Render("· "+timestamp)
					}
					
					if i == m.dmIndex {
						b.WriteString(selectedItem.Render("> " + dmLine))
					} else {
						b.WriteString("  " + dmLine)
					}
					b.WriteString("\n")
				}
				if endIdx < len(m.filteredDirectRooms) {
					b.WriteString("  ↓ More items below\n")
				}
			}
		
		case lobbyViewPeople:
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
		// Show different help text based on current view
		if m.currentView == lobbyViewDirectMessages {
			b.WriteString(helpStyle.Render("Tab: switch view | ↑/↓: navigate | Enter: select | n: new DM | /: search | m/Esc: menu | q: quit"))
		} else {
			b.WriteString(helpStyle.Render("Tab: switch view | ↑/↓: navigate | Enter: select | /: search | m/Esc: menu | q: quit"))
		}

	case stateConversation:
		// Display header based on whether this is a DM or group room
		if m.currentDMUser != nil {
			// Direct message header - show other user's name and online status
			var statusIcon string
			if m.currentDMUser.IsOnline {
				statusIcon = onlineStyle.Render("●")
			} else {
				statusIcon = offlineStyle.Render("○")
			}
			b.WriteString(titleStyle.Render(m.currentDMUser.Username))
			b.WriteString(" ")
			b.WriteString(statusIcon)
			b.WriteString("\n\n")
		} else if m.currentRoom != nil {
			// Group room header - show room name
			b.WriteString(titleStyle.Render("# " + m.currentRoom.Name))
			b.WriteString(" ")
			b.WriteString(statusStyle.Render("- " + m.user.Username))
			b.WriteString("\n\n")
		}

		// Display messages viewport
		if m.viewportReady {
			b.WriteString(borderStyle.Render(m.messageViewport.View()))
			b.WriteString("\n\n")
		} else {
			if len(m.messages) == 0 {
				b.WriteString(statusStyle.Render("No messages yet. Be the first to say something!"))
				b.WriteString("\n\n")
			} else {
				b.WriteString(statusStyle.Render(fmt.Sprintf("Loaded %d messages", len(m.messages))))
				b.WriteString("\n\n")
			}
		}

		// Message input
		b.WriteString(m.messageInput.View())
		b.WriteString("\n\n")

		// Help text with edit/delete/reply options
		if m.selectedMessageIndex >= 0 {
			// Check if selected message belongs to the user
			selectedMsg := m.messages[m.selectedMessageIndex]
			if selectedMsg.UserID == m.user.ID {
				b.WriteString(helpStyle.Render("↑/↓: navigate | r: reply | e: edit | d: delete | Tab: type | Esc: back"))
			} else {
				b.WriteString(helpStyle.Render("↑/↓: navigate | r: reply | Tab: type | Esc: back"))
			}
		} else if m.replyingTo != nil {
			b.WriteString(helpStyle.Render("Replying mode | Enter: send reply | Esc: cancel reply | Tab: navigate"))
		} else if m.messageInput.Focused() {
			b.WriteString(helpStyle.Render("Enter: send | Esc: back"))
		} else {
			b.WriteString(helpStyle.Render("Tab: type message | ↑/↓: navigate | Esc: back"))
		}

	case stateEditingMessage:
		// Display header based on whether this is a DM or group room
		if m.currentDMUser != nil {
			var statusIcon string
			if m.currentDMUser.IsOnline {
				statusIcon = onlineStyle.Render("●")
			} else {
				statusIcon = offlineStyle.Render("○")
			}
			b.WriteString(titleStyle.Render(m.currentDMUser.Username))
			b.WriteString(" ")
			b.WriteString(statusIcon)
			b.WriteString(" ")
			b.WriteString(statusStyle.Render("- Editing Message"))
			b.WriteString("\n\n")
		} else if m.currentRoom != nil {
			b.WriteString(titleStyle.Render("# " + m.currentRoom.Name))
			b.WriteString(" ")
			b.WriteString(statusStyle.Render("- Editing Message"))
			b.WriteString("\n\n")
		}

		// Show the original message being edited
		if m.editingMessage != nil {
			b.WriteString(statusStyle.Render(fmt.Sprintf("Editing message from %s:", m.editingMessage.User.Username)))
			b.WriteString("\n")
			b.WriteString(normalItem.Render(fmt.Sprintf("Original: %s", m.editingMessage.Content)))
			b.WriteString("\n\n")
		}

		// Edit input
		b.WriteString("New content:\n")
		b.WriteString(m.editInput.View())
		b.WriteString("\n\n")

		// Help text
		b.WriteString(helpStyle.Render("Enter: save changes | Esc: cancel"))

	case stateConfirmDelete:
		// Display header based on whether this is a DM or group room
		if m.currentDMUser != nil {
			var statusIcon string
			if m.currentDMUser.IsOnline {
				statusIcon = onlineStyle.Render("●")
			} else {
				statusIcon = offlineStyle.Render("○")
			}
			b.WriteString(titleStyle.Render(m.currentDMUser.Username))
			b.WriteString(" ")
			b.WriteString(statusIcon)
			b.WriteString(" ")
			b.WriteString(statusStyle.Render("- Delete Message"))
			b.WriteString("\n\n")
		} else if m.currentRoom != nil {
			b.WriteString(titleStyle.Render("# " + m.currentRoom.Name))
			b.WriteString(" ")
			b.WriteString(statusStyle.Render("- Delete Message"))
			b.WriteString("\n\n")
		}

		// Show the message being deleted
		if m.editingMessage != nil {
			b.WriteString(errorStyle.Render("⚠ Are you sure you want to delete this message?"))
			b.WriteString("\n\n")
			b.WriteString(statusStyle.Render(fmt.Sprintf("From: %s", m.editingMessage.User.Username)))
			b.WriteString("\n")
			b.WriteString(normalItem.Render(fmt.Sprintf("Content: %s", m.editingMessage.Content)))
			b.WriteString("\n\n")
		}

		// Confirmation prompt
		b.WriteString(helpStyle.Render("Press 'y' to delete, 'n' or Esc to cancel"))

	case stateAdminRooms:
		b.WriteString(titleStyle.Render("Admin: Manage Rooms"))
		b.WriteString(" ")
		b.WriteString(statusStyle.Render("- " + m.user.Username))
		b.WriteString("\n\n")

		if len(m.rooms) == 0 {
			b.WriteString(statusStyle.Render("No rooms available."))
			b.WriteString("\n\n")
		} else {
			b.WriteString(fmt.Sprintf("Total rooms: %d\n\n", len(m.rooms)))
			// Show max 15 items for scrolling
			startIdx := 0
			endIdx := len(m.rooms)
			if endIdx > 15 {
				if m.adminRoomIndex > 7 {
					startIdx = m.adminRoomIndex - 7
				}
				endIdx = startIdx + 15
				if endIdx > len(m.rooms) {
					endIdx = len(m.rooms)
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
				room := m.rooms[i]
				if i == m.adminRoomIndex {
					b.WriteString(selectedItem.Render(fmt.Sprintf("> %s (ID: %d)", room.Name, room.ID)))
				} else {
					b.WriteString(normalItem.Render(fmt.Sprintf("  %s (ID: %d)", room.Name, room.ID)))
				}
				b.WriteString("\n")
			}
			if endIdx < len(m.rooms) {
				b.WriteString("  ↓ More items below\n")
			}
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/↓: navigate | c: create | e: edit | d: delete | Esc: back"))

	case stateCreateRoom:
		b.WriteString(titleStyle.Render("Admin: Create New Room"))
		b.WriteString("\n\n")
		b.WriteString("Enter room name:\n")
		b.WriteString(m.roomNameInput.View())
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Enter: create room | Esc: cancel"))

	case stateEditRoom:
		b.WriteString(titleStyle.Render("Admin: Edit Room"))
		b.WriteString("\n\n")
		b.WriteString(statusStyle.Render(fmt.Sprintf("Editing: %s (ID: %d)", m.selectedRoomName, m.selectedRoomID)))
		b.WriteString("\n\n")
		b.WriteString("New room name:\n")
		b.WriteString(m.roomNameInput.View())
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Enter: save changes | Esc: cancel"))

	case stateConfirmDeleteRoom:
		b.WriteString(titleStyle.Render("Admin: Delete Room"))
		b.WriteString("\n\n")
		b.WriteString(errorStyle.Render("⚠ Are you sure you want to delete this room?"))
		b.WriteString("\n\n")
		b.WriteString(statusStyle.Render(fmt.Sprintf("Room: %s (ID: %d)", m.selectedRoomName, m.selectedRoomID)))
		b.WriteString("\n")
		b.WriteString(normalItem.Render("⚠ This will soft-delete the room. All messages will be preserved but the room will be hidden."))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Press 'y' to delete, 'n' or Esc to cancel"))

	case stateThreadView:
		if m.threadParentMessage == nil {
			b.WriteString(errorStyle.Render("Error: No thread parent message"))
			break
		}

		b.WriteString(titleStyle.Render("Thread View"))
		b.WriteString(" ")
		b.WriteString(statusStyle.Render("- " + m.user.Username))
		b.WriteString("\n\n")

		var threadContent strings.Builder
		threadContent.WriteString(separatorStyle.Render("━━━ Parent Message ━━━"))
		threadContent.WriteString("\n")

		parentTimestamp := m.threadParentMessage.CreatedAt.Format("3:04 PM")
		parentUsername := m.threadParentMessage.User.Username
		threadContent.WriteString(helpStyle.Render(parentTimestamp))
		threadContent.WriteString(" ")
		threadContent.WriteString(selectedItem.Render(parentUsername))
		threadContent.WriteString(": ")
		threadContent.WriteString(m.threadParentMessage.Content)
		threadContent.WriteString("\n\n")

		threadContent.WriteString(separatorStyle.Render(fmt.Sprintf("━━━ Replies (%d) ━━━", len(m.threadMessages))))
		threadContent.WriteString("\n")

		if len(m.threadMessages) == 0 {
			threadContent.WriteString(helpStyle.Render("  No replies yet. Be the first to reply!"))
			threadContent.WriteString("\n")
		} else {
			for i, msg := range m.threadMessages {
				timestamp := msg.CreatedAt.Format("3:04 PM")
				username := msg.User.Username

				isSelected := i == m.threadMessageIndex && !m.threadInput.Focused()

				var messageLine strings.Builder
				messageLine.WriteString(helpStyle.Render(timestamp))
				messageLine.WriteString(" ")
				messageLine.WriteString(selectedItem.Render(username))
				messageLine.WriteString(": ")
				messageLine.WriteString(msg.Content)

				if isSelected {
					threadContent.WriteString(lipgloss.NewStyle().
						Foreground(lipgloss.Color("11")).
						Bold(true).
						Render("> " + messageLine.String()))
				} else {
					threadContent.WriteString("  " + messageLine.String())
				}
				threadContent.WriteString("\n")
			}
		}

		b.WriteString(borderStyle.Render(threadContent.String()))
		b.WriteString("\n\n")

		b.WriteString(m.threadInput.View())
		b.WriteString("\n\n")

		if m.threadInput.Focused() {
			b.WriteString(helpStyle.Render("Enter: send reply | Tab: navigate messages | Esc: exit thread"))
		} else if m.threadMessageIndex >= 0 && m.threadMessageIndex < len(m.threadMessages) {
			selectedMsg := m.threadMessages[m.threadMessageIndex]
			if selectedMsg.UserID == m.user.ID {
				b.WriteString(helpStyle.Render("↑/↓: navigate | e: edit | d: delete | Tab: type | Esc: exit thread"))
			} else {
				b.WriteString(helpStyle.Render("↑/↓: navigate | Tab: type | Esc: exit thread"))
			}
		} else {
			b.WriteString(helpStyle.Render("Tab: type reply | Esc: exit thread"))
		}

	case stateNewDMDialog:
		b.WriteString(titleStyle.Render("Start New Direct Message"))
		b.WriteString(" ")
		b.WriteString(statusStyle.Render("- " + m.user.Username))
		b.WriteString("\n\n")

		// Search input
		b.WriteString("Search users: ")
		b.WriteString(m.dmSearchInput.View())
		b.WriteString("\n\n")

		// User list
		if len(m.filteredAvailableUsers) == 0 {
			if m.dmSearchInput.Value() != "" {
				b.WriteString(statusStyle.Render("No users match your search."))
			} else if len(m.availableUsers) == 0 {
				b.WriteString(statusStyle.Render("Loading users..."))
			} else {
				b.WriteString(statusStyle.Render("No users available."))
			}
		} else {
			b.WriteString(fmt.Sprintf("Available users (%d):\n\n", len(m.filteredAvailableUsers)))

			// Show max 15 items for scrolling
			startIdx := 0
			endIdx := len(m.filteredAvailableUsers)
			if endIdx > 15 {
				if m.availableUserIndex > 7 {
					startIdx = m.availableUserIndex - 7
				}
				endIdx = startIdx + 15
				if endIdx > len(m.filteredAvailableUsers) {
					endIdx = len(m.filteredAvailableUsers)
					startIdx = endIdx - 15
					if startIdx < 0 {
						startIdx = 0
					}
				}
			}

			if startIdx > 0 {
				b.WriteString("  ↑ More users above\n")
			}

			for i := startIdx; i < endIdx; i++ {
				user := m.filteredAvailableUsers[i]

				// Status indicator
				var statusIcon string
				if user.IsOnline {
					statusIcon = onlineStyle.Render("●")
				} else {
					statusIcon = offlineStyle.Render("○")
				}

				// DM exists indicator
				var dmIndicator string
				if user.HasDM {
					dmIndicator = helpStyle.Render(" (existing DM)")
				}

				userLine := fmt.Sprintf("%s %s%s", statusIcon, user.Username, dmIndicator)

				if i == m.availableUserIndex {
					b.WriteString(selectedItem.Render("> " + userLine))
				} else {
					b.WriteString("  " + userLine)
				}
				b.WriteString("\n")
			}

			if endIdx < len(m.filteredAvailableUsers) {
				b.WriteString("  ↓ More users below\n")
			}
		}

		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/↓: navigate | Enter: select | Esc: cancel"))
	}

	return menuStyle.Render(b.String())
}
