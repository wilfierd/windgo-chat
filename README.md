# WindGo Chat App
<p align="left">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go">
  <img src="https://img.shields.io/badge/Node.js-18+-339933?logo=node.js">
  <img src="https://img.shields.io/badge/Next.js-React-000?logo=next.js">
  <img src="https://img.shields.io/badge/PostgreSQL-Database-336791?logo=postgresql">
  <img src="https://img.shields.io/badge/JWT-Authentication-FFB300?logo=jsonwebtokens">
  <img src="https://img.shields.io/badge/Docker-Optional-2496ED?logo=docker">
</p>

A modern real-time chat application featuring authentication, user profiles, and chat rooms.  
Built with Go for the backend and Next.js for the frontend.




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

### Frontend Setup

```bash
cd chat-frontend-next
npm install
npm run dev
```
The frontend will start on `http://localhost:3000`.

---

## Integration & Usage

- The frontend communicates with the backend API at `http://localhost:8080`.
- Authentication is handled via JWT (see `utils/jwt.go` and `middleware/auth.go`).
- Main pages:
  - `/chat` — chat room UI
  - `/login` — authentication
  - `/profile` — user profile
- Main components:
  - `LoginForm.tsx`, UI elements in `components/ui/`

---

## Contributing & Testing

- No errors or warnings found in the current codebase.
- To contribute, fork the repo and submit a PR.
- Add environment variables as needed for DB and JWT secrets.

---

## License

MIT


