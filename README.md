# WindGo Chat App
<p align="left">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go">
  <img src="https://img.shields.io/badge/Node.js-18+-339933?logo=node.js">
  <img src="https://img.shields.io/badge/PostgreSQL-Database-336791?logo=postgresql">
  <img src="https://img.shields.io/badge/JWT-Authentication-FFB300?logo=jsonwebtokens">
  <img src="https://img.shields.io/badge/Docker-Optional-2496ED?logo=docker">
</p>

A modern real-time chat application featuring authentication, user profiles, chat rooms, direct messaging, and user mentions.
Built with Go for the backend. The web frontend has moved to a separate repository, and this repo is moving toward a CLI-based chat client.

## Features

- **Authentication**: JWT-based auth with GitHub OAuth support
- **Real-time messaging**: WebSocket-powered instant message delivery
- **Chat rooms**: Group chat rooms with admin management
- **Direct messaging**: Private 1-on-1 conversations
- **User mentions**: @username mentions with real-time notifications
- **Message search**: Full-text search powered by Meilisearch with <100ms response times
- **Message threading**: Reply to specific messages
- **Message editing/deletion**: Edit or soft-delete your own messages
- **Unread tracking**: Track unread message counts per room

> Notice: Frontend moved to its own repo
>
> The web frontend lives at: https://github.com/wilfierd/wildgo-Fe
>
> You can continue using that FE. This repo now focuses on the backend and an upcoming CLI chat client.




---

## Getting Started



- Docker (optional, for containerized setup)

---

### Backend Setup

```bash
cd chat-backend-go
go mod tidy
go run main.go
```
The backend will start on `http://localhost:8080`.

#### Database Initialization

- The backend uses PostgreSQL for data storage and Meilisearch for message search.
- To initialize, run the SQL script:
  ```bash
  psql -U <username> -d <database> -f init.sql
  ```
- Update `config/database.go` with your DB credentials.

#### Environment Variables

Create a `.env` file in `chat-backend-go/` (see `.env.example`):

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=windgo_chat

# JWT
JWT_SECRET=change-me-in-production

# Search (optional - gracefully degrades if unavailable)
MEILISEARCH_HOST=http://localhost:7700
MEILISEARCH_API_KEY=
```

#### Docker Setup (Recommended)

To run backend, database, and search engine with Docker:
```bash
cd chat-backend-go
docker-compose up
```

This starts:
- PostgreSQL on port 5432
- Meilisearch on port 7700
- Adminer (DB admin UI) on port 8081

**Note**: The backend gracefully handles Meilisearch being unavailable - message operations continue normally, but search functionality returns a 503 error.
