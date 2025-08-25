#!/bin/bash

echo "🚀 WindGo Chat GitHub OAuth Setup & Test"
echo "========================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Navigate to backend directory
cd chat-backend-go

print_status "Installing backend dependencies..."
go mod tidy
if [ $? -eq 0 ]; then
    print_success "Backend dependencies installed"
else
    print_error "Failed to install backend dependencies"
    exit 1
fi

print_status "Building backend..."
go build -o windgo-chat-server .
if [ $? -eq 0 ]; then
    print_success "Backend built successfully"
else
    print_error "Failed to build backend"
    exit 1
fi

print_status "Starting database (if using Docker)..."
docker-compose up -d
if [ $? -eq 0 ]; then
    print_success "Database started"
else
    print_warning "Docker compose failed - make sure PostgreSQL is running"
fi

print_status "Waiting for database to be ready..."
sleep 3

print_status "Starting backend server..."
./windgo-chat-server &
SERVER_PID=$!
sleep 3

# Check if server is running
if curl -s http://localhost:8080/health > /dev/null; then
    print_success "Backend server is running"
else
    print_error "Backend server failed to start"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

# Navigate to CLI directory
cd ../ssh-auth-cli

print_status "Installing CLI dependencies..."
go mod tidy
if [ $? -eq 0 ]; then
    print_success "CLI dependencies installed"
else
    print_error "Failed to install CLI dependencies"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

print_status "Building CLI..."
go build -o ssh-auth-cli .
if [ $? -eq 0 ]; then
    print_success "CLI built successfully"
else
    print_error "Failed to build CLI"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

echo ""
echo "🎉 Setup Complete!"
echo "=================="
echo ""
echo "📋 Next Steps:"
echo "1. Make sure you have SSH keys in ~/.ssh/ (id_rsa, id_ed25519, or id_ecdsa)"
echo "2. Make sure your SSH public keys are added to your GitHub account"
echo "3. Test the SSH authentication:"
echo ""
echo "   cd ssh-auth-cli"
echo "   ./ssh-auth-cli YOUR_GITHUB_USERNAME"
echo ""
echo "📡 API Endpoints available:"
echo "   POST /auth/nonce              - Get authentication nonce"
echo "   POST /auth/login/ssh          - SSH-based GitHub login"
echo "   POST /auth/refresh            - Refresh access token"
echo "   POST /auth/logout             - Logout and revoke device"
echo "   GET  /me                      - Get current user info"
echo "   POST /devices/revoke/:deviceId - Revoke specific device"
echo ""
echo "🌐 Test the API:"
echo "   curl http://localhost:8080/health"
echo "   curl -X POST http://localhost:8080/auth/nonce"
echo ""
echo "🛑 To stop the server:"
echo "   kill $SERVER_PID"

# Keep script running
echo ""
print_status "Press Ctrl+C to stop the server and exit"
wait $SERVER_PID
