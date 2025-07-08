# WindGo Chat App

A real-time chat application built with Go backend and Next.js frontend.

## Tech Stack

**Backend**
- Go
- Gin framework
- JWT authentication

**Frontend**
- Next.js 15
- TypeScript
- Tailwind CSS
- Radix UI
- Axios

## Project Structure

```
windgo-chat-app/
├── chat-backend-go/          # Go backend server
├── chat-frontend-next/       # Next.js frontend
└── README.md
```

## Getting Started

### Prerequisites
- Go 1.21+
- Node.js 18+
- npm

### Backend Setup

```bash
cd chat-backend-go
go mod tidy
go run main.go
```

The backend will start on `http://localhost:8080`.

### Frontend Setup

```bash
cd chat-frontend-next
npm install
npm run dev
```

The frontend will start on `http://localhost:3000`.

