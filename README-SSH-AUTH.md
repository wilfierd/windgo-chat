# WindGo Chat - GitHub SSH Authentication

## 🎯 Overview

WindGo Chat now supports GitHub SSH authentication, allowing users to authenticate using their GitHub account and SSH keys without needing a browser. This implementation follows a secure, stateless architecture with device management.

## 🏗️ Architecture

### Authentication Flow

1. **Nonce Generation**: Client requests a cryptographic nonce from server
2. **SSH Signing**: Client signs the nonce using their private SSH key
3. **GitHub Verification**: Server fetches user's public keys from GitHub API
4. **Signature Verification**: Server verifies the signature against GitHub public keys
5. **Token Generation**: Server issues JWT access token and refresh token
6. **Device Registration**: Server registers the device for session management

### Components

- **Backend (Go + Fiber)**: REST API with WebSocket support
- **CLI Tool (Go)**: Command-line interface for SSH authentication
- **Frontend (Next.js)**: Web interface with cookie-based authentication

## 🔧 Setup

### Prerequisites

1. **PostgreSQL Database**: Running locally or via Docker
2. **SSH Keys**: Generated and added to your GitHub account
3. **Go 1.21+**: For building the applications

### Quick Start

```bash
# Clone and setup
git clone <your-repo>
cd windgo-chat-app

# Make setup script executable
chmod +x setup-and-test.sh

# Run setup and test
./setup-and-test.sh
```

### Manual Setup

#### 1. Backend Setup

```bash
cd chat-backend-go

# Install dependencies
go mod tidy

# Start database (if using Docker)
docker-compose up -d

# Build and run server
go build -o windgo-chat-server .
./windgo-chat-server
```

#### 2. CLI Setup

```bash
cd ssh-auth-cli

# Install dependencies
go mod tidy

# Build CLI
go build -o ssh-auth-cli .

# Test authentication
./ssh-auth-cli YOUR_GITHUB_USERNAME
```

## 🔐 Authentication Methods

### 1. SSH Authentication (Recommended)

```bash
# Using CLI
./ssh-auth-cli your-github-username

# Using API directly
curl -X POST http://localhost:8080/auth/nonce
# Sign the nonce with your SSH key
curl -X POST http://localhost:8080/auth/login/ssh \
  -H "Content-Type: application/json" \
  -d '{
    "github_user": "your-username",
    "signed": "base64-signature",
    "pub_fingerprint": "SHA256:...",
    "nonce_id": "nonce-id",
    "device_name": "My Device",
    "device_type": "cli"
  }'
```

### 2. Traditional Login (Fallback)

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password"
  }'
```

## 📡 API Endpoints

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/nonce` | Generate authentication nonce |
| POST | `/auth/login/ssh` | SSH-based GitHub authentication |
| POST | `/auth/login` | Traditional email/password login |
| POST | `/auth/register` | User registration |
| POST | `/auth/refresh` | Refresh access token |
| POST | `/auth/logout` | Logout and revoke device |

### User & Device Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/me` | Get current user info and devices |
| POST | `/devices/revoke/:deviceId` | Revoke specific device |

### Health & Status

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/` | API status |

## 🛡️ Security Features

### Authentication Security

- **SSH Signature Verification**: Uses cryptographically secure SSH signatures
- **Nonce-based Protection**: One-time nonces prevent replay attacks
- **GitHub Key Validation**: Verifies signatures against GitHub's public key API
- **Device Management**: Track and revoke specific devices

### API Security

- **JWT Tokens**: Stateless authentication with user and device context
- **Cookie Security**: HttpOnly, Secure, SameSite=Strict cookies
- **Rate Limiting**: Protection against brute force attacks
- **CORS**: Controlled cross-origin access

### Token Management

- **Access Tokens**: Short-lived (24 hours) JWT tokens
- **Refresh Tokens**: Longer-lived (7 days) for token rotation
- **Token Rotation**: Automatic refresh token rotation on use
- **Device Revocation**: Ability to revoke individual devices

## 🔧 Configuration

### Environment Variables

```bash
# Database Configuration
DATABASE_URL=postgres://user:password@localhost:5432/windgo_chat
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=windgo_chat

# JWT Configuration
JWT_SECRET=your-super-secure-jwt-secret-key

# Server Configuration
PORT=8080
```

### SSH Key Requirements

1. **Supported Key Types**: RSA, Ed25519, ECDSA
2. **Key Location**: `~/.ssh/id_rsa`, `~/.ssh/id_ed25519`, or `~/.ssh/id_ecdsa`
3. **GitHub Integration**: Public key must be added to your GitHub account
4. **SSH Agent**: Optional but recommended for key management

## 🧪 Testing

### Test SSH Authentication

```bash
# Using the CLI tool
cd ssh-auth-cli
./ssh-auth-cli your-github-username

# Manual API testing
curl -X POST http://localhost:8080/auth/nonce
```

### Test Traditional Authentication

```bash
# Register a user
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123"
  }'

# Login
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

### Test Protected Endpoints

```bash
# Get user info (using cookie or Authorization header)
curl -X GET http://localhost:8080/me \
  -H "Authorization: Bearer your-jwt-token"
```

## 🎨 Frontend Integration

The system supports both cookie-based and header-based authentication:

```javascript
// Using cookies (automatic)
fetch('/api/me', {
  credentials: 'include'
});

// Using Authorization header
fetch('/api/me', {
  headers: {
    'Authorization': `Bearer ${accessToken}`
  }
});
```

## 🔄 Device Management

Users can manage their authenticated devices:

```bash
# List devices
curl -X GET http://localhost:8080/me \
  -H "Authorization: Bearer your-token"

# Revoke a device
curl -X POST http://localhost:8080/devices/revoke/dev_abc123 \
  -H "Authorization: Bearer your-token"
```

## 🐛 Troubleshooting

### Common Issues

1. **SSH Key Not Found**
   - Ensure you have SSH keys in `~/.ssh/`
   - Supported names: `id_rsa`, `id_ed25519`, `id_ecdsa`

2. **GitHub API Errors**
   - Check that your public key is added to GitHub
   - Verify GitHub username is correct

3. **Database Connection**
   - Ensure PostgreSQL is running
   - Check database credentials in environment variables

4. **Signature Verification Failed**
   - Ensure your SSH key matches the one on GitHub
   - Try regenerating SSH keys if issues persist

### Debug Mode

Enable debug logging by setting environment variables:

```bash
export DEBUG=true
export LOG_LEVEL=debug
```

## 🚀 Production Deployment

### Security Checklist

- [ ] Set strong `JWT_SECRET` environment variable
- [ ] Use HTTPS in production
- [ ] Configure proper CORS origins
- [ ] Set up rate limiting
- [ ] Use secure database connections
- [ ] Enable logging and monitoring

### Docker Deployment

```bash
# Build and run with Docker
docker-compose up -d
```

## 📄 License

MIT License - see LICENSE file for details.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📞 Support

- **Issues**: Open an issue on GitHub
- **Documentation**: See `/docs` directory
- **Examples**: Check `/examples` directory
