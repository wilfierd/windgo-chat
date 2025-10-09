package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wilfierd/windgo-chat-app/cli/internal/api"
)

var (
	// Styles for the chat view
	chatTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	senderStyle = lipgloss.NewStyle().Bold(true)

	helpTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// chatModel holds the state for a single chat room view.
type chatModel struct {
	room         api.Room
	messages     []api.Message
	viewport     viewport.Model
	textarea     textarea.Model
	client       *api.Client
	token        string
	currentUser  *api.User
	err          error
	width        int
	height       int
	sending      bool
	viewportReady bool
}

func newChatModel(client *api.Client, token string, user *api.User, room api.Room) chatModel {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(50)  // Initial width
	ta.SetHeight(1) // Single line input

	// Remove the cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false

	vp := viewport.New(50, 10) // Initial size

	return chatModel{
		client:      client,
		token:       token,
		currentUser: user,
		room:        room,
		textarea:    ta,
		viewport:    vp,
	}
}

func (m *chatModel) SetSize(w, h int) {
	m.width = w
	m.height = h

	// Set viewport size, leaving space for title, input box, and status line
	m.viewport.Width = w
	m.viewport.Height = h - 8
	m.textarea.SetWidth(w - 3)
	m.viewportReady = true
}

func (m chatModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m chatModel) Update(msg tea.Msg) (chatModel, tea.Cmd) {
	var (
		taCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, taCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			// This signals to the main model to go back
			return m, func() tea.Msg { return backToLobbyMsg{} }
		case tea.KeyEnter:
			if !m.sending {
				content := strings.TrimSpace(m.textarea.Value())
				if content != "" {
					m.sending = true
					return m, sendMessageCmd(m.client, m.token, m.room.ID, content)
				}
			}
			m.textarea.SetValue("") // Clear input after sending
		}
	case messageSentMsg:
		m.sending = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Optimistically append the new message
			m.messages = append(m.messages, *msg.message)
			m.updateViewportContent()
		}
		m.textarea.SetValue("") // Clear input after sending
	case messagesLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Messages are loaded in reverse chronological order, so reverse them for display
			for i, j := 0, len(msg.messages)-1; i < j; i, j = i+1, j-1 {
				msg.messages[i], msg.messages[j] = msg.messages[j], msg.messages[i]
			}
			m.messages = msg.messages
			m.updateViewportContent()
		}
	}

	return m, tea.Batch(taCmd, vpCmd)
}

func (m *chatModel) updateViewportContent() {
	var content strings.Builder
	for _, msg := range m.messages {
		sender := msg.User.Username
		if msg.UserID == m.currentUser.ID {
			sender = "You"
		}

		// Style the sender's name
		sender = senderStyle.Render(sender)

		// Format timestamp
		timestamp := msg.CreatedAt.Format("3:04 PM")

		content.WriteString(fmt.Sprintf("%s [%s]\n", sender, timestamp))
		content.WriteString(fmt.Sprintf("%s\n\n", msg.Content))
	}
	m.viewport.SetContent(content.String())
	m.viewport.GotoBottom()
}

func (m chatModel) View() string {
	if !m.viewportReady {
		return "Initializing..."
	}

	var b strings.Builder

	// Title
	b.WriteString(chatTitleStyle.Render(fmt.Sprintf("Chatting in #%s", m.room.Name)))
	b.WriteString("\n\n")

	// Viewport for messages
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Text area for input
	b.WriteString(m.textarea.View())
	b.WriteString("\n")

	// Help text
	help := "Esc: back to lobby | Enter: send"
	if m.sending {
		help = "Sending..."
	}
	if m.err != nil {
		help = errorStyle.Render(fmt.Sprintf("Error: %s", m.err))
	}
	b.WriteString(helpTextStyle.Render(help))

	return b.String()
}

// Commands and Messages for the Chat Model

type backToLobbyMsg struct{}

type messagesLoadedMsg struct {
	messages []api.Message
	err      error
}

type messageSentMsg struct {
	message *api.Message
	err     error
}

func loadMessagesCmd(client *api.Client, token string, roomID uint) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.GetMessages(token, roomID, 1, 50)
		if err != nil {
			return messagesLoadedMsg{err: err}
		}
		return messagesLoadedMsg{messages: resp.Messages}
	}
}

func sendMessageCmd(client *api.Client, token string, roomID uint, content string) tea.Cmd {
	return func() tea.Msg {
		msg, err := client.SendMessage(token, roomID, content)
		if err != nil {
			return messageSentMsg{err: err}
		}
		return messageSentMsg{message: msg}
	}
}