# WindGo Chat App
<p align="left">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go">
  <img src="https://img.shields.io/badge/Node.js-18+-339933?logo=node.js">
  <img src="https://img.shields.io/badge/PostgreSQL-Database-336791?logo=postgresql">
  <img src="https://img.shields.io/badge/JWT-Authentication-FFB300?logo=jsonwebtokens">
  <img src="https://img.shields.io/badge/Docker-Optional-2496ED?logo=docker">
</p>

A modern real-time chat application featuring authentication, user profiles, and chat rooms.  
Built with Go for the backend. The web frontend has moved to a separate repository, and this repo is moving toward a CLI-based chat client.

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

- The backend uses a SQL database.  
- To initialize, run the SQL script:
  ```bash
  psql -U <username> -d <database> -f init.sql
  ```
- Update `config/database.go` with your DB credentials.

#### Docker Setup (Optional)

To run backend and database with Docker:
```bash
docker-compose up
```

---

### Frontend (Moved)

The web frontend has moved to a separate repository: https://github.com/wilfierd/wildgo-Fe

- Use that repository if you prefer a browser UI. Follow its README for setup and commands.
- This repository will evolve toward a CLI chat client; the web UI remains available via `wildgo-Fe`.

---

## Configuration

Set environment variables (or copy `chat-backend-go/.env.example` to `chat-backend-go/.env` and adjust):

- PORT: server port (default `8080`).
- DATABASE_URL: full Postgres DSN; if empty, uses `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`.
- JWT_SECRET: secret key for signing JWTs.
- CORS_ORIGIN: allowed origin for CORS (default `http://localhost:3000`).

The backend listens on `http://localhost:<PORT>` and exposes REST APIs for auth, rooms, and messages.


## License

MIT

