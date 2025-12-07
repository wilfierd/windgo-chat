# WindGo Chat - TODO & Progress Tracker

## 📊 Project Status Overview

This document tracks the development progress of WindGo Chat, a real-time chat application with Go backend and terminal CLI client.

---

## ✅ Completed Features (Recent Work)

### 1. **Core Authentication System** ✓

- ✅ Email/password authentication (login & registration)
- ✅ JWT token-based authentication (24h expiration)
- ✅ GitHub OAuth (both web and device flow)
- ✅ Token refresh endpoint
- ✅ User profile endpoint
- ✅ Credential persistence in CLI (`~/.config/windgo/credentials.json`)

### 2. **Real-Time Messaging** ✓

- ✅ WebSocket infrastructure with Hub/Client architecture
- ✅ Real-time message broadcasting to room members
- ✅ Join/leave room events
- ✅ Typing indicators
- ✅ Online user tracking
- ✅ User activity tracking (last_active_at, is_online, status)
- ✅ Heartbeat ping/pong for connection health

### 3. **Message Management** ✓

- ✅ Create messages (POST)
- ✅ Edit messages (PUT) - users can edit their own messages
- ✅ Delete messages (DELETE) - soft delete, users can delete their own
- ✅ Message threading/replies with parent_id
- ✅ Get messages with pagination
- ✅ Parent message data preloaded in responses

### 4. **Room Management (Admin)** ✓

- ✅ Create rooms (POST /api/v1/rooms) - admin only
- ✅ Update rooms (PUT /api/v1/rooms/:id) - admin only
- ✅ Delete rooms (DELETE /api/v1/rooms/:id) - soft delete, admin only
- ✅ Get room by ID (GET /api/v1/rooms/:id) - public
- ✅ List all rooms (GET /api/v1/rooms) - public
- ✅ Room name uniqueness validation
- ✅ CLI admin UI for room management

### 5. **Terminal CLI Client** ✓

- ✅ Bubble Tea TUI framework implementation
- ✅ Login menu (email or GitHub device flow)
- ✅ Main menu navigation
- ✅ Chat lobby with rooms and people tabs
- ✅ Real-time message display with WebSocket
- ✅ Message selection and navigation (↑/↓, j/k)
- ✅ Message editing (e key)
- ✅ Message deletion with confirmation (d key)
- ✅ Message replies/threading (r key)
- ✅ Search/filter for rooms and users
- ✅ Online status indicators
- ✅ Admin room management UI
- ✅ Input handling fixes (no more text input mess)
- ✅ Navigation command improvements

### 6. **Database & Backend Architecture** ✓

- ✅ PostgreSQL with GORM ORM
- ✅ Auto-migration on startup
- ✅ Optimized indexes (users, messages, rooms)
- ✅ Connection pooling configuration
- ✅ Soft delete support (gorm.DeletedAt)
- ✅ Demo data seeding
- ✅ Activity tracking middleware

### 7. **Message Search (Meilisearch)** ✓

- ✅ Full-text search with Meilisearch integration
- ✅ Search endpoint: GET /api/v1/search with query, room_id filter, cursor pagination
- ✅ Navigation context endpoint: GET /api/v1/search/navigate/:messageId
- ✅ Automatic message indexing on create/update/delete
- ✅ Searchable fields: content, username, room_name
- ✅ Filterable by room_id, user_id; sortable by created_at
- ✅ Graceful degradation (503 if Meilisearch unavailable)

### 8. **Documentation** ✓

- ✅ CLAUDE.md - Comprehensive project guide
- ✅ MESSAGE_THREADING.md - Threading implementation docs
- ✅ MESSAGE_EDITING_DELETION.md - Edit/delete docs
- ✅ ROOM_MANAGEMENT.md - Room admin guide
- ✅ ROOM_MANAGEMENT_QUICKSTART.md - Quick reference
- ✅ UI_FLOW_VISUAL.md - Visual UI flow guide
- ✅ USER_MENTIONS.md - User mention system docs

### 9. **CI/CD & DevOps** ✓

- ✅ GitHub Actions workflow
- ✅ Discord notifications for builds
- ✅ Binary builds tracked in git

---

## 🚧 Known Issues & Bug Fixes Needed

### High Priority Bugs:

1. ✅ **Reply text input error** - FIXED: Added proper focus management in reply mode
2. ✅ **Type assertion safety** - FIXED: Added type-safe GetUserID helper with proper error handling
3. ✅ **Race condition** - FIXED: Added unique constraint to room.name with proper error handling

### Medium Priority:

4. ✅ **WebSocket reconnection** - FIXED: Implemented auto-reconnect with exponential backoff and room rejoin
5. ⚠️ **Error handling** - Some error messages could be more user-friendly in CLI
6. ⚠️ **Input validation** - Server-side room name length limits not enforced

---

## 🎯 Next Steps - Prioritized Roadmap

### **PHASE 1: Bug Fixes & Stability** (Immediate - Week 1-2)

#### Critical:

- [x] **Fix reply text input error** - Resolve remaining input handling issues ✓
- [x] **Add type safety** - Ensure userID type consistency (middleware → handlers) ✓
- [x] **Add DB unique constraint** - Add unique index on room.name to prevent race conditions ✓
- [x] **Add WebSocket auto-reconnect** - Handle connection drops gracefully in CLI ✓

#### Testing & Validation:

- [x] **Integration tests** - Test room CRUD with admin/non-admin users
- [x] **Manual QA** - Test all CLI flows end-to-end
- [x] **Verify soft-delete behavior** - Ensure deleted rooms/messages are hidden correctly
- [x] **Load testing** - Test WebSocket with multiple concurrent clients

#### Code Quality:

- [x] **Add recovery middleware** - Catch panics in handlers
- [x] **Improve error messages** - Better user-facing error messages
- [x] **Add input validation** - Server-side length limits and sanitization
- [x] **Add logging** - Structured logging for debugging (e.g., logrus or zap)

---

### **PHASE 2: Core Feature Enhancements** (Week 3-4)

#### High Priority Features:

- [x] **Direct messaging (DMs)** - Private 1-on-1 conversations ✓

  - ✅ New room type: "direct" vs "group"
  - ✅ Room membership table (many-to-many: users ↔ rooms)
  - ✅ API endpoints: POST /api/v1/rooms/direct, GET /api/v1/rooms/direct
  - ✅ CLI UI: "Direct Messages" tab in lobby
  - ✅ Spec created: `.kiro/specs/direct-messaging/` (requirements, design, tasks)
  - 📝 Implementation ready: 13 core tasks + 2 optional tasks defined

- [x] **Unread message tracking** ✓

  - ✅ New table: user_room_last_read (user_id, room_id, last_read_message_id, last_read_at)
  - ✅ Endpoint: POST /api/v1/rooms/:id/read (mark as read)
  - ✅ CLI: Show unread count badges on rooms

- [x] **Room membership & permissions** ✓
  - ✅ Room membership table (user_id, room_id, role: member/admin/owner)
  - ✅ Endpoint: POST /api/v1/rooms/:id/members (invite users)
  - ✅ Endpoint: DELETE /api/v1/rooms/:id/members/:userId (kick/leave)
  - ✅ Permission checks: Can user post to room? Can user see room?
  - ✅ Role-based access control: member/admin/owner hierarchy
  - ✅ Self-removal allowed, admins/owners can remove others with proper checks

#### Medium Priority:

- [ ] **Message reactions/emoji** - React to messages with emoji

  - New table: message_reactions (message_id, user_id, emoji, created_at)
  - Endpoint: POST /api/v1/messages/:id/reactions
  - WebSocket: Broadcast reaction events

- [x] **Rate limiting** - Prevent spam ✓

  - ✅ Implemented via nginx reverse proxy (`nginx.conf.example`)
  - ✅ Per-IP rate limits: 10 messages/minute, 60 requests/minute for general API
  - ✅ Auth endpoints: 5 requests/minute with burst of 3
  - ✅ Returns 429 Too Many Requests with JSON error response

- [x] **Cursor-based pagination** - Better pagination for active chats ✓
  - ✅ Cursor parameter: GET /api/v1/rooms/:id/messages?cursor=messageID&limit=50
  - ✅ Returns next_cursor and prev_cursor in response
  - ✅ Base64-encoded timestamp cursors for stable pagination

---

### **PHASE 3: Secure File Transfer & User Experience** (Week 5-6)

- [ ] **Secure file transfer** - Private file sharing for sensitive data (keys, configs, documents)

  - End-to-end encryption for file content
  - Direct peer-to-peer transfer when possible (WebRTC data channels)
  - Encrypted blob storage fallback (server-side encrypted at rest)
  - File metadata: name, size, hash (SHA-256), expiration
  - Transfer endpoints:
    - POST /api/v1/files/offer (sender initiates transfer)
    - GET /api/v1/files/:id/accept (receiver accepts)
    - WebSocket: file transfer events and progress
  - No persistent storage of decrypted files on server
  - Security features:
    - Password-protected file transfers (optional)
    - One-time download links (file deleted after retrieval)
    - Transfer expiration (24h default, configurable)
    - File size limits (100MB for CLI focus)

- [x] **User mentions (@username)** - Mention users in messages ✓

  - ✅ Parse @username in message content
  - ✅ Store mentions: message_mentions table (message_id, mentioned_user_id)
  - ✅ Endpoint: GET /api/v1/mentions (get messages mentioning me)
  - ✅ WebSocket notifications for mentioned users
  - ✅ Cascade delete mentions when message is deleted
  - 📝 See `docs/USER_MENTIONS.md` for documentation

- [x] **Message search** - Search message history ✓

  - ✅ Endpoint: GET /api/v1/search?q=query&room_id=1&cursor=...&limit=20
  - ✅ Powered by Meilisearch for <100ms response times
  - ✅ Navigation context: GET /api/v1/search/navigate/:messageId
  - ✅ Cursor-based pagination with next_cursor and has_more
  - ✅ Automatic indexing on create/update/delete
  - ✅ Graceful degradation when Meilisearch unavailable

- [ ] **Read receipts** - See who read messages

  - Track: user_message_reads (user_id, message_id, read_at)
  - Endpoint: GET /api/v1/messages/:id/reads

---

### **PHASE 4: Performance & Scalability** (Week 7-8)

- [ ] **Message caching** - Reduce DB load

  - Redis cache for recent messages (last 100 per room)
  - Cache invalidation on new/edit/delete
  - TTL: 1 hour

- [ ] **Database optimization**

  - Add composite indexes: (room_id, created_at), (user_id, room_id)
  - Query optimization: EXPLAIN ANALYZE slow queries
  - Consider partitioning messages table by date if >10M rows

- [ ] **WebSocket scaling** - Support horizontal scaling

  - Use Redis Pub/Sub for multi-instance message broadcasting
  - Sticky sessions or consistent hashing for WebSocket connections

- [ ] **API optimization**
  - Response compression (gzip middleware)
  - ETag/If-None-Match caching for static resources
  - GraphQL or gRPC for complex queries (optional)

---

### **PHASE 5: Advanced Features** (Week 9+)

- [ ] **Room categories/tags** - Organize rooms

  - Room.category field (e.g., "work", "social", "announcements")
  - Filter rooms by category

- [ ] **User profiles** - Enhanced user profiles

  - Bio, status message, timezone, public key (for secure file transfers)
  - Endpoint: GET /api/v1/users/:id, PUT /api/v1/users/me

- [ ] **Audit logging** - Track admin actions

  - Audit log table (user_id, action, resource_type, resource_id, timestamp)
  - Log: room create/update/delete, user ban/unban, etc.
  - Endpoint: GET /api/v1/admin/audit-log

- [ ] **Room restore/undelete** - Recover deleted rooms

  - Endpoint: POST /api/v1/admin/rooms/:id/restore

- [ ] **Bulk admin operations**

  - Bulk delete messages
  - Bulk invite users to room
  - Export room history to JSON/CSV

- [ ] **Advanced search** - Search with filters

  - Search by user, date range, room, has:link, has:file
  - Search syntax: from:@username after:2024-01-01

- [ ] **Message formatting** - Rich text support
  - Markdown rendering (bold, italic, code blocks, links)
  - Syntax highlighting for code blocks

---

## 🧪 Testing Strategy

### Unit Tests (Priority)

- [ ] Handler tests (mock DB with sqlmock or testify)
- [ ] API client tests (mock HTTP with httptest)
- [ ] WebSocket hub/client tests
- [ ] JWT utils tests
- [ ] Message threading validation tests

### Integration Tests

- [ ] End-to-end API tests (start test server, call endpoints)
- [ ] WebSocket integration tests (connect, join room, send/receive)
- [ ] Database integration tests (use test DB)



### Load Tests

- [ ] WebSocket load test (100+ concurrent connections)
- [ ] Message throughput test (messages/second)
- [ ] Database query performance tests

---

## 📦 Deployment & Infrastructure

### Containerization

- [ ] Dockerfile for backend
- [ ] Docker Compose for local dev (backend + postgres + redis)
- [ ] Multi-stage build for smaller images

### Production Readiness

- [ ] Environment-based config (dev/staging/prod)
- [ ] Health check endpoints (/health, /ready)
- [ ] Metrics/observability (Prometheus, Grafana)
- [ ] Graceful shutdown handling
- [ ] Database migration tool (golang-migrate)
- [ ] Secrets management (Vault, AWS Secrets Manager)



---

## 📝 Documentation Needs

- [ ] API documentation (Swagger/OpenAPI)
- [ ] User guide for CLI
- [ ] Admin guide
- [ ] Deployment guide
- [ ] Contributing guide
- [ ] Security best practices doc
- [ ] Troubleshooting guide

---

## 🔒 Security Enhancements

### Critical Security Features:

- [ ] **End-to-end encryption** - Encrypt messages before sending (critical for privacy-focused app)
- [ ] **Secure key exchange** - Implement secure key exchange protocol (ECDH)
- [ ] **File transfer encryption** - AES-256 encryption for files in transit
- [ ] **Zero-knowledge server** - Server cannot decrypt user content

### Standard Security:

- [x] Rate limiting (API & WebSocket) - ✅ Implemented via nginx
- [ ] Input sanitization (prevent injection attacks)
- [ ] SQL injection prevention audit (GORM should handle, but verify)
- [ ] CSRF protection for web OAuth flow
- [ ] Secrets rotation mechanism
- [ ] User ban/suspension system
- [ ] Report abuse system
- [ ] Message moderation/filtering (spam detection)

---

## 🎨 Future Ideas (Brainstorm)

- **Backend Features:**
  - Bots/integrations (webhooks, API integrations)
  - Message pinning
  - Scheduled messages
  - Export chat logs (encrypted backup)
- **Advanced Security:**
  - End-to-end encryption (Signal protocol)
  - Multi-server federation (Matrix-like)
  - Zero-knowledge architecture
  - Secure message self-destruct timer
  - Encrypted file vault (personal storage)
- **Developer Tools:**
  - Code snippet sharing with syntax highlighting
  - Git integration (share commits, diffs)

---

## 📊 Metrics to Track

- Daily/monthly active users
- Messages sent per day
- Average response time (API)
- WebSocket connection count
- Database query latency (p50, p95, p99)
- Error rate (4xx, 5xx)
- Uptime/availability

---

## 🏁 Current Sprint Focus (Based on Recent Commits)

### ✅ Completed in Last Sprint:

1. Room CRUD implementation (da6c3fa)
2. Navigation and command fixes (31b97d0)
3. Fixed navigation commands (12751a0)
4. Workflow updates (8a1de52, c610b03)
5. **Critical bug fixes completed:**
   - Reply text input focus management fixed
   - Type-safe userID context handling (middleware.GetUserID helper)
   - DB unique constraint on room.name (with error handling)
   - WebSocket auto-reconnect with exponential backoff

### 🎯 Next Sprint Goals (Week 1-2):

1. ✅ **Direct Messaging spec completed** - Ready for implementation
2. ✅ **Rate limiting** - Implemented via nginx reverse proxy
3. ✅ **Message search** - Meilisearch integration complete
4. ✅ **Cursor-based pagination** - Implemented for messages endpoint
5. Implement Direct Messaging feature (see `.kiro/specs/direct-messaging/tasks.md`)
6. Add comprehensive testing (unit + integration)
7. Improve error messages in CLI
8. Add server-side input validation

---

**Last Updated:** Based on commits through c610b03
**Project Phase:** Post-MVP, entering stability & feature enhancement phase
**Primary Goal:** Bug fixes → Testing → DM implementation → Performance optimization
