# Conversation Feature Implementation

## Overview
This document describes the implementation of Phase 1 and Phase 2 of the conversation view feature for the WindGo CLI chat application.

## What Was Implemented

### Phase 1: Basic Conversation View ✅

#### 1. API Client Updates (`cli/internal/api/client.go`)

**New Message Type:**
```go
type Message struct {
    ID        uint      `json:"id"`
    UserID    uint      `json:"user_id"`
    RoomID    uint      `json:"room_id"`
    Content   string    `json:"content"`
    User      User      `json:"user"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

**New API Methods:**
- `GetMessages(token string, roomID uint, page, limit int) ([]Message, error)`
  - Fetches messages for a specific room with pagination
  - Default: page 1, limit 50 (max 100)
  - Endpoint: `GET /api/v1/rooms/:roomId/messages`

- `SendMessage(token string, roomID uint, content string) (*Message, error)`
  - Sends a new message to a room
  - Endpoint: `POST /api/v1/messages`
  - Returns the created message with user data

#### 2. UI State Management (`cli/internal/ui/app.go`)

**New View State:**
- Added `stateConversation` - the active chat room view

**New Model Fields:**
```go
// Conversation state
currentRoom     *api.Room         // Currently active room
currentDMUser   *api.User         // For future DM support
messages        []api.Message     // Loaded messages
messageInput    textinput.Model   // Input field for typing messages
messageViewport viewport.Model    // Scrollable message display
lastMessageID   uint              // For deduplication
pollingActive   bool              // Controls polling lifecycle
lastPollTime    time.Time         // Rate limiting for polls
```

**New Message Types:**
- `messagesLoadedMsg` - Messages fetched successfully
- `messageSentMsg` - Message sent successfully
- `pollTickMsg` - Triggers periodic message refresh

#### 3. Message Display & Input

**Message Viewport:**
- Displays messages in reverse chronological order (newest at bottom)
- Shows timestamp (HH:MM format)
- Highlights username in color
- Distinguishes "You" from other users
- Auto-scrolls to bottom on new messages

**Message Input:**
- Text input with 1000 character limit
- Focused automatically when entering conversation
- Placeholder: "Type a message... (ESC to go back)"
- Width: 80 characters

### Phase 2: Real-time Updates ✅

#### 1. Background Polling

**Poll Implementation:**
```go
func pollMessagesCmd() tea.Cmd {
    return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
        return pollTickMsg(t)
    })
}
```

**Polling Logic:**
- Polls every 3 seconds when in conversation view
- Only polls if `pollingActive == true` and `currentRoom != nil`
- Rate limiting: minimum 2 seconds between actual API calls
- Automatically starts when entering a room
- Stops when leaving conversation (ESC or /back)

#### 2. Message Deduplication

**Deduplication Strategy:**
- Tracks `lastMessageID` to identify newest message
- When sending a message, checks if it already exists in the list before adding
- Prevents duplicate messages from appearing during polling

**Implementation:**
```go
// Check if message already exists
found := false
for _, existing := range m.messages {
    if existing.ID == msg.message.ID {
        found = true
        break
    }
}
if !found {
    m.messages = append([]api.Message{*msg.message}, m.messages...)
}
```

#### 3. Window Resize Handling

**Responsive Layout:**
- Listens to `tea.WindowSizeMsg`
- Dynamically resizes message viewport based on terminal size
- Message viewport: `Width-4` by `Height-8` (leaves room for UI chrome)
- Updates viewport content on resize

## User Flow

### Entering a Conversation

1. User is in Chat Lobby (Tab: Rooms view)
2. User navigates with ↑/↓ to select a room
3. User presses Enter
4. CLI transitions to `stateConversation`
5. Background tasks start:
   - Load initial messages via `loadMessagesCmd`
   - Start polling every 3 seconds via `pollMessagesCmd`
6. Message input is automatically focused

### Sending a Message

1. User types a message in the input field
2. User presses Enter
3. CLI checks if content starts with `/` (command)
   - If yes: handle as command
   - If no: send as regular message
4. `sendMessageCmd` is called
5. On success:
   - Input is cleared
   - New message is added to viewport (if not duplicate)
   - Viewport scrolls to bottom

### Receiving Messages (Polling)

1. Every 3 seconds, `pollTickMsg` is emitted
2. CLI checks if still in conversation state
3. If yes, calls `loadMessagesCmd` for the current room
4. New messages are fetched and displayed
5. Viewport is updated with new content
6. Next poll is scheduled

### Exiting a Conversation

1. User presses ESC or types `/back`
2. Polling stops (`pollingActive = false`)
3. State returns to `stateChatLobby`
4. Room and message data is cleared
5. Message input is blurred

## Keyboard Controls

### In Conversation View

| Key(s) | Action |
|--------|--------|
| Enter | Send message (or execute command if starts with /) |
| ESC | Return to Chat Lobby |
| ↑ / k | Scroll up one line |
| ↓ / j | Scroll down one line |
| PgUp | Scroll up one page |
| PgDown | Scroll down one page |
| Ctrl+C | Quit application |

## Command System

Commands are prefixed with `/` and handled by `handleCommand()`:

| Command | Status | Description |
|---------|--------|-------------|
| /vault | 🔜 Placeholder | Future feature for secure storage |
| /help | ✅ Working | Shows available commands |
| /back | ✅ Working | Returns to Chat Lobby |
| /quit | ✅ Working | Exits conversation (same as ESC) |

**Adding New Commands:**
```go
case "/mycommand":
    m.status = helpStyle.Render("Command output here")
    m.messageInput.SetValue("")
```

## Architecture Decisions

### Why Polling Instead of WebSocket?

**Current Implementation:**
- Simple REST API polling every 3 seconds
- Easy to implement and debug
- No persistent connection management
- Works with existing backend endpoints

**Future Migration Path:**
- Backend: Add WebSocket upgrade handler
- CLI: Replace polling with WS connection
- Benefits: Real-time updates, reduced server load

### Message Ordering

**Current:**
- Backend returns messages in DESC order (newest first)
- CLI reverses them for display (newest at bottom)
- This matches typical chat UX expectations

### Rate Limiting

**Why 2-second minimum between polls?**
- Prevents excessive API calls if ticks fire rapidly
- Balances responsiveness vs server load
- Can be adjusted based on user feedback

## Backend API Requirements

The CLI expects these endpoints to be available:

1. **Get Messages:**
   ```
   GET /api/v1/rooms/:roomId/messages?page=1&limit=50
   Authorization: Bearer <token>
   
   Response:
   {
     "messages": [
       {
         "id": 1,
         "user_id": 1,
         "room_id": 1,
         "content": "Hello!",
         "user": { "id": 1, "username": "john", ... },
         "created_at": "2025-09-30T12:00:00Z",
         "updated_at": "2025-09-30T12:00:00Z"
       }
     ]
   }
   ```

2. **Send Message:**
   ```
   POST /api/v1/messages
   Authorization: Bearer <token>
   Content-Type: application/json
   
   Body:
   {
     "room_id": 1,
     "content": "Hello world!"
   }
   
   Response:
   {
     "message": "Message sent successfully",
     "data": {
       "id": 123,
       "user_id": 1,
       "room_id": 1,
       "content": "Hello world!",
       "user": { ... },
       "created_at": "2025-09-30T12:00:00Z"
     }
   }
   ```

## Testing Guide

### Manual Testing Steps

1. **Start the Backend:**
   ```bash
   cd chat-backend-go
   go run main.go
   ```

2. **Start the CLI:**
   ```bash
   cd cli
   go run ./cmd/windgo
   ```

3. **Test Flow:**
   - Login with email/password or GitHub
   - Navigate to Chat Lobby
   - Select a room and press Enter
   - Type a message and press Enter
   - Open another CLI instance, login, join same room
   - Verify messages appear in both clients within ~3 seconds
   - Test scrolling with arrow keys
   - Test commands: `/help`, `/vault`, `/back`
   - Test ESC to return to lobby

### Edge Cases to Test

- [ ] Empty room (no messages)
- [ ] Long messages (approaching 1000 char limit)
- [ ] Rapid message sending
- [ ] Network interruption during polling
- [ ] Window resize while in conversation
- [ ] Multiple users sending simultaneously

## Known Limitations

1. **No Message History Pagination**
   - Currently loads only the latest 50 messages
   - No UI to load older messages
   - Future: Add "Load more" command or scroll-to-top trigger

2. **No Direct Messages**
   - Backend may not have DM support yet
   - CLI has placeholder fields (`currentDMUser`)
   - Shows "DM not yet implemented" message

3. **No Typing Indicators**
   - Users don't see when others are typing
   - Future: Needs backend WebSocket support

4. **No Read Receipts**
   - No indication of who has read messages
   - Future feature

5. **No Message Editing/Deletion**
   - Messages are immutable once sent
   - Future: Add `/edit` and `/delete` commands

## Future Enhancements

### Phase 3: Commands & Polish
- [ ] Implement `/vault` functionality
- [ ] Add message timestamps with date separators
- [ ] Show user avatars as ASCII art or emojis
- [ ] Add unread message indicators
- [ ] Implement message search
- [ ] Add emoji support with `:emoji:` syntax

### Phase 4: WebSocket Migration
- [ ] Backend: Add WebSocket endpoint `/api/v1/rooms/:id/ws`
- [ ] CLI: Replace polling with persistent WS connection
- [ ] Implement reconnection logic
- [ ] Add connection status indicator
- [ ] Real-time typing indicators
- [ ] Presence updates (user join/leave notifications)

### Phase 5: Advanced Features
- [ ] File attachments (text-based: paste code, URLs)
- [ ] Message reactions
- [ ] Thread replies
- [ ] Room creation/management from CLI
- [ ] User mentions with `@username`
- [ ] Notification sounds (system beep)

## Performance Considerations

### Memory Usage
- Messages are stored in `[]api.Message` slice
- With 50 messages × ~200 bytes each = ~10KB per room
- Multiple rooms: cleared when switching
- **Recommendation:** Add message limit or LRU cache

### Network Usage
- Poll every 3 seconds = 1200 requests/hour per user
- Each request: ~1-5KB depending on new messages
- **Optimization:** Implement long-polling or WebSockets

### CPU Usage
- Message viewport rendering on every poll
- Terminal UI updates are generally lightweight
- No noticeable impact in testing

## Troubleshooting

### Messages not appearing
1. Check backend is running: `curl http://localhost:8080/api/v1/rooms`
2. Verify token is valid: Check `.config/windgo/credentials.json`
3. Check backend logs for errors
4. Ensure polling is active: `pollingActive` should be `true`

### "Failed to load messages" error
1. Verify room exists in database
2. Check user has access to the room
3. Review backend authentication middleware
4. Check API endpoint path matches backend routes

### Duplicate messages appearing
1. Message deduplication relies on message IDs
2. Ensure backend returns consistent IDs
3. Check if `lastMessageID` is being updated correctly

### Input not working
1. Verify message input is focused (should happen automatically)
2. Check for key binding conflicts
3. Try pressing ESC and re-entering the room

## Code Locations

### API Client
- **File:** `cli/internal/api/client.go`
- **Lines:** 65-77 (Message type), 258-341 (API methods)

### UI Components
- **File:** `cli/internal/ui/app.go`
- **Constants:** Lines 21-31 (states)
- **Model:** Lines 98-141 (fields)
- **Commands:** Lines 331-357 (message commands)
- **Update Logic:** Lines 513-580 (message handlers)
- **View Rendering:** Lines 1221-1254 (conversation view)
- **Input Handling:** Lines 890-926 (conversation keys)

## Contributors

- Initial implementation: Phase 1 & 2
- Date: 2025-09-30
- Commit: (pending)

## Related Documents

- `AGENTS.md` - Repository guidelines
- `README.md` - Project setup and overview
- `chat-backend-go/handlers/message_handlers.go` - Backend message API